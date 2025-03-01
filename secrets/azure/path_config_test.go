// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "root_password_ttl defaults to 6 months",
			config: map[string]interface{}{
				"subscription_id": "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":       "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":       "testClientId",
				"client_secret":   "testClientSecret",
			},
			expected: map[string]interface{}{
				"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":         "testClientId",
				"environment":       "",
				"root_password_ttl": 15768000,
			},
		},
		{
			name: "root_password_ttl set if provided",
			config: map[string]interface{}{
				"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":         "testClientId",
				"client_secret":     "testClientSecret",
				"root_password_ttl": "1m",
			},
			expected: map[string]interface{}{
				"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":         "testClientId",
				"environment":       "",
				"root_password_ttl": 60,
			},
		},
		{
			name: "environment set if provided",
			config: map[string]interface{}{
				"subscription_id": "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":       "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":       "testClientId",
				"client_secret":   "testClientSecret",
				"environment":     "AZURECHINACLOUD",
			},
			expected: map[string]interface{}{
				"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
				"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
				"client_id":         "testClientId",
				"root_password_ttl": 15768000,
				"environment":       "AZURECHINACLOUD",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, s := getTestBackendMocked(t, false)
			testConfigCreate(t, b, s, tc.config, tc.wantErr)

			if !tc.wantErr {
				testConfigRead(t, b, s, tc.expected)

				// Test that updating one element retains the others
				tc.expected["tenant_id"] = "800e371d-ee51-4145-9ac8-5c43e4ceb79b"
				configSubset := map[string]interface{}{
					"tenant_id": "800e371d-ee51-4145-9ac8-5c43e4ceb79b",
				}

				testConfigUpdate(t, b, s, configSubset, tc.wantErr)
				testConfigRead(t, b, s, tc.expected)
			}
		})
	}
}

func TestConfigEnvironmentClouds(t *testing.T) {
	b, s := getTestBackendMocked(t, false)

	config := map[string]interface{}{
		"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":         "testClientId",
		"client_secret":     "testClientSecret",
		"environment":       "AZURECHINACLOUD",
		"root_password_ttl": int((24 * time.Hour).Seconds()),
	}

	testConfigCreate(t, b, s, config, false)

	tests := []struct {
		env      string
		url      string
		expError bool
	}{
		{"AZURECHINACLOUD", "https://microsoftgraph.chinacloudapi.cn", false},
		{"AZUREPUBLICCLOUD", "https://graph.microsoft.com", false},
		{"AZUREUSGOVERNMENTCLOUD", "https://graph.microsoft.us", false},
		{"invalidEnv", "", true},
	}

	for _, test := range tests {
		expectedConfig := map[string]interface{}{
			"environment": test.env,
		}

		// Error is in the response, not in the error variable.
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "config",
			Data:      expectedConfig,
			Storage:   s,
		})

		if resp.Error() == nil && test.expError {
			t.Fatal("expected error, got none")
		} else if err != nil && !test.expError {
			t.Fatalf("expected no errors: %s", err)
		}

		if !test.expError {
			config, err := b.getConfig(context.Background(), s)
			if err != nil {
				t.Fatal(err)
			}

			clientSettings, err := b.getClientSettings(context.Background(), config)
			if err != nil {
				t.Fatal(err)
			}

			if clientSettings.GraphURI != test.url {
				t.Fatalf("expected url %s, got %s", test.url, clientSettings.GraphURI)
			}
		}
	}
}

func TestConfigDelete(t *testing.T) {
	b, s := getTestBackendMocked(t, false)

	// Test valid config
	config := map[string]interface{}{
		"subscription_id":   "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":         "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":         "testClientId",
		"client_secret":     "testClientSecret",
		"environment":       "AZURECHINACLOUD",
		"root_password_ttl": int((24 * time.Hour).Seconds()),
	}

	testConfigCreate(t, b, s, config, false)

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
		"subscription_id":   "",
		"tenant_id":         "",
		"client_id":         "",
		"environment":       "",
		"root_password_ttl": 0,
	}
	testConfigRead(t, b, s, config)
}

func testConfigCreate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}, wantErr bool) {
	t.Helper()
	testConfigCreateUpdate(t, b, logical.CreateOperation, s, d, wantErr)
}

func testConfigUpdate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}, wantErr bool) {
	t.Helper()
	testConfigCreateUpdate(t, b, logical.UpdateOperation, s, d, wantErr)
}

func testConfigCreateUpdate(t *testing.T, b logical.Backend, op logical.Operation, s logical.Storage, d map[string]interface{}, wantErr bool) {
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

	if !wantErr && err != nil {
		t.Fatal(err)
	}

	if !wantErr && resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	if wantErr {
		assert.True(t, resp.IsError() || err != nil, "expected error, got nil")
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
