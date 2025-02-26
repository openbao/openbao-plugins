// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "required params happy path",
			config: map[string]interface{}{
				"resource":  "resource",
				"tenant_id": "tid",
			},
			expected: map[string]interface{}{
				"client_id":                  "",
				"environment":                "",
				"identity_token_audience":    "",
				"identity_token_ttl":         int64(0),
				"max_retries":                defaultMaxRetries,
				"max_retry_delay":            defaultMaxRetryDelay,
				"resource":                   "resource",
				"retry_delay":                defaultRetryDelay,
				"root_password_ttl":          15768000,
				"tenant_id":                  "tid",
				"rotation_window":            float64(0),
				"rotation_period":            float64(0),
				"rotation_schedule":          "",
				"disable_automated_rotation": false,
			},
		},
		{
			name: "environment happy path",
			config: map[string]interface{}{
				"resource":    "resource",
				"tenant_id":   "tid",
				"environment": "AzurePublicCloud",
			},
			expected: map[string]interface{}{
				"client_id":                  "",
				"environment":                "AzurePublicCloud",
				"identity_token_audience":    "",
				"identity_token_ttl":         int64(0),
				"max_retries":                defaultMaxRetries,
				"max_retry_delay":            defaultMaxRetryDelay,
				"resource":                   "resource",
				"retry_delay":                defaultRetryDelay,
				"root_password_ttl":          15768000,
				"tenant_id":                  "tid",
				"rotation_window":            float64(0),
				"rotation_period":            float64(0),
				"rotation_schedule":          "",
				"disable_automated_rotation": false,
			},
		},
		{
			name:    "errors when required params unset",
			config:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "errors on invalid environment",
			config: map[string]interface{}{
				"resource":    "resource",
				"tenant_id":   "tid",
				"environment": "AzureNotRealCloud",
			},
			wantErr: true,
		},
		{
			name: "client_secret and identity_token_audience are mutually exclusive",
			config: map[string]interface{}{
				"client_id":               "testClientId",
				"client_secret":           "testClientSecret",
				"identity_token_audience": "vault-azure-secrets-d0f0d253",
				"resource":                "resource",
				"tenant_id":               "tid",
			},
			wantErr: true,
		},
		{
			name: "wif happy path",
			config: map[string]interface{}{
				"identity_token_audience": "vault-azure-secrets-d0f0d253",
				"identity_token_ttl":      int64(500),
				"resource":                "resource",
				"tenant_id":               "tid",
			},
			expected: map[string]interface{}{
				"client_id":                  "",
				"environment":                "",
				"identity_token_audience":    "vault-azure-secrets-d0f0d253",
				"identity_token_ttl":         int64(500),
				"max_retries":                int32(3),
				"max_retry_delay":            time.Duration(60000000000),
				"resource":                   "resource",
				"retry_delay":                time.Duration(4000000000),
				"root_password_ttl":          15768000,
				"tenant_id":                  "tid",
				"rotation_window":            float64(0),
				"rotation_period":            float64(0),
				"rotation_schedule":          "",
				"disable_automated_rotation": false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, s := getTestBackend(t)
			_, err := testConfigCreate(t, b, s, tc.config)

			if tc.wantErr {
				assert.True(t, err != nil, "expected error, got none")
			}

			if !tc.wantErr {
				if err != nil {
					t.Fatal(err)
				}

				testConfigRead(t, b, s, tc.expected)

				// Test that updating one element retains the others
				tc.expected["tenant_id"] = "foo"
				configSubset := map[string]interface{}{
					"tenant_id": "foo",
				}

				_, err = testConfigUpdate(t, b, s, configSubset)
				if err != nil {
					t.Fatal(err)
				}

				testConfigRead(t, b, s, tc.expected)
			}
		})
	}
}

func TestRotationConfig(t *testing.T) {
	// Ensure rotation settings return an error
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id":       "tid",
		"resource":        "res",
		"rotation_period": 10,
	}
	_, err := testConfigCreate(t, b, s, configData)
	if err.Error() != "error registering rotation job: rotation manager capabilities not supported in Vault community edition" {
		t.Fail()
	}
}

