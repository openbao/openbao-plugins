package backend_test

import (
	"context"
	"testing"

	backend "github.com/openbao/openbao-plugins/secrets/home"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// newTestBackend creates a Home backend wired to an in-memory storage backend,
// the same pattern used throughout the OpenBao codebase for unit tests.
func newTestBackend(t *testing.T) (logical.Backend, logical.Storage) {
	t.Helper()

	storage := &logical.InmemStorage{}
	config := &logical.BackendConfig{
		StorageView: storage,
		System:      logical.StaticSystemView{},
	}

	b, err := backend.Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return b, storage
}

// entityRequest builds a logical.Request with a stable EntityID, simulating a
// token that has been resolved to an Identity entity.
func entityRequest(op logical.Operation, path string, data map[string]interface{}, entityID string, storage logical.Storage) *logical.Request {
	req := &logical.Request{
		Operation: op,
		Path:      path,
		Data:      data,
		Storage:   storage,
		EntityID:  entityID,
	}
	return req
}

// ── Write / Read ─────────────────────────────────────────────────────────────

func TestWriteAndRead(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()
	entityID := "entity-abc-123"

	// Write
	writeReq := entityRequest(logical.UpdateOperation, "mysecret", map[string]interface{}{
		"password": "hunter2",
	}, entityID, s)
	resp, err := b.HandleRequest(ctx, writeReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("write failed: err=%v resp=%v", err, resp)
	}

	// Read back
	readReq := entityRequest(logical.ReadOperation, "mysecret", nil, entityID, s)
	resp, err = b.HandleRequest(ctx, readReq)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if v, ok := resp.Data["password"]; !ok || v != "hunter2" {
		t.Fatalf("unexpected data: %v", resp.Data)
	}
}

// ── Isolation between entities ───────────────────────────────────────────────

func TestEntityIsolation(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()

	entityA := "entity-aaa"
	entityB := "entity-bbb"

	// Entity A writes a secret.
	writeA := entityRequest(logical.UpdateOperation, "secret", map[string]interface{}{
		"value": "only-for-A",
	}, entityA, s)
	if _, err := b.HandleRequest(ctx, writeA); err != nil {
		t.Fatalf("write A failed: %v", err)
	}

	// Entity B should NOT see entity A's secret.
	readB := entityRequest(logical.ReadOperation, "secret", nil, entityB, s)
	resp, err := b.HandleRequest(ctx, readB)
	if err != nil {
		t.Fatalf("read B failed: %v", err)
	}
	if resp != nil {
		t.Fatalf("entity B should not see entity A's secret, got: %v", resp.Data)
	}
}

// ── Same entity, different tokens, shared access ─────────────────────────────

func TestSameEntitySharedAccess(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()
	entityID := "entity-shared-xyz"

	// Token 1 (same entity) writes.
	write := entityRequest(logical.UpdateOperation, "shared-key", map[string]interface{}{
		"data": "hello",
	}, entityID, s)
	if _, err := b.HandleRequest(ctx, write); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Token 2 (same entity) reads – simulated by using the same entityID.
	read := entityRequest(logical.ReadOperation, "shared-key", nil, entityID, s)
	resp, err := b.HandleRequest(ctx, read)
	if err != nil || resp == nil {
		t.Fatalf("read by second token of same entity failed: err=%v resp=%v", err, resp)
	}
	if resp.Data["data"] != "hello" {
		t.Fatalf("expected 'hello', got %v", resp.Data["data"])
	}
}

// ── No entity ID → permission denied ─────────────────────────────────────────

func TestNoEntityID(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()

	// Empty EntityID simulates root token or tokens without an entity.
	req := entityRequest(logical.UpdateOperation, "anything", map[string]interface{}{
		"k": "v",
	}, "" /* no entity */, s)

	resp, err := b.HandleRequest(ctx, req)

	// The backend returns a logical.ErrPermissionDenied alongside an error
	// response; either signal is acceptable for this test.
	if err == nil && (resp == nil || !resp.IsError()) {
		t.Fatal("expected error for request with no EntityID, got success")
	}
}

// ── Delete ───────────────────────────────────────────────────────────────────

func TestDelete(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()
	entityID := "entity-del-test"

	write := entityRequest(logical.UpdateOperation, "todelete", map[string]interface{}{
		"foo": "bar",
	}, entityID, s)
	if _, err := b.HandleRequest(ctx, write); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	del := entityRequest(logical.DeleteOperation, "todelete", nil, entityID, s)
	if _, err := b.HandleRequest(ctx, del); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	read := entityRequest(logical.ReadOperation, "todelete", nil, entityID, s)
	resp, err := b.HandleRequest(ctx, read)
	if err != nil {
		t.Fatalf("read after delete failed: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response after delete, got: %v", resp.Data)
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestList(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()
	entityID := "entity-list-test"

	for _, path := range []string{"alpha", "beta", "gamma"} {
		w := entityRequest(logical.UpdateOperation, path, map[string]interface{}{"x": "y"}, entityID, s)
		if _, err := b.HandleRequest(ctx, w); err != nil {
			t.Fatalf("write %s failed: %v", path, err)
		}
	}

	list := entityRequest(logical.ListOperation, "", nil, entityID, s)
	resp, err := b.HandleRequest(ctx, list)
	if err != nil || resp == nil {
		t.Fatalf("list failed: err=%v resp=%v", err, resp)
	}

	rawKeys, ok := resp.Data["keys"]
	if !ok {
		t.Fatal("list response missing 'keys' field")
	}
	keys, ok := rawKeys.([]string)
	if !ok {
		t.Fatalf("unexpected keys type: %T", rawKeys)
	}

	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !keySet[expected] {
			t.Errorf("expected key %q in list, got: %v", expected, keys)
		}
	}
}

// ── Nested paths ─────────────────────────────────────────────────────────────

func TestNestedPaths(t *testing.T) {
	b, s := newTestBackend(t)
	ctx := context.Background()
	entityID := "entity-nested"

	write := entityRequest(logical.UpdateOperation, "folder/subfolder/key", map[string]interface{}{
		"nested": "value",
	}, entityID, s)
	if _, err := b.HandleRequest(ctx, write); err != nil {
		t.Fatalf("nested write failed: %v", err)
	}

	read := entityRequest(logical.ReadOperation, "folder/subfolder/key", nil, entityID, s)
	resp, err := b.HandleRequest(ctx, read)
	if err != nil || resp == nil {
		t.Fatalf("nested read failed: err=%v resp=%v", err, resp)
	}
	if resp.Data["nested"] != "value" {
		t.Fatalf("unexpected nested value: %v", resp.Data)
	}
}
