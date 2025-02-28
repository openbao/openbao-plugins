// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathPubkey_Read(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ReadOperation, "pubkey/my-key")
	})

	t.Run("asymmetric_decrypt", func(t *testing.T) {

		algorithms := []kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm{
			kmspb.CryptoKeyVersion_RSA_DECRYPT_OAEP_2048_SHA256,
			kmspb.CryptoKeyVersion_RSA_DECRYPT_OAEP_3072_SHA256,
			kmspb.CryptoKeyVersion_RSA_DECRYPT_OAEP_4096_SHA256,
		}

		for _, algo := range algorithms {
			algo := algo
			name := strings.ToLower(algo.String())

			t.Run(name, func(t *testing.T) {

				cryptoKey, cleanup := testCreateKMSCryptoKeyAsymmetricDecrypt(t, algo)
				defer cleanup()

				b, storage := testBackend(t)

				ctx := context.Background()
				if err := storage.Put(ctx, &logical.StorageEntry{
					Key:   "keys/my-key",
					Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
				}); err != nil {
					t.Fatal(err)
				}

				// Get the public key
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.ReadOperation,
					Path:      "pubkey/my-key",
					Data: map[string]interface{}{
						"key_version": 1,
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				// Verify it's a pem public key (this is kinda testing KMS, but it's
				// a good test to ensure the API doesn't change).
				pemStr, ok := resp.Data["pem"].(string)
				if !ok {
					t.Fatal("missing pem")
				}

				// Extract the PEM-encoded data block
				block, _ := pem.Decode([]byte(pemStr))
				if block == nil {
					t.Fatalf("not pem: %s", pemStr)
				}

				// Decode the public key
				if _, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	t.Run("asymmetric_sign", func(t *testing.T) {

		algorithms := []kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm{
			kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
			kmspb.CryptoKeyVersion_RSA_SIGN_PSS_3072_SHA256,
			kmspb.CryptoKeyVersion_RSA_SIGN_PSS_4096_SHA256,
			kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
			kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_3072_SHA256,
			kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_4096_SHA256,
			kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256,
			kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384,
		}

		for _, algo := range algorithms {
			algo := algo
			name := strings.ToLower(algo.String())

			t.Run(name, func(t *testing.T) {

				cryptoKey, cleanup := testCreateKMSCryptoKeyAsymmetricSign(t, algo)
				defer cleanup()

				b, storage := testBackend(t)

				ctx := context.Background()
				if err := storage.Put(ctx, &logical.StorageEntry{
					Key:   "keys/my-key",
					Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
				}); err != nil {
					t.Fatal(err)
				}

				// Get the public key
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.ReadOperation,
					Path:      "pubkey/my-key",
					Data: map[string]interface{}{
						"key_version": 1,
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				// Verify it's a pem public key (this is kinda testing KMS, but it's
				// a good test to ensure the API doesn't change).
				pemStr, ok := resp.Data["pem"].(string)
				if !ok {
					t.Fatal("missing pem")
				}

				// Extract the PEM-encoded data block
				block, _ := pem.Decode([]byte(pemStr))
				if block == nil {
					t.Fatalf("not pem: %s", pemStr)
				}

				// Decode the public key
				if _, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
}
