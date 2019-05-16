// This script is used to iterate over all keys in the project and destroy them.
// The tests do a good job of cleaning up, but during panics and through the
// course of use, a stray key or key version may exist.
//
// This cleans them.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gammazero/workerpool"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/protobuf/field_mask"

	kmsapi "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

var (
	mu sync.Mutex
	wg sync.WaitGroup

	kmsClient, _ = kmsapi.NewKeyManagementClient(context.Background())
	project      = os.Getenv("GOOGLE_CLOUD_PROJECT")
)

func main() {
	if project == "" {
		log.Fatalf("missing GOOGLE_CLOUD_PROJECT")
	}

	var allCkvs []string

	locations := []string{
		"global",
		"asia-east1",
		"asia-northeast1",
		"asia-south1",
		"asia-southeast1",
		"australia-southeast1",
		"europe-north1",
		"europe-west1",
		"europe-west2",
		"europe-west3",
		"europe-west4",
		"northamerica-northeast1",
		"southamerica-east1",
		"us-central1",
		"us-east1",
		"us-east4",
		"us-west1",
		"us-west2",
	}

	krwp := workerpool.New(100)
	ckwp := workerpool.New(250)
	ckvwp := workerpool.New(1000)
	for _, loc := range locations {
		parent := fmt.Sprintf("projects/%s/locations/%s", project, loc)

		krwp.Submit(func() {
			krs := listKRs(parent)

			for _, kr := range krs {
				kr := kr

				ckwp.Submit(func() {
					cks := listCKs(kr)

					for _, ck := range cks {
						ck := ck

						disableCKRotation(ck)

						ckvwp.Submit(func() {
							ckvs := listCKVs(ck)

							mu.Lock()
							allCkvs = append(allCkvs, ckvs...)
							mu.Unlock()
						})
					}
				})
			}
		})
	}

	krwp.StopWait()
	ckwp.StopWait()
	ckvwp.StopWait()

	dwp := workerpool.New(100)
	for _, ckv := range allCkvs {
		ckv := ckv

		dwp.Submit(func() {
			log.Printf("cleaning orphaned crypto key version: %s", ckv)

			ctx := context.Background()
			if _, err := kmsClient.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{
				Name: ckv,
			}); err != nil {
				log.Printf("cleanup: failed to destroy crypto key version %q: %s", ckv, err)
			}
		})
	}

	dwp.StopWait()
}

func listCKVs(parent string) []string {
	var ckvs []string

	ctx := context.Background()
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: parent,
	})
	for {
		ckv, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			log.Printf("cleanup: failed to list crypto key versions: %s %s", parent, err)
		}

		if ckv.State != kmspb.CryptoKeyVersion_DESTROYED &&
			ckv.State != kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
			ckvs = append(ckvs, ckv.Name)
		}
	}

	return ckvs
}

func listCKs(parent string) []string {
	var cks []string

	rwp := workerpool.New(100)

	ctx := context.Background()
	it := kmsClient.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: parent,
	})
	for {
		ck, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			log.Printf("cleanup: failed to list crypto keys: %s %s", parent, err)
		}

		rwp.Submit(func() {
			if ck.Purpose == kmspb.CryptoKey_ENCRYPT_DECRYPT {
				ctx := context.Background()
				if _, err := kmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Name:             ck.Name,
						NextRotationTime: nil,
						RotationSchedule: nil,
					},
					UpdateMask: &field_mask.FieldMask{
						Paths: []string{
							"next_rotation_time",
							"rotation_period",
						},
					},
				}); err != nil {
					log.Printf("cleanup: failed to disable rotation on crypto key: %s, %s", ck.Name, err)
				}
			}
		})

		cks = append(cks, ck.Name)
	}

	rwp.StopWait()

	return cks
}

func disableCKRotation(ck string) {
	ctx := context.Background()

	if _, err := kmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
		CryptoKey: &kmspb.CryptoKey{
			Name:             ck,
			NextRotationTime: nil,
			RotationSchedule: nil,
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"next_rotation_time", "rotation_period"},
		},
	}); err != nil {
		log.Printf("cleanup: failed to disable rotation on crypto key: %s, %s", ck, err)
	}
}

func listKRs(parent string) []string {
	var krs []string

	ctx := context.Background()
	it := kmsClient.ListKeyRings(ctx, &kmspb.ListKeyRingsRequest{
		Parent: parent,
	})
	for {
		kr, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			log.Printf("cleanup: failed to list key rings: %s", err)
		}

		if strings.HasPrefix(kr.Name, "vault-test-") {
			krs = append(krs, kr.Name)
		}
	}

	return krs
}
