package azuresecrets

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/logical"
)

func TestRoleCreate(t *testing.T) {
	b, s := getTestBackend(t, true)

	t.Run("SP role", func(t *testing.T) {
		spRole1 := map[string]interface{}{
			"azure_roles": compactJSON(`[
		{
			"role_name": "Owner",
			"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner",
			"scope":  "test_scope_1"
		},
		{
			"role_name": "Owner2",
			"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner2",
			"scope":  "test_scope_2"
		}]`),
			"ttl":     int64(0),
			"max_ttl": int64(0),
		}

		spRole2 := map[string]interface{}{
			"azure_roles": compactJSON(`[
		{
			"role_name": "Contributor",
			"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
			"scope":  "test_scope_3"
		},
		{
			"role_name": "Contributor2",
			"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor2",
			"scope":  "test_scope_3"
		}]`),
			"ttl":     int64(300),
			"max_ttl": int64(3000),
		}

		// Verify basic updates of the name role
		name := generateUUID()
		testRoleCreate(t, b, s, name, spRole1)

		resp, err := testRoleRead(t, b, s, name)
		nilErr(t, err)

		convertRespTypes(resp.Data)
		equal(t, spRole1, resp.Data)

		testRoleCreate(t, b, s, name, spRole2)

		resp, err = testRoleRead(t, b, s, name)
		nilErr(t, err)

		convertRespTypes(resp.Data)
		equal(t, spRole2, resp.Data)
	})

	t.Run("Optional role TTLs", func(t *testing.T) {
		testRole := map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "Contributor",
					"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
					"scope":  "test_scope_3"
				}]`,
			)}

		// Verify that ttl and max_ttl are 0 if not provided
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		testRole["ttl"] = int64(0)
		testRole["max_ttl"] = int64(0)

		resp, err := testRoleRead(t, b, s, name)
		nilErr(t, err)

		convertRespTypes(resp.Data)
		equal(t, testRole, resp.Data)
	})

	t.Run("Role TTL Checks", func(t *testing.T) {
		b, s := getTestBackend(t, true)

		const skip = -999
		tests := []struct {
			ttl      int64
			maxTTL   int64
			expError bool
		}{
			{5, 10, false},
			{5, skip, false},
			{skip, 10, false},
			{100, 100, false},
			{101, 100, true},
			{101, 0, false},
		}

		for i, test := range tests {
			role := map[string]interface{}{
				"azure_roles": compactJSON(`[{}]`),
			}

			if test.ttl != skip {
				role["ttl"] = test.ttl
			}
			if test.maxTTL != skip {
				role["max_ttl"] = test.maxTTL
			}
			name := generateUUID()
			resp, err := b.HandleRequest(context.Background(), &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "roles/" + name,
				Data:      role,
				Storage:   s,
			})
			nilErr(t, err)

			if resp.IsError() != test.expError {
				t.Fatalf("\ncase %d\nexp error: %t\ngot: %v", i, test.expError, err)
			}
		}
	})

	t.Run("Role name lookup", func(t *testing.T) {
		b, s := getTestBackend(t, true)
		var role = map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "Owner",
					"role_id": "",
					"scope":  "test_scope_1"
				},
				{
					"role_name": "will be replaced",
					"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
					"scope":  "test_scope_2"
				}
			]`),
		}

		name := generateUUID()
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "roles/" + name,
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)
		if resp.IsError() {
			t.Fatal("received unxpected error response")
		}

		resp, err = testRoleRead(t, b, s, name)
		nilErr(t, err)
		roles := resp.Data["azure_roles"].([]*azureRole)
		equal(t, "Owner", roles[0].RoleName)
		equal(t, "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner", roles[0].RoleID)
		equal(t, "Contributor", roles[1].RoleName)
		equal(t, "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor", roles[1].RoleID)
	})

	t.Run("Role name lookup (multiple match)", func(t *testing.T) {
		b, s := getTestBackend(t, true)

		// if role_name=="multiple", the mock will return multiple IDs, which are not allowed
		var role = map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "multiple",
					"role_id": "",
					"scope":  "test_scope_1"
				},
				{
					"role_name": "will be replaced",
					"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
					"scope":  "test_scope_2"
				}
			]`),
		}

		name := generateUUID()
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "roles/" + name,
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)
		if !resp.IsError() {
			t.Fatal("expected error response")
		}
	})

}

func TestRoleCreateBad(t *testing.T) {
	b, s := getTestBackend(t, true)

	// missing roles
	role := map[string]interface{}{}
	resp := testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg := "missing Azure role definitions"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}

	// invalid roles
	role = map[string]interface{}{"azure_roles": "asdf"}
	resp = testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg = "invalid Azure role definitions"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}
}

func TestRoleUpdateError(t *testing.T) {
	b, s := getTestBackend(t, true)

	role := map[string]interface{}{
		"azure_roles": compactJSON(`[{}]`),
	}

	name := generateUUID()
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/" + name,
		Data:      role,
		Storage:   s,
	})

	if err == nil {
		t.Fatal("expected error trying to update nonexistent role")
	}
}

func TestRoleList(t *testing.T) {
	b, s := getTestBackend(t, true)

	// Verify empty list
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   s,
	})
	if err != nil {
		t.Fatalf("expected nil error, actual:%#v", err.Error())
	}

	if resp.Data["keys"] != nil {
		t.Fatalf("expected nil, actual: %#v", resp.Data["keys"])
	}

	// Add some roles and verify the resulting list
	role := map[string]interface{}{
		"azure_roles": compactJSON(`[{}]`),
	}
	testRoleCreate(t, b, s, "r1", role)
	testRoleCreate(t, b, s, "r2", role)
	testRoleCreate(t, b, s, "r3", role)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   s,
	})
	nilErr(t, err)

	exp := []string{"r1", "r2", "r3"}
	sort.Strings(resp.Data["keys"].([]string))
	equal(t, exp, resp.Data["keys"])

	// Delete a role and verify list is updated
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/r2",
		Storage:   s,
	})
	nilErr(t, err)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   s,
	})
	nilErr(t, err)

	exp = []string{"r1", "r3"}
	sort.Strings(resp.Data["keys"].([]string))
	equal(t, exp, resp.Data["keys"])
}

func TestRoleDelete(t *testing.T) {
	b, s := getTestBackend(t, true)
	name := "test_role"
	nameAlt := "test_role_alt"

	role := map[string]interface{}{
		"azure_roles": compactJSON(`[{}]`),
	}

	// Create two roles and verify they're present
	testRoleCreate(t, b, s, name, role)
	testRoleCreate(t, b, s, nameAlt, role)

	// Delete one role and verify it is gone, and the other remains
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      fmt.Sprintf("roles/%s", name),
		Storage:   s,
	})
	nilErr(t, err)

	resp, err = testRoleRead(t, b, s, name)
	if resp != nil || err != nil {
		t.Fatalf("expected nil response and error, actual:%#v and %#v", resp, err.Error())
	}

	resp, err = testRoleRead(t, b, s, nameAlt)
	nilErr(t, err)
	if resp == nil {
		t.Fatalf("expected non-nil response, actual:%#v", resp)
	}

	// Verify that delete against a missing role is a succesful no-op
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/not_a_role",
		Storage:   s,
	})
	if resp != nil || err != nil {
		t.Fatalf("expected nil response and error, actual:%#v and %#v", resp, err)
	}
}

// Utility function to create a role and fail on errors
func testRoleCreate(t *testing.T, b *azureSecretBackend, s logical.Storage, name string, d map[string]interface{}) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      fmt.Sprintf("roles/%s", name),
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
}

// Utility function to create a role while, returning any response (including errors)
func testRoleCreateBasic(t *testing.T, b *azureSecretBackend, s logical.Storage, name string, d map[string]interface{}) *logical.Response {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      fmt.Sprintf("roles/%s", name),
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		t.Fatal(err)
	}

	return resp
}

// Utility function to read a role and return any errors
func testRoleRead(t *testing.T, b *azureSecretBackend, s logical.Storage, name string) (*logical.Response, error) {
	t.Helper()
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      fmt.Sprintf("roles/%s", name),
		Storage:   s,
	})
}

// Utility function to convert response types back to the format that is used as
// input in order to streamline the comparison steps.
func convertRespTypes(data map[string]interface{}) {
	data["azure_roles"] = encodeJSON(data["azure_roles"])
	data["ttl"] = int64(data["ttl"].(time.Duration))
	data["max_ttl"] = int64(data["max_ttl"].(time.Duration))
}
