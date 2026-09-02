// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"reflect"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestPathKeysConfig_Read(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ReadOperation, "keys/config/my-key")
	})

	cases := []struct {
		name string
		data string
		exp  map[string]interface{}
		err  bool
	}{
		{
			"key_exist",
			`{"name":"my-key", "crypto_key_id":"example"}`,
			map[string]interface{}{
				"name":       "my-key",
				"crypto_key": "example",
			},
			false,
		},
		{
			"key_exist_parameters",
			`{"name":"my-key", "crypto_key_id":"example", "min_version":3, "max_version":5}`,
			map[string]interface{}{
				"name":        "my-key",
				"crypto_key":  "example",
				"min_version": 3,
				"max_version": 5,
			},
			false,
		},
		{
			"key_not_exist",
			"",
			nil,
			true,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				b, storage := testBackend(t)

				if tc.data != "" {
					if err := storage.Put(context.Background(), &logical.StorageEntry{
						Key:   "keys/my-key",
						Value: []byte(tc.data),
					}); err != nil {
						t.Fatal(err)
					}
				}

				ctx := context.Background()
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.ReadOperation,
					Path:      "keys/config/my-key",
				})
				if err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				if !reflect.DeepEqual(tc.exp, resp.Data) {
					t.Errorf("expected %#v to be %#v", tc.exp, resp.Data)
				}
			})
		}
	})
}

func TestPathKeysConfig_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ReadOperation, "keys/config/my-key")
	})

	cases := []struct {
		name string
		key  string
		data map[string]interface{}
		exp  *Key
		err  bool
	}{
		{
			"key_exists",
			"my-key",
			map[string]interface{}{
				"min_version": 50,
				"max_version": 100,
			},
			&Key{
				Name:       "my-key",
				MinVersion: 50,
				MaxVersion: 100,
			},
			false,
		},
		{
			"key_not_exists",
			"my-non-existent-key",
			nil,
			nil,
			true,
		},
		{
			"zero_min",
			"my-key",
			map[string]interface{}{
				"min_version": 0,
			},
			&Key{
				Name:       "my-key",
				MinVersion: 0,
			},
			false,
		},
		{
			"zero_max",
			"my-key",
			map[string]interface{}{
				"max_version": 0,
			},
			&Key{
				Name:       "my-key",
				MaxVersion: 0,
			},
			false,
		},
		{
			"negative_min",
			"my-key",
			map[string]interface{}{
				"min_version": -1,
			},
			&Key{
				Name:       "my-key",
				MinVersion: 0,
			},
			false,
		},
		{
			"negative_max",
			"my-key",
			map[string]interface{}{
				"max_version": -1,
			},
			&Key{
				Name:       "my-key",
				MaxVersion: 0,
			},
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				b, storage := testBackend(t)

				if err := storage.Put(context.Background(), &logical.StorageEntry{
					Key:   "keys/my-key",
					Value: []byte(`{"name":"my-key"}`),
				}); err != nil {
					t.Fatal(err)
				}

				_, err := b.HandleRequest(context.Background(), &logical.Request{
					Storage:   storage,
					Operation: logical.CreateOperation,
					Path:      "keys/config/" + tc.key,
					Data:      tc.data,
				})
				if err != nil {
					if tc.err {
						return
					}

					t.Fatal(err)
				}

				k, err := b.Key(context.Background(), storage, tc.key)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(tc.exp, k) {
					t.Errorf("expected %#v to equal %#v", tc.exp, k)
				}
			})
		}
	})

	t.Run("ignores_if_not_specified", func(t *testing.T) {

		b, storage := testBackend(t)

		if err := storage.Put(context.Background(), &logical.StorageEntry{
			Key:   "keys/my-key",
			Value: []byte(`{"name":"my-key", "min_version":3, "max_version":5}`),
		}); err != nil {
			t.Fatal(err)
		}

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      "keys/config/my-key",
			Data:      map[string]interface{}{},
		})
		if err != nil {
			t.Fatal(err)
		}

		k, err := b.Key(context.Background(), storage, "my-key")
		if err != nil {
			t.Fatal(err)
		}

		exp := &Key{
			Name:       "my-key",
			MinVersion: 3,
			MaxVersion: 5,
		}

		if !reflect.DeepEqual(exp, k) {
			t.Errorf("expected %#v to equal %#v", exp, k)
		}
	})
}
