// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"reflect"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestBackend_PathConfigRead(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.ReadOperation, "config")
	})

	t.Run("not_exist", func(t *testing.T) {

		b, storage := testBackend(t)
		ctx := context.Background()
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      "config",
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := resp.Data["scopes"]; !ok {
			t.Errorf("expected %q to include %q", resp.Data, "scopes")
		}
	})

	t.Run("exist", func(t *testing.T) {

		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON("config", &Config{
			Scopes:      []string{"foo"},
			Credentials: "creds",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := storage.Put(context.Background(), entry); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      "config",
		})
		if err != nil {
			t.Fatal(err)
		}

		if v, exp := resp.Data["scopes"].([]string), []string{"foo"}; !reflect.DeepEqual(v, exp) {
			t.Errorf("expected %q to be %q", v, exp)
		}

		if _, ok := resp.Data["credentials"]; ok {
			t.Errorf("should not return credentials")
		}
	})
}

func TestBackend_PathConfigUpdate(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.UpdateOperation, "config")
	})

	t.Run("not_exist", func(t *testing.T) {

		b, storage := testBackend(t)
		if _, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      "config",
			Data: map[string]interface{}{
				"scopes":      "foo,bar",
				"credentials": "creds",
			},
		}); err != nil {
			t.Fatal(err)
		}

		config, err := b.Config(context.Background(), storage)
		if err != nil {
			t.Fatal(err)
		}

		if v, exp := config.Credentials, "creds"; v != exp {
			t.Errorf("expected %q to be %q", v, exp)
		}

		if v, exp := config.Scopes, []string{"bar", "foo"}; !reflect.DeepEqual(v, exp) {
			t.Errorf("expected %q to be %q", v, exp)
		}
	})

	t.Run("exist", func(t *testing.T) {

		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON("config", &Config{
			Scopes:      []string{"foo"},
			Credentials: "creds",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := storage.Put(context.Background(), entry); err != nil {
			t.Fatal(err)
		}

		if _, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      "config",
			Data: map[string]interface{}{
				"scopes":      "foo,bar",
				"credentials": "new-creds",
			},
		}); err != nil {
			t.Fatal(err)
		}

		config, err := b.Config(context.Background(), storage)
		if err != nil {
			t.Fatal(err)
		}

		if v, exp := config.Credentials, "new-creds"; v != exp {
			t.Errorf("expected %q to be %q", v, exp)
		}

		if v, exp := config.Scopes, []string{"bar", "foo"}; !reflect.DeepEqual(v, exp) {
			t.Errorf("expected %q to be %q", v, exp)
		}
	})
}

func TestBackend_PathConfigDelete(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {

		testFieldValidation(t, logical.DeleteOperation, "config")
	})

	t.Run("not_exist", func(t *testing.T) {

		b, storage := testBackend(t)
		if _, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.DeleteOperation,
			Path:      "config",
		}); err != nil {
			t.Fatal(err)
		}

		config, err := b.Config(context.Background(), storage)
		if err != nil {
			t.Fatal(err)
		}

		if def := DefaultConfig(); !reflect.DeepEqual(config, def) {
			t.Errorf("expected %v to be %v", config, def)
		}
	})

	t.Run("exist", func(t *testing.T) {

		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON("config", &Config{
			Scopes:      []string{"foo"},
			Credentials: "creds",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := storage.Put(context.Background(), entry); err != nil {
			t.Fatal(err)
		}

		if _, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.DeleteOperation,
			Path:      "config",
		}); err != nil {
			t.Fatal(err)
		}

		config, err := b.Config(context.Background(), storage)
		if err != nil {
			t.Fatal(err)
		}

		if def := DefaultConfig(); !reflect.DeepEqual(config, def) {
			t.Errorf("expected %v to be %v", config, def)
		}
	})
}
