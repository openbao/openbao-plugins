// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathEncrypt_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "encrypt/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	kmsClient := testKMSClient(t)

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-versioned-key",
		Value: []byte(`{"name":"my-versioned-key", "crypto_key_id":"` + cryptoKey + `", "min_version":3, "max_version":5}`),
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		aad  string
		pt   string
		err  bool
	}{
		{
			"encrypts",
			"",
			"hello world",
			false,
		},
		{
			"encrypts_aad",
			"yo yo yo",
			"hello world",
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
					Operation: logical.UpdateOperation,
					Path:      "encrypt/my-key",
					Data: map[string]interface{}{
						"additional_authenticated_data": tc.aad,
						"plaintext":                     tc.pt,
					},
				})
				if err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				ciphertext, ok := resp.Data["ciphertext"].(string)
				if !ok {
					t.Fatalf("expected ciphertext: %#v", resp)
				}

				ciphertextDec, err := base64.StdEncoding.DecodeString(ciphertext)
				if err != nil {
					t.Fatal(err)
				}

				// Decrypt the data
				decryptResp, err := kmsClient.Decrypt(ctx, &kmspb.DecryptRequest{
					Name:                        cryptoKey,
					Ciphertext:                  []byte(ciphertextDec),
					AdditionalAuthenticatedData: []byte(tc.aad),
				})
				if err != nil {
					t.Fatal(err)
				}

				if v, exp := string(decryptResp.Plaintext), tc.pt; v != exp {
					t.Errorf("expected %q to be %q", v, exp)
				}
			})
		}
	})

	t.Run("less_min_version", func(t *testing.T) {

		ctx := context.Background()
		_, err := b.HandleRequest(ctx, &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      "encrypt/my-versioned-key",
			Data: map[string]interface{}{
				"plaintext":   "hello world",
				"key_version": 2,
			},
		})
		if err != logical.ErrPermissionDenied {
			t.Errorf("expected %q to be %q", err, logical.ErrPermissionDenied)
		}
	})

	t.Run("greater_max_version", func(t *testing.T) {

		ctx := context.Background()
		_, err := b.HandleRequest(ctx, &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      "encrypt/my-versioned-key",
			Data: map[string]interface{}{
				"plaintext":   "hello world",
				"key_version": 7,
			},
		})
		if err != logical.ErrPermissionDenied {
			t.Errorf("expected %q to be %q", err, logical.ErrPermissionDenied)
		}
	})
}
