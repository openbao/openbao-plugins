package azureauth

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_Write(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{}
	if err := testConfigCreate(t, b, s, configData); err == nil {
		t.Fatal("expected error")
	}

	configData = map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	configData["environment"] = "AzureNotRealCloud"
	if err := testConfigCreate(t, b, s, configData); err == nil {
		t.Fatal("expected error")
	}

	configData["environment"] = "AzureUSGovernmentCloud"
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

}

func TestConfigDelete(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
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

func testConfigCreate(t *testing.T, b *azureAuthBackend, s logical.Storage, d map[string]interface{}) error {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		return err
	}
	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}
