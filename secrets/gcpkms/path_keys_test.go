// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathKeys_List(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ListOperation, "keys")
	})

	b, storage := testBackend(t)

	ctx := context.Background()
	if err := storage.Put(ctx, &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"foo"}`),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := b.HandleRequest(ctx, &logical.Request{
		Storage:   storage,
		Operation: logical.ListOperation,
		Path:      "keys",
	})
	if err != nil {
		t.Fatal(err)
	}

	if v, exp := resp.Data["keys"].([]string), []string{"my-key"}; !reflect.DeepEqual(v, exp) {
		t.Errorf("expected %q to be %q", v, exp)
	}
}

func TestPathKeys_Read(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ReadOperation, "keys/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/key-without-crypto-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"not-a-real-cryptokey"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		key  string
		err  bool
	}{
		{
			"key_not_exist",
			"not-a-real-key",
			true,
		},
		{
			"crypto_key_not_exist",
			"key-without-crypto-key",
			true,
		},
		{
			"success",
			"my-key",
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				ctx := context.Background()
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.ReadOperation,
					Path:      "keys/" + tc.key,
				})
				if err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				for _, v := range []string{
					"id",
					"primary_version",
					"purpose",
				} {
					if _, ok := resp.Data[v]; !ok {
						t.Errorf("missing %q", v)
					}
				}
			})
		}
	})
}

func TestPathKeys_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.CreateOperation, "keys/my-key")
		testFieldValidation(t, logical.UpdateOperation, "keys/my-key")
	})

	keyringNoExist := testKMSKeyRingName(t, "")
	defer testCleanupKeyRing(t, keyringNoExist)

	keyringExist, cleanup := testCreateKMSKeyRing(t, "")
	defer cleanup()

	cases := []struct {
		name string
		data map[string]interface{}
		err  bool
	}{
		{
			"key_ring_no_exist",
			map[string]interface{}{
				"key_ring":   keyringNoExist,
				"crypto_key": "my-crypto-key",
			},
			true,
		},
		{
			"key_ring_exist_crypto_key_no_exist",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "my-crypto-key",
			},
			true,
		},
		{
			"algorithm_symmetric_encryption",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "symmetric_encryption",
				"algorithm":  "symmetric_encryption",
				"purpose":    "encrypt_decrypt",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pss_2048_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pss_2048_sha256",
				"algorithm":  "rsa_sign_pss_2048_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pss_3072_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pss_3072_sha256",
				"algorithm":  "rsa_sign_pss_3072_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pss_4096_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pss_4096_sha256",
				"algorithm":  "rsa_sign_pss_4096_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pkcs1_2048_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pkcs1_2048_sha256",
				"algorithm":  "rsa_sign_pkcs1_2048_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pkcs1_3072_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pkcs1_3072_sha256",
				"algorithm":  "rsa_sign_pkcs1_3072_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_sign_pkcs1_4096_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_sign_pkcs1_4096_sha256",
				"algorithm":  "rsa_sign_pkcs1_4096_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_ec_sign_p256_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "ec_sign_p256_sha256",
				"algorithm":  "ec_sign_p256_sha256",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_ec_sign_p384_sha384",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "ec_sign_p384_sha384",
				"algorithm":  "ec_sign_p384_sha384",
				"purpose":    "asymmetric_sign",
			},
			false,
		},
		{
			"algorithm_rsa_decrypt_oaep_2048_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_decrypt_oaep_2048_sha256",
				"algorithm":  "rsa_decrypt_oaep_2048_sha256",
				"purpose":    "asymmetric_decrypt",
			},
			false,
		},
		{
			"algorithm_rsa_decrypt_oaep_3072_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_decrypt_oaep_3072_sha256",
				"algorithm":  "rsa_decrypt_oaep_3072_sha256",
				"purpose":    "asymmetric_decrypt",
			},
			false,
		},
		{
			"algorithm_rsa_decrypt_oaep_4096_sha256",
			map[string]interface{}{
				"key_ring":   keyringExist,
				"crypto_key": "rsa_decrypt_oaep_4096_sha256",
				"algorithm":  "rsa_decrypt_oaep_4096_sha256",
				"purpose":    "asymmetric_decrypt",
			},
			false,
		},
		{
			"protection_level_software",
			map[string]interface{}{
				"key_ring":         keyringExist,
				"crypto_key":       "software",
				"protection_level": "software",
			},
			false,
		},
		{
			"protection_level_hsm",
			map[string]interface{}{
				"key_ring":         keyringExist,
				"crypto_key":       "hsm",
				"protection_level": "hsm",
			},
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				b, storage := testBackend(t)

				ctx := context.Background()
				if _, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.CreateOperation,
					Path:      "keys/my-key",
					Data:      tc.data,
				}); err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				kmsClient := testKMSClient(t)
				cryptoKey := fmt.Sprintf("%s/cryptoKeys/%s", tc.data["key_ring"], tc.data["crypto_key"])
				ck, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{
					Name: cryptoKey,
				})
				if err != nil {
					t.Fatal(err)
				}

				if exp, ok := tc.data["purpose"]; ok {
					if v := strings.ToLower(ck.Purpose.String()); v != exp {
						t.Errorf("expected %q to be %q", v, exp)
					}
				}

				if exp, ok := tc.data["protection_level"]; ok {
					vt := ck.VersionTemplate
					if vt == nil {
						t.Errorf("missing version template")
					} else {
						if v := strings.ToLower(vt.ProtectionLevel.String()); v != exp {
							t.Errorf("expected %q to be %q", v, exp)
						}
					}
				}
			})
		}
	})
}

func TestPathKeys_Delete(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.DeleteOperation, "keys/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/key-without-crypto-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"not-a-real-cryptokey"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		key  string
		err  bool
	}{
		{
			"key_not_exist",
			"not-a-real-key",
			true,
		},
		{
			"crypto_key_not_exist",
			"key-without-crypto-key",
			true,
		},
		{
			"success",
			"my-key",
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				ctx := context.Background()
				if _, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.DeleteOperation,
					Path:      "keys/" + tc.key,
				}); err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				entry, err := storage.Get(ctx, "keys/"+tc.key)
				if err != nil {
					t.Fatal(err)
				}
				if entry != nil {
					t.Errorf("expected deletion: %#v", entry)
				}

				// If we didn't expect an error, check that the key has the right
				// attributes.
				if !tc.err {
					kmsClient := testKMSClient(t)
					ck, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{
						Name: cryptoKey,
					})
					if err != nil {
						t.Fatal(err)
					}

					if ck.NextRotationTime != nil {
						t.Errorf("expected %q to be nil", ck.NextRotationTime)
					}

					if ck.RotationSchedule != nil {
						t.Errorf("expected %q to be nil", ck.RotationSchedule)
					}
				}
			})
		}
	})
}
