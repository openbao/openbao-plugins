// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathDecrypt_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "decrypt/my-key")
	})

	t.Run("asymmetric", func(t *testing.T) {

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

				ckv := cryptoKey + "/cryptoKeyVersions/1"

				// Get the public key
				kmsClient := testKMSClient(t)
				pk, err := kmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
					Name: ckv,
				})
				if err != nil {
					t.Fatal(err)
				}

				// Extract the PEM-encoded data block
				block, _ := pem.Decode([]byte(pk.Pem))
				if block == nil {
					t.Fatalf("not pem: %s", pk.Pem)
				}

				// Decode the public key
				pub, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					t.Fatal(err)
				}

				// Encrypt with the public key
				exp := "hello world"
				enc, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub.(*rsa.PublicKey), []byte(exp), nil)
				if err != nil {
					t.Fatal(err)
				}

				// Now decrypt it
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "decrypt/my-key",
					Data: map[string]interface{}{
						"ciphertext":  base64.StdEncoding.EncodeToString(enc),
						"key_version": 1,
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				if v := resp.Data["plaintext"]; v != exp {
					t.Errorf("expected %q to be %q", v, exp)
				}
			})
		}
	})

	t.Run("symmetric", func(t *testing.T) {

		cases := []struct {
			name string
			aad  string
			exp  string
		}{
			{
				"decrypts",
				"",
				"hello world",
			},
			{
				"decrypts_aad",
				"yo yo yo",
				"hello world",
			},
		}

		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
				defer cleanup()

				b, storage := testBackend(t)

				ctx := context.Background()
				if err := storage.Put(ctx, &logical.StorageEntry{
					Key:   "keys/my-key",
					Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
				}); err != nil {
					t.Fatal(err)
				}

				// Encrypt the data
				kmsClient := testKMSClient(t)
				encryptResp, err := kmsClient.Encrypt(ctx, &kmspb.EncryptRequest{
					Name:                        cryptoKey,
					Plaintext:                   []byte(tc.exp),
					AdditionalAuthenticatedData: []byte(tc.aad),
				})
				if err != nil {
					t.Fatal(err)
				}

				// Now decrypt it
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "decrypt/my-key",
					Data: map[string]interface{}{
						"additional_authenticated_data": tc.aad,
						"ciphertext":                    base64.StdEncoding.EncodeToString(encryptResp.Ciphertext),
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				if v, exp := resp.Data["plaintext"], tc.exp; v != exp {
					t.Errorf("expected %q to be %q", v, exp)
				}
			})
		}
	})
}
