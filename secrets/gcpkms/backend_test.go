// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/openbao/openbao/sdk/v2/logical"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/connectivity"

	kmsapi "cloud.google.com/go/kms/apiv1"
	hclog "github.com/hashicorp/go-hclog"
	uuid "github.com/satori/go.uuid"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// testBackend creates a new isolated instance of the backend for testing.
func testBackend(tb testing.TB) (*backend, logical.Storage) {
	tb.Helper()

	config := logical.TestBackendConfig()
	config.StorageView = new(logical.InmemStorage)
	config.Logger = hclog.NewNullLogger()

	b, err := Factory(context.Background(), config)
	if err != nil {
		tb.Fatal(err)
	}
	return b.(*backend), config.StorageView
}

// testFieldValidation verifies the given path has field validation.
func testFieldValidation(tb testing.TB, op logical.Operation, pth string) {
	tb.Helper()

	b, storage := testBackend(tb)
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Storage:   storage,
		Operation: op,
		Path:      pth,
		Data: map[string]interface{}{
			"literally-never-a-key": true,
		},
	})
	if err == nil {
		tb.Error("expected error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		tb.Error(err)
	}
}

// testKMSClient creates a new KMS client with the default scopes and user
// agent.
func testKMSClient(tb testing.TB) *kmsapi.KeyManagementClient {
	tb.Helper()

	ctx := context.Background()
	kmsClient, err := kmsapi.NewKeyManagementClient(ctx,
		option.WithScopes(defaultScope),
	)
	if err != nil {
		tb.Fatalf("failed to create kms client: %s", err)
	}

	return kmsClient
}

// testKMSKeyRingName creates a keyring name. If the given "name" is
// blank, a UUID name is generated.
func testKMSKeyRingName(tb testing.TB, name string) string {
	tb.Helper()

	if testing.Short() {
		tb.Skip("skipping integration test (short)")
	}

	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		tb.Fatal("missing GOOGLE_CLOUD_PROJECT")
	}

	if name == "" {
		name = fmt.Sprintf("vault-test-%s", uuid.NewV4())
	}

	return fmt.Sprintf("projects/%s/locations/us-east1/keyRings/%s", project, name)
}

// testCreateKMSKeyRing creates a keyring with the given name.
func testCreateKMSKeyRing(tb testing.TB, name string) (string, func()) {
	tb.Helper()

	keyRing := testKMSKeyRingName(tb, name)

	kmsClient := testKMSClient(tb)

	// Check if the key ring exists
	ctx := context.Background()
	kr, err := kmsClient.GetKeyRing(ctx, &kmspb.GetKeyRingRequest{
		Name: keyRing,
	})
	if err != nil {
		if terr, ok := grpcstatus.FromError(err); ok && terr.Code() == grpccodes.NotFound {
			// Key ring does not exist, try to create it
			kr, err = kmsClient.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{
				Parent:    path.Dir(path.Dir(keyRing)),
				KeyRingId: path.Base(keyRing),
			})
			if err != nil {
				tb.Fatalf("failed to create keyring: %s", err)
			}
		} else {
			tb.Fatalf("failed to get keyring: %s", err)
		}
	}

	return kr.Name, func() { testCleanupKeyRing(tb, kr.Name) }
}

// testCreateKMSCryptoKeySymmetric creates a new crypto key under the
// vault-gcpkms-plugin-test key ring in the given google project.
func testCreateKMSCryptoKeySymmetric(tb testing.TB) (string, func()) {
	return testCreateKMSCryptoKeyPurpose(tb,
		kmspb.CryptoKey_ENCRYPT_DECRYPT,
		kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION,
	)
}

// testCreateKMSCryptoKeyAsymmetricDecrypt creates a new KMS crypto key that is
// used for asymmetric decryption.
func testCreateKMSCryptoKeyAsymmetricDecrypt(tb testing.TB, algo kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) (string, func()) {
	return testCreateKMSCryptoKeyPurpose(tb,
		kmspb.CryptoKey_ASYMMETRIC_DECRYPT,
		algo,
	)
}

// testCreateKMSCryptoKeyAsymmetricSign creates a new KMS crypto key that is
// used for asymmetric signing.
func testCreateKMSCryptoKeyAsymmetricSign(tb testing.TB, algo kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) (string, func()) {
	return testCreateKMSCryptoKeyPurpose(tb,
		kmspb.CryptoKey_ASYMMETRIC_SIGN,
		algo,
	)
}

