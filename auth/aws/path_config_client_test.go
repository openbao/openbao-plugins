// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package aws

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestBackend_pathConfigClient(t *testing.T) {
	config := logical.TestBackendConfig()
	storage := &logical.InmemStorage{}
	config.StorageView = storage

	b, err := Backend(config)
	if err != nil {
		t.Fatal(err)
	}

	err = b.Setup(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	// make sure we start with empty roles, which gives us confidence that the read later
	// actually is the two roles we created
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/client",
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	// at this point, resp == nil is valid as no client config exists
	// if resp != nil, then resp.Data must have EndPoint and IAMServerIdHeaderValue as nil
	if resp != nil {
		if resp.IsError() {
			t.Fatalf("failed to read client config entry")
		} else if resp.Data["endpoint"] != nil || resp.Data["iam_server_id_header_value"] != nil {
			t.Fatalf("returned endpoint or iam_server_id_header_value non-nil")
		}
	}

	data := map[string]interface{}{
		"sts_endpoint":               "https://my-custom-sts-endpoint.example.com",
		"sts_region":                 "us-east-2",
		"iam_server_id_header_value": "vault_server_identification_314159",
	}
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config/client",
		Data:      data,
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.IsError() {
		t.Fatal("failed to create the client config entry")
	}

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/client",
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.IsError() {
		t.Fatal("failed to read the client config entry")
	}
	if resp.Data["iam_server_id_header_value"] != data["iam_server_id_header_value"] {
		t.Fatalf("expected iam_server_id_header_value: '%#v'; returned iam_server_id_header_value: '%#v'",
			data["iam_server_id_header_value"], resp.Data["iam_server_id_header_value"])
	}
	if resp.Data["sts_endpoint"] != data["sts_endpoint"] {
		t.Fatalf("expected sts_endpoint: '%#v'; returned sts_endpoint: '%#v'",
			data["sts_endpoint"], resp.Data["sts_endpoint"])
	}
	if resp.Data["sts_region"] != data["sts_region"] {
		t.Fatalf("expected sts_region: '%#v'; returned sts_region: '%#v'",
			data["sts_region"], resp.Data["sts_region"])
	}

	data = map[string]interface{}{
		"sts_endpoint":               "https://my-custom-sts-endpoint2.example.com",
		"sts_region":                 "us-west-1",
		"iam_server_id_header_value": "vault_server_identification_2718281",
	}
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/client",
		Data:      data,
		Storage:   storage,
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.IsError() {
		t.Fatal("failed to update the client config entry")
	}

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/client",
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.IsError() {
		t.Fatal("failed to read the client config entry")
	}
	if resp.Data["iam_server_id_header_value"] != data["iam_server_id_header_value"] {
		t.Fatalf("expected iam_server_id_header_value: '%#v'; returned iam_server_id_header_value: '%#v'",
			data["iam_server_id_header_value"], resp.Data["iam_server_id_header_value"])
	}
	if resp.Data["sts_endpoint"] != data["sts_endpoint"] {
		t.Fatalf("expected sts_endpoint: '%#v'; returned sts_endpoint: '%#v'",
			data["sts_endpoint"], resp.Data["sts_endpoint"])
	}
	if resp.Data["sts_region"] != data["sts_region"] {
		t.Fatalf("expected sts_region: '%#v'; returned sts_region: '%#v'",
			data["sts_region"], resp.Data["sts_region"])
	}
}
