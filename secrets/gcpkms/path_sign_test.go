// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

func TestPathSign_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "sign/my-key")
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

				var digest []byte
				switch algo {
				case kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_4096_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_4096_SHA256,
					kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256:
					h := sha256.Sum256([]byte("hello world"))
					digest = h[:]
				case kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384:
					h := sha512.Sum384([]byte("hello world"))
					digest = h[:]
				}

				// Now sign it
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "sign/my-key",
					Data: map[string]interface{}{
						"digest":      base64.StdEncoding.EncodeToString(digest),
						"key_version": 1,
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				sigb64, ok := resp.Data["signature"]
				if !ok {
					t.Fatal("missing signature")
				}

				sig, err := base64.StdEncoding.DecodeString(sigb64.(string))
				if err != nil {
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

				// Verify the signature
				switch pk.Algorithm {
				case kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256:
					var parsedSig struct{ R, S *big.Int }
					if _, err := asn1.Unmarshal(sig, &parsedSig); err != nil {
						t.Errorf("failed to unmarshal signature: %s", err)
					}
					if !ecdsa.Verify(pub.(*ecdsa.PublicKey), digest, parsedSig.R, parsedSig.S) {
						t.Error("invalid signature")
					}
				case kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384:
					var parsedSig struct{ R, S *big.Int }
					if _, err := asn1.Unmarshal(sig, &parsedSig); err != nil {
						t.Errorf("failed to unmarshal signature: %s", err)
					}
					if !ecdsa.Verify(pub.(*ecdsa.PublicKey), digest, parsedSig.R, parsedSig.S) {
						t.Error("invalid signature")
					}
				case kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PSS_4096_SHA256:
					if err := rsa.VerifyPSS(pub.(*rsa.PublicKey), crypto.SHA256, digest, sig, &rsa.PSSOptions{}); err != nil {
						t.Errorf("invalid signature: %s", err)
					}
				case kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_3072_SHA256,
					kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_4096_SHA256:
					if err := rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.SHA256, digest, sig); err != nil {
						t.Errorf("invalid signature: %s", err)
					}
				default:
					t.Fatalf("unknown algorithm: %s", pk.Algorithm)
				}
			})
		}
	})
}
