// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathVerify_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "verify/my-key")
	})

	t.Run("asymmetric", func(t *testing.T) {

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

				// Sign the digest
				message := "hello world"
				var digest []byte
				var dig *kmspb.Digest

				switch algo {
				case kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_4096_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_4096_SHA256,
					kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256:
					d := sha256.Sum256([]byte(message))
					digest = d[:]
					dig = &kmspb.Digest{
						Digest: &kmspb.Digest_Sha256{
							Sha256: digest,
						},
					}
				case kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384:
					d := sha512.Sum384([]byte(message))
					digest = d[:]
					dig = &kmspb.Digest{
						Digest: &kmspb.Digest_Sha384{
							Sha384: digest,
						},
					}
				default:
					t.Fatalf("unknown key signing algorithm: %s", algo)
				}

				ckv := cryptoKey + "/cryptoKeyVersions/1"
				kmsClient := testKMSClient(t)
				signResp, err := kmsClient.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{
					Name:   ckv,
					Digest: dig,
				})
				if err != nil {
					t.Fatal(err)
				}

				// Now verify it
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "verify/my-key",
					Data: map[string]interface{}{
						"digest":      base64.StdEncoding.EncodeToString(digest),
						"signature":   base64.StdEncoding.EncodeToString(signResp.Signature),
						"key_version": 1,
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				valid, ok := resp.Data["valid"]
				if !ok {
					t.Fatal("missing valid")
				}

				if b, ok := valid.(bool); !ok || !b {
					t.Errorf("expected valid %t to be %t", b, true)
				}
			})
		}
	})
}
