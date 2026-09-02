// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"reflect"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestKey_Key(t *testing.T) {

	cases := []struct {
		name string
		c    []byte
		e    *Key
		err  bool
	}{
		{
			"default",
			nil,
			nil,
			true,
		},
		{
			"saved",
			[]byte(`{"name":"foo", "crypto_key_id":"bar"}`),
			&Key{
				Name:        "foo",
				CryptoKeyID: "bar",
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
					Key:   "keys/my-key",
					Value: tc.c,
				}); err != nil {
					t.Fatal(err)
				}
			}

			c, err := b.Key(context.Background(), storage, "my-key")
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c, tc.e) {
				t.Errorf("expected %#v to be %#v", c, tc.e)
			}
		})
	}
}

func TestKey_Keys(t *testing.T) {

	cases := []struct {
		name string
		c    []byte
		e    []string
		err  bool
	}{
		{
			"default",
			nil,
			nil,
			false,
		},
		{
			"saved",
			[]byte(`{"name":"foo", "crypto_key_id":"bar"}`),
			[]string{"my-key"},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {

			b, storage := testBackend(t)

			if tc.c != nil {
				if err := storage.Put(context.Background(), &logical.StorageEntry{
					Key:   "keys/my-key",
					Value: tc.c,
				}); err != nil {
					t.Fatal(err)
				}
			}

			c, err := b.Keys(context.Background(), storage)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c, tc.e) {
				t.Errorf("expected %#v to be %#v", c, tc.e)
			}
		})
	}
}
