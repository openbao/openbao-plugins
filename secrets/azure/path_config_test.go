package azuresecrets

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig(t *testing.T) {
	b, s := getTestBackend(t, false)

	// Test valid config
	config := map[string]interface{}{
		"subscription_id":         "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":               "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":               "testClientId",
		"client_secret":           "testClientSecret",
		"environment":             "AZURECHINACLOUD",
		"use_microsoft_graph_api": false,
	}

	testConfigCreate(t, b, s, config)

	delete(config, "client_secret")
	testConfigRead(t, b, s, config)

	// Test test updating one element retains the others
	config["tenant_id"] = "800e371d-ee51-4145-9ac8-5c43e4ceb79b"
	configSubset := map[string]interface{}{
		"tenant_id": "800e371d-ee51-4145-9ac8-5c43e4ceb79b",
	}
	testConfigCreate(t, b, s, configSubset)
	testConfigUpdate(t, b, s, config)

	// Test bad environment
	config = map[string]interface{}{
		"environment": "invalidEnv",
	}

	resp, _ := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Data:      config,
		Storage:   s,
	})

	if !resp.IsError() {
		t.Fatal("expected a response error")
	}
}

func TestConfigDelete(t *testing.T) {
	b, s := getTestBackend(t, false)

	// Test valid config
	config := map[string]interface{}{
		"subscription_id":         "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":               "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":               "testClientId",
		"client_secret":           "testClientSecret",
		"environment":             "AZURECHINACLOUD",
		"use_microsoft_graph_api": false,
	}

	testConfigCreate(t, b, s, config)

	delete(config, "client_secret")
	testConfigRead(t, b, s, config)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "config",
		Storage:   s,
	})

	assertErrorIsNil(t, err)

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	config = map[string]interface{}{
		"subscription_id":         "",
		"tenant_id":               "",
		"client_id":               "",
		"environment":             "",
		"use_microsoft_graph_api": false,
	}
	testConfigRead(t, b, s, config)
}

func testConfigCreate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) {
	t.Helper()
	testConfigCreateUpdate(t, b, logical.CreateOperation, s, d)
}

func testConfigUpdate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) {
	t.Helper()
	testConfigCreateUpdate(t, b, logical.UpdateOperation, s, d)
}

func testConfigCreateUpdate(t *testing.T, b logical.Backend, op logical.Operation, s logical.Storage, d map[string]interface{}) {
	t.Helper()

	// save and restore the client since the config change will clear it
	settings := b.(*azureSecretBackend).settings
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: op,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	b.(*azureSecretBackend).settings = settings

	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
}

func testConfigRead(t *testing.T, b logical.Backend, s logical.Storage, expected map[string]interface{}) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   s,
	})

	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	equal(t, expected, resp.Data)
}