func TestConfigDelete(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}
	if _, err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "config",
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp != nil {
		t.Fatal("expected nil config after delete")
	}
}

func testConfigCreate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	return testConfigCreateUpdate(t, b, logical.CreateOperation, s, d)
}

func testConfigUpdate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	return testConfigCreateUpdate(t, b, logical.UpdateOperation, s, d)
}

func testConfigCreateUpdate(t *testing.T, b logical.Backend, op logical.Operation, s logical.Storage, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.IsError() {
		return resp, resp.Error()
	}
	return resp, nil
}

func testConfigRead(t *testing.T, b *azureAuthBackend, s logical.Storage, expected map[string]interface{}) {
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

	require.Equal(t, expected, resp.Data)
}

func TestConfig_RetryDefaults(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}

	if _, err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	expected := map[string]interface{}{
		"client_id":                  "",
		"environment":                "",
		"identity_token_audience":    "",
		"identity_token_ttl":         int64(0),
		"max_retries":                defaultMaxRetries,
		"max_retry_delay":            defaultMaxRetryDelay,
		"resource":                   "resource",
		"retry_delay":                defaultRetryDelay,
		"root_password_ttl":          15768000,
		"tenant_id":                  "tid",
		"rotation_window":            float64(0),
		"rotation_period":            float64(0),
		"rotation_schedule":          "",
		"disable_automated_rotation": false,
	}
	testConfigRead(t, b, s, expected)

	config, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	azureSettings, err := b.getAzureSettings(context.Background(), config)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if azureSettings.MaxRetries != defaultMaxRetries {
		t.Fatalf("wrong 'max_retries' default azure settings value: expected %v, got %v", defaultMaxRetries, azureSettings.MaxRetries)
	}

	if azureSettings.MaxRetryDelay != defaultMaxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' default azure settings value: expected %v, got %v", defaultMaxRetryDelay, azureSettings.MaxRetryDelay)
	}

	if azureSettings.RetryDelay != defaultRetryDelay {
		t.Fatalf("wrong 'retry_delay' default azure settings value: expected %v, got %v", defaultRetryDelay, azureSettings.RetryDelay)
	}
}

func TestConfig_RetryCustom(t *testing.T) {
	b, s := getTestBackend(t)
	maxRetries := int32(60)
	maxRetryDelay := time.Second * 120
	retryDelay := time.Second * 10

	configData := map[string]interface{}{
		"tenant_id":       "tid",
		"resource":        "resource",
		"max_retries":     maxRetries,
		"max_retry_delay": maxRetryDelay,
		"retry_delay":     retryDelay,
	}

	if _, err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	expected := map[string]interface{}{
		"client_id":                  "",
		"environment":                "",
		"identity_token_audience":    "",
		"identity_token_ttl":         int64(0),
		"max_retries":                maxRetries,
		"max_retry_delay":            maxRetryDelay,
		"resource":                   "resource",
		"retry_delay":                retryDelay,
		"root_password_ttl":          15768000,
		"tenant_id":                  "tid",
		"rotation_window":            float64(0),
		"rotation_period":            float64(0),
		"rotation_schedule":          "",
		"disable_automated_rotation": false,
	}
	testConfigRead(t, b, s, expected)

	config, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	azureSettings, err := b.getAzureSettings(context.Background(), config)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if azureSettings.MaxRetries != maxRetries {
		t.Fatalf("wrong 'max_retries' azure settings value: expected %v, got %v", maxRetries, azureSettings.MaxRetries)
	}

	if azureSettings.MaxRetryDelay != maxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' azure settings value: expected %v, got %v", maxRetryDelay, azureSettings.MaxRetryDelay)
	}

	if azureSettings.RetryDelay != retryDelay {
		t.Fatalf("wrong 'retry_delay' azure settings value: expected %v, got %v", retryDelay, azureSettings.RetryDelay)
	}
}
