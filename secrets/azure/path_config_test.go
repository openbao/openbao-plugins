package azuresecrets

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/logical"
	"github.com/mitchellh/copystructure"
)

func TestConfig(t *testing.T) {
	b, s := getTestBackend(t, false)

	// Test valid config
	cfg := map[string]interface{}{
		"subscription_id": "a228ceec-bf1a-4411-9f95-39678d8cdb34",
		"tenant_id":       "7ac36e27-80fc-4209-a453-e8ad83dc18c2",
		"client_id":       "testClientId",
		"client_secret":   "testClientSecret",
		"environment":     "AZURECHINACLOUD",
		"ttl":             int64(5),
		"max_ttl":         int64(18000),
	}

	testConfigCreate(t, b, s, cfg)

	delete(cfg, "client_secret")
	testConfigRead(t, b, s, cfg)

	// Test string versions of ttls
	c, _ := copystructure.Copy(cfg)
	cfg2 := c.(map[string]interface{})
	cfg2["ttl"] = "5s"
	cfg2["max_ttl"] = "5h"

	testConfigCreate(t, b, s, cfg2)
	testConfigRead(t, b, s, cfg)

	// Test test updating one element retains the others
	cfg["tenant_id"] = "800e371d-ee51-4145-9ac8-5c43e4ceb79b"
	cfgSubset := map[string]interface{}{
		"tenant_id": "800e371d-ee51-4145-9ac8-5c43e4ceb79b",
	}
	testConfigCreate(t, b, s, cfgSubset)
	testConfigUpdate(t, b, s, cfg)

	// Test bad environment
	cfg = map[string]interface{}{
		"environment": "invalidEnv",
	}

	resp, _ := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Data:      cfg,
		Storage:   s,
	})

	if !resp.IsError() {
		t.Fatal("expected a response error")
	}
}

func TestConfigTTLs(t *testing.T) {
	b, s := getTestBackend(t, false)

	const skip = -999
	tests := []struct {
		ttl      int64
		max_ttl  int64
		expError bool
	}{
		{5, 10, false},
		{5, skip, false},
		{skip, 10, false},
		{-1, skip, true},
		{skip, -1, true},
		{-2, -1, true},
		{100, 100, false},
		{101, 100, true},
		{101, 0, false},
	}

	for i, test := range tests {
		cfg := map[string]interface{}{}
		if test.ttl != skip {
			cfg["ttl"] = test.ttl
		}
		if test.max_ttl != skip {
			cfg["max_ttl"] = test.max_ttl
		}
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "config",
			Data:      cfg,
			Storage:   s,
		})
		nilErr(t, err)

		if resp.IsError() != test.expError {
			t.Fatalf("\ncase %d\nexp error: %t\ngot: %v", i, test.expError, err)
		}
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

	// save and restore the mock provider since the config change will clear it
	provider := b.(*azureSecretBackend).provider
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: op,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	b.(*azureSecretBackend).provider = provider

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
