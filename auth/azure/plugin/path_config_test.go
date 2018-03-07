package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/logical"
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

func testConfigCreate(t *testing.T, b *azureAuthBackend, s logical.Storage, d map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      fmt.Sprintf("config"),
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