// testCreateKMSCryptoKeyPurpose is a lower-level testing helper to create a KMS
// crypto key with the given purpose.
func testCreateKMSCryptoKeyPurpose(tb testing.TB, purpose kmspb.CryptoKey_CryptoKeyPurpose, algo kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) (string, func()) {
	tb.Helper()

	if testing.Short() {
		tb.Skip("skipping integration test (short)")
	}

	kmsClient := testKMSClient(tb)

	keyRing, cleanup := testCreateKMSKeyRing(tb, "")
	keyName := fmt.Sprintf("%s", uuid.NewV4())

	ctx := context.Background()

	ck, err := kmsClient.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRing,
		CryptoKeyId: fmt.Sprintf("%s", keyName),
		CryptoKey: &kmspb.CryptoKey{
			Purpose: purpose,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: algo,
			},
		},
	})
	if err != nil {
		tb.Fatalf("failed to create crypto key: %s", err)
	}

	// Wait for the key to be ready.
	if err := retryFib(func() error {
		ckv, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{
			Name: ck.Name + "/cryptoKeyVersions/1",
		})
		if err != nil {
			return err
		}
		if ckv.State == kmspb.CryptoKeyVersion_ENABLED {
			return nil
		}
		return errors.New("key is not in ready state")
	}); err != nil {
		tb.Fatal("key did not enter ready state")
	}

	return ck.Name, cleanup
}

// testCleanupKeyRing deletes all key versions in the ring.
func testCleanupKeyRing(tb testing.TB, keyRing string) {
	tb.Helper()

	if testing.Short() {
		tb.Skip("skipping integration test (short)")
	}

	kmsClient := testKMSClient(tb)

	var ckvs []string

	ctx := context.Background()
	it := kmsClient.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: keyRing,
	})
	for {
		ck, err := it.Next()
		if err != nil {
			if err != iterator.Done {
				tb.Errorf("cleanup: failed to list crypto keys: %s %s", keyRing, err)
			}
			break
		}

		it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
			Parent: ck.Name,
		})
		for {
			ckv, err := it.Next()
			if err != nil {
				if err != iterator.Done {
					tb.Errorf("cleanup: failed to list crypto key versions: %s %s", ck.Name, err)
				}
				break
			}

			if ckv.State != kmspb.CryptoKeyVersion_DESTROYED &&
				ckv.State != kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
				ckvs = append(ckvs, ckv.Name)
			}
		}
	}

	wp := workerpool.New(25)
	for _, ckv := range ckvs {
		ckv := ckv

		wp.Submit(func() {
			if err := retryFib(func() error {
				if _, err := kmsClient.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{
					Name: ckv,
				}); err != nil {
					return err
				}
				return nil
			}); err != nil {
				tb.Errorf("cleanup: failed to destroy crypto key version %q: %s", ckv, err)
			}
		})
	}

	wp.StopWait()
}

func TestBackend_KMSClient(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test (short)")
	}

	t.Run("allows_concurrent_reads", func(t *testing.T) {

		b, storage := testBackend(t)

		_, closer1, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}
		defer closer1()

		doneCh := make(chan struct{})
		go func() {
			_, closer2, err := b.KMSClient(storage)
			if err != nil {
				t.Fatal(err)
			}
			defer closer2()
			close(doneCh)
		}()

		select {
		case <-doneCh:
		case <-time.After(1 * time.Second):
			t.Errorf("client was not available")
		}
	})

	t.Run("caches", func(t *testing.T) {

		b, storage := testBackend(t)

		client1, closer1, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}
		defer closer1()

		client2, closer2, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}
		defer closer2()

		// Note: not a bug; literally checking object equality
		if client1 != client2 {
			t.Errorf("expected %#v to be %#v", client1, client2)
		}
	})

	t.Run("expires", func(t *testing.T) {

		b, storage := testBackend(t)
		b.kmsClientLifetime = 50 * time.Millisecond

		client1, closer1, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}
		closer1()

		time.Sleep(100 * time.Millisecond)

		client2, closer2, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}
		closer2()

		if client1 == client2 {
			t.Errorf("expected %#v to not be %#v", client1, client2)
		}
	})
}

func TestBackend_ResetClient(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test (short)")
	}

	t.Run("closes_client", func(t *testing.T) {

		b, storage := testBackend(t)

		client, closer, err := b.KMSClient(storage)
		if err != nil {
			t.Fatal(err)
		}

		// Verify the client is "open"
		if client.Connection().GetState() == connectivity.Shutdown {
			t.Fatalf("connection is already stopped")
		}

		// Stop read lock
		closer()

		// Reset the clients
		b.ResetClient()

		// Verify the client closed
		if state := client.Connection().GetState(); state != connectivity.Shutdown {
			t.Errorf("expected client to be closed, was: %v", state)
		}
	})
}

func TestBackend_Config(t *testing.T) {

	cases := []struct {
		name string
		c    []byte
		e    *Config
		err  bool
	}{
		{
			"default",
			nil,
			DefaultConfig(),
			false,
		},
		{
			"saved",
			[]byte(`{"credentials":"foo", "scopes":["bar"]}`),
			&Config{
				Credentials: "foo",
				Scopes:      []string{"bar"},
			},
			false,
		},
		{
			"invalid",
			[]byte(`{x`),
			nil,
			true,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {

			b, storage := testBackend(t)

			if tc.c != nil {
				if err := storage.Put(context.Background(), &logical.StorageEntry{
					Key:   "config",
					Value: tc.c,
				}); err != nil {
					t.Fatal(err)
				}
			}

			c, err := b.Config(context.Background(), storage)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c, tc.e) {
				t.Errorf("expected %#v to be %#v", c, tc.e)
			}
		})
	}
}
