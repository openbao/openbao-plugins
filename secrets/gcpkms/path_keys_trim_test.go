// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
	"google.golang.org/api/iterator"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathKeysTrim_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.CreateOperation, "keys/trim/my-key")
		testFieldValidation(t, logical.UpdateOperation, "keys/trim/my-key")
		testFieldValidation(t, logical.DeleteOperation, "keys/trim/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	kmsClient := testKMSClient(t)

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-versioned-key",
		Value: []byte(`{"name":"my-versioned-key", "crypto_key_id":"` + cryptoKey + `", "min_version":3, "max_version":5}`),
	}); err != nil {
		t.Fatal(err)
	}

	for i := 2; i <= 5; i++ {
		ctx := context.Background()
		if _, err := kmsClient.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{
			Parent: cryptoKey,
			CryptoKeyVersion: &kmspb.CryptoKeyVersion{
				State: kmspb.CryptoKeyVersion_ENABLED,
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	if _, err := b.HandleRequest(ctx, &logical.Request{
		Storage:   storage,
		Operation: logical.UpdateOperation,
		Path:      "keys/trim/my-versioned-key",
	}); err != nil {
		t.Fatal(err)
	}

	var ckvs []string
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: cryptoKey,
	})
	for {
		resp, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			t.Fatal(err)
		}

		if resp.State != kmspb.CryptoKeyVersion_DESTROYED &&
			resp.State != kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
			ckvs = append(ckvs, resp.Name)
		}
	}

	if len(ckvs) > 3 {
		t.Errorf("expected less crypto key versions: %q", ckvs)
	}
}
