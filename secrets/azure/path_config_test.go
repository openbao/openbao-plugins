package azuresecrets

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig(t *testing.T) {
	b, s := getTestBackend(t, false)

	// Test valid config
	expectedConfig := map[string]interface{}{
		"subscription_id":         "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":               "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":               "testClientId",
		"client_secret":           "testClientSecret",
		"environment":             "AZURECHINACLOUD",
		"use_microsoft_graph_api": false,
		"root_password_ttl":       int((24 * time.Hour).Seconds()),
	}

	testConfigCreate(t, b, s, expectedConfig)

	delete(expectedConfig, "client_secret")
	testConfigRead(t, b, s, expectedConfig)

	// Test test updating one element retains the others
	expectedConfig["tenant_id"] = "800e371d-ee51-4145-9ac8-5c43e4ceb79b"
	configSubset := map[string]interface{}{
		"tenant_id": "800e371d-ee51-4145-9ac8-5c43e4ceb79b",
	}
	testConfigCreate(t, b, s, configSubset)
	testConfigUpdate(t, b, s, expectedConfig)

	// Test bad environment
	expectedConfig = map[string]interface{}{
		"environment": "invalidEnv",
	}

	resp, _ := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Data:      expectedConfig,
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
		"root_password_ttl":       int((24 * time.Hour).Seconds()),
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
		"root_password_ttl":       0,
	}
	testConfigRead(t, b, s, config)
}

func TestAddAADWarning(t *testing.T) {
	type testCase struct {
		useMSGraphAPI bool
		resp          *logical.Response
		expectedResp  *logical.Response
	}

	tests := map[string]testCase{
		"nil resp, using AAD": {
			useMSGraphAPI: false,
			resp:          nil,
			expectedResp: &logical.Response{
				Warnings: []string{aadWarning},
			},
		},
		"nil resp, using ms-graph": {
			useMSGraphAPI: true,
			resp:          nil,
			expectedResp:  nil,
		},
		"non-nil resp, using AAD": {
			useMSGraphAPI: false,
			resp: &logical.Response{
				Data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expectedResp: &logical.Response{
				Data: map[string]interface{}{
					"foo": "bar",
				},
				Warnings: []string{aadWarning},
			},
		},
		"non-nil resp, using ms-graph": {
			useMSGraphAPI: true,
			resp: &logical.Response{
				Data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expectedResp: &logical.Response{
				Data: map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &azureConfig{
				UseMsGraphAPI: test.useMSGraphAPI,
			}
			actualResp := addAADWarning(test.resp, cfg)
			if !reflect.DeepEqual(actualResp, test.expectedResp) {
				t.Fatalf("Actual: %#v\nExpected: %#v", actualResp, test.expectedResp)
			}
		})
	}
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
