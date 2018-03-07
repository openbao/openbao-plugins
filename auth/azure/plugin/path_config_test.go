package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/logical"
)

func testConfigCreate(t *testing.T, b *azureAuthBackend, s logical.Storage, d map[string]interface{}) {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      fmt.Sprintf("config"),
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
	b.provider = new(mockProvider)
}
