// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcp

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/helper/jsonutil"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	b, reqStorage := getTestBackend(t)

	testConfigRead(t, b, reqStorage, nil)

	credJson, err := getTestCredentials()
	if err != nil {
		t.Fatal(err)
	}

	testConfigUpdate(t, b, reqStorage, map[string]interface{}{
		"credentials":           credJson,
		"service_account_email": "",
	})

	expected := map[string]interface{}{
		"ttl":                   int64(0),
		"max_ttl":               int64(0),
		"service_account_email": "",
	}

	testConfigRead(t, b, reqStorage, expected)
	testConfigUpdate(t, b, reqStorage, map[string]interface{}{
		"ttl": "50s",
	})

	expected["ttl"] = int64(50)
	testConfigRead(t, b, reqStorage, expected)
}

func testConfigUpdate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
}

func testConfigRead(t *testing.T, b logical.Backend, s logical.Storage, expected map[string]interface{}) {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp == nil && expected == nil {
		return
	}

	if resp.IsError() {
		t.Fatal(resp.Error())
	}

	if len(expected) != len(resp.Data) {
		t.Errorf("read data mismatch (expected %d values, got %d)", len(expected), len(resp.Data))
	}

	for k, expectedV := range expected {
		actualV, ok := resp.Data[k]

		if !ok {
			t.Errorf(`expected data["%s"] = %v but was not included in read output"`, k, expectedV)
		} else if expectedV != actualV {
			t.Errorf(`expected data["%s"] = %v, instead got %v"`, k, expectedV, actualV)
		}
	}

	if t.Failed() {
		t.FailNow()
	}
}

func getTestCredentials() ([]byte, error) {
	creds := map[string]interface{}{
		"client_email":   "testUser@google.com",
		"client_id":      "user123",
		"private_key_id": "privateKey123",
		"private_key":    "iAmAPrivateKey",
		"project_id":     "project123",
	}

	credJson, err := jsonutil.EncodeJSON(creds)
	if err != nil {
		return nil, err
	}

	return credJson, nil
}

type testSystemView struct {
	logical.StaticSystemView
}
