// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/helper/logging"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestRoleCreate(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

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
			"azure_groups": compactJSON(`[
		{
			"group_name": "foo",
			"object_id": "239b11fe-6adf-409a-b231-08b918e9de23FAKE_GROUP-foo"
		},
		{
			"group_name": "bar",
			"object_id": "31c5bf7e-e1e8-42c8-882c-856f776290afFAKE_GROUP-bar"
		}]`),
			"ttl":                   int64(0),
			"max_ttl":               int64(0),
			"explicit_max_ttl":      int64(0),
			"application_object_id": "",
			"permanently_delete":    true,
			"persist_app":           false,
			"sign_in_audience":      "AzureADMyOrg",
			"tags":                  []string{"project:vault_test"},
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
			"azure_groups": compactJSON(`[
		{
			"group_name": "baz",
			"object_id": "de55c630-8415-4bd3-b329-530688e60173FAKE_GROUP-baz"
		},
		{
			"group_name": "bam",
			"object_id": "a6a834a6-36c3-4575-8e2b-05095963d603FAKE_GROUP-bam"
		}]`),
			"ttl":                   int64(300),
			"max_ttl":               int64(3000),
			"explicit_max_ttl":      int64(3000),
			"application_object_id": "",
			"permanently_delete":    true,
			"persist_app":           false,
			"sign_in_audience":      "AzureADMultipleOrgs",
			"tags":                  []string{"project:vault_test"},
		}

		// Verify basic updates of the name role
		name := generateUUID()
		testRoleCreate(t, b, s, name, spRole1)

		resp, err := testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		equal(t, spRole1, resp.Data)

		testRoleCreate(t, b, s, name, spRole2)

		resp, err = testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		equal(t, spRole2, resp.Data)
	})

	t.Run("SP persistent role", func(t *testing.T) {
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
			"azure_groups": compactJSON(`[
		{
			"group_name": "foo",
			"object_id": "239b11fe-6adf-409a-b231-08b918e9de23FAKE_GROUP-foo"
		},
		{
			"group_name": "bar",
			"object_id": "31c5bf7e-e1e8-42c8-882c-856f776290afFAKE_GROUP-bar"
		}]`),
			"ttl":                   int64(0),
			"max_ttl":               int64(0),
			"explicit_max_ttl":      int64(0),
			"application_object_id": "",
			"persist_app":           true,
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
			"azure_groups": compactJSON(`[
		{
			"group_name": "baz",
			"object_id": "de55c630-8415-4bd3-b329-530688e60173FAKE_GROUP-baz"
		},
		{
			"group_name": "bam",
			"object_id": "a6a834a6-36c3-4575-8e2b-05095963d603FAKE_GROUP-bam"
		}]`),
			"ttl":                   int64(300),
			"max_ttl":               int64(3000),
			"explicit_max_ttl":      int64(3000),
			"application_object_id": "",
			"persist_app":           true,
		}

		// Verify basic updates of the name role
		name := generateUUID()
		testRoleCreate(t, b, s, name, spRole1)

		resp, err := testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		assertNotEmptyString(t, resp.Data["application_object_id"].(string))

		//Get role and check all values are set
		fullRole, err := getRole(context.Background(), name, s)
		assertErrorIsNil(t, err)

		assertNotNil(t, fullRole.ApplicationID)
		assertNotNil(t, fullRole.ApplicationObjectID)
		assertStrSliceIsNotEmpty(t, fullRole.GroupMembershipIDs)
		assertStrSliceIsNotEmpty(t, fullRole.RoleAssignmentIDs)
		assertNotNil(t, fullRole.ServicePrincipalObjectID)

		originalAppID := fullRole.ApplicationID
		originalAppObjID := fullRole.ApplicationObjectID

		testRoleCreate(t, b, s, name, spRole2)
		resp, err = testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		assertNotEmptyString(t, resp.Data["application_object_id"].(string))

		fullRole, err = getRole(context.Background(), name, s)
		assertErrorIsNil(t, err)

		equal(t, fullRole.ApplicationID, originalAppID)
		equal(t, fullRole.ApplicationObjectID, originalAppObjID)

	})

	t.Run("Static SP role", func(t *testing.T) {
		spRole1 := map[string]interface{}{
			"application_object_id": "00000000-0000-0000-0000-000000000000",
			"ttl":                   int64(300),
			"max_ttl":               int64(3000),
			"explicit_max_ttl":      int64(3000),
			"azure_roles":           "[]",
			"azure_groups":          "[]",
			"sign_in_audience":      "PersonalMicrosoftAccount",
			"tags":                  []string{"environment:production"},
			"permanently_delete":    false,
			"persist_app":           false,
		}

		name := generateUUID()
		testRoleCreate(t, b, s, name, spRole1)

		resp, err := testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		equal(t, spRole1, resp.Data)
	})

	t.Run("Optional role TTLs", func(t *testing.T) {
		testRole := map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "Contributor",
					"role_id": "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
					"scope":  "test_scope_3"
				}]`,
			),
			"application_object_id": "",
			"sign_in_audience":      "AzureADandPersonalMicrosoftAccount",
			"tags":                  []string{"project:vault_testing"},
			"azure_groups":          "[]",
			"persist_app":           false,
		}

		// Verify that ttl, max_ttl and explicit_max_ttl are 0 if not provided
		// and that permanently_delete is false if not provided
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		testRole["ttl"] = int64(0)
		testRole["max_ttl"] = int64(0)
		testRole["explicit_max_ttl"] = int64(0)
		testRole["permanently_delete"] = false

		resp, err := testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)

		convertRespTypes(resp.Data)
		equal(t, testRole, resp.Data)
	})

	t.Run("Role TTL Checks", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

		const skip = -999
		tests := []struct {
			ttl            int64
			maxTTL         int64
			explicitMaxTTL int64
			expError       bool
		}{
			{5, 10, skip, false},
			{5, skip, skip, false},
			{skip, 10, skip, false},
			{100, 100, skip, false},
			{101, 100, skip, true},
			{101, 0, skip, false},
			{5, 10, 10, false},
			{10, skip, 5, true},
			{10, 10, 5, true},
			{skip, skip, 10, false},
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
			if test.explicitMaxTTL != skip {
				role["explicit_max_ttl"] = test.explicitMaxTTL
			}
			name := generateUUID()
			resp, err := b.HandleRequest(context.Background(), &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "roles/" + name,
				Data:      role,
				Storage:   s,
			})
			assertErrorIsNil(t, err)

			if resp.IsError() != test.expError {
				t.Fatalf("\ncase %d\nexp error: %t\ngot: %v", i, test.expError, err)
			}
		}
	})

	t.Run("Role name lookup", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)
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
		assertErrorIsNil(t, err)
		if resp.IsError() {
			t.Fatalf("received unexpected error response: %v", resp.Error())
		}

		resp, err = testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)
		roles := resp.Data["azure_roles"].([]*AzureRole)
		equal(t, "Owner", roles[0].RoleName)
		equal(t, "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner", roles[0].RoleID)
		equal(t, "Contributor", roles[1].RoleName)
		equal(t, "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor", roles[1].RoleID)
	})

	t.Run("Group name lookup", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)
		var group = map[string]interface{}{
			"azure_groups": compactJSON(`[
				{
					"group_name": "baz",
					"object_id": ""
				},
				{
					"group_name": "will be replaced",
					"object_id": "a6a834a6-36c3-4575-8e2b-05095963d603FAKE_GROUP-bam"
				}
			]`),
		}

		name := generateUUID()
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "roles/" + name,
			Data:      group,
			Storage:   s,
		})
		assertErrorIsNil(t, err)
		if resp.IsError() {
			t.Fatalf("received unexpected error response: %v", resp.Error())
		}

		resp, err = testRoleRead(t, b, s, name)
		assertErrorIsNil(t, err)
		groups := resp.Data["azure_groups"].([]*AzureGroup)
		equal(t, "baz", groups[0].GroupName)
		equal(t, "00000000-1111-2222-3333-444444444444FAKE_GROUP-baz", groups[0].ObjectID)
		equal(t, "bam", groups[1].GroupName)
		equal(t, "a6a834a6-36c3-4575-8e2b-05095963d603FAKE_GROUP-bam", groups[1].ObjectID)
	})

	t.Run("Duplicate role name and scope", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

		var role = map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "Owner",
					"scope":  "test_scope_1"
				},
				{
					"role_name": "Owner",
					"scope":  "test_scope_1"
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
		assertErrorIsNil(t, err)
		if !resp.IsError() {
			t.Fatal("expected error response for duplicate role & scope")
		}
	})

	t.Run("Duplicate role name, different scope", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

		var role = map[string]interface{}{
			"azure_roles": compactJSON(`[
				{
					"role_name": "Owner",
					"scope":  "test_scope_1"
				},
				{
					"role_name": "Owner",
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
		assertErrorIsNil(t, err)
		if resp.IsError() {
			t.Fatalf("received unexpected error response: %v", resp.Error())
		}
	})

	t.Run("Duplicate group object ID", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

		var role = map[string]interface{}{
			"azure_groups": compactJSON(`[
				{
					"display_name": "foo",
					"object_id":  "a93d630b-c088-4e5d-801b-7cd264900e84"
				},
				{
					"display_name": "foo",
					"object_id":  "a93d630b-c088-4e5d-801b-7cd264900e84"
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
		assertErrorIsNil(t, err)
		if !resp.IsError() {
			t.Fatal("expected error response for duplicate object_id")
		}
	})

	t.Run("Role name lookup (multiple match)", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

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
		assertErrorIsNil(t, err)
		if !resp.IsError() {
			t.Fatal("expected error response")
		}
	})

	t.Run("Group name lookup (multiple match)", func(t *testing.T) {
		b, s := getTestBackendMocked(t, true)

		// if group_name=="multiple", the mock will return multiple IDs, which are not allowed
		var role = map[string]interface{}{
			"azure_groups": compactJSON(`[
				{
					"display_name": "multiple",
					"object_id": ""
				},
				{
					"display_name": "will be replaced",
					"role_id": "a6a834a6-36c3-4575-8e2b-05095963d603FAKE_GROUP-bam"
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
		assertErrorIsNil(t, err)
		if !resp.IsError() {
			t.Fatal("expected error response")
		}
	})

}

func TestRoleCreateBad(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	// missing roles and Application ID
	role := map[string]interface{}{}
	resp := testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg := "either Azure role definitions, group definitions, or an Application Object ID must be provided"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}

	// invalid roles
	role = map[string]interface{}{"azure_roles": "asdf"}
	resp = testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg = "error parsing Azure roles"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}

	// invalid roles with group membership
	role = map[string]interface{}{"azure_groups": "asdf"}
	resp = testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg = "error parsing Azure groups"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}

	// invalid roles, with application_object_id
	role = map[string]interface{}{"application_object_id": "abc", "azure_roles": "asdf"}
	resp = testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg = "error parsing Azure roles"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}

	// invalid signInAudience
	role = map[string]interface{}{"sign_in_audience": "asdfg"}
	resp = testRoleCreateBasic(t, b, s, "test_role_1", role)
	msg = "Invalid value for sign_in_audience field. Valid values are: AzureADMyOrg, AzureADMultipleOrgs, AzureADandPersonalMicrosoftAccount, PersonalMicrosoftAccount"
	if !strings.Contains(resp.Error().Error(), msg) {
		t.Fatalf("expected to find: %s, got: %s", msg, resp.Error().Error())
	}
}

// TestRolesCreate_applicationObjectID tests that a role
// can be created using the Application Object ID field.
// This test requires valid, sufficiently-privileged
// Azure credentials in env variables.
func TestRolesCreate_applicationObjectID(t *testing.T) {
	if os.Getenv("VAULT_ACC") != "1" {
		t.SkipNow()
	}

	if os.Getenv("AZURE_CLIENT_SECRET") == "" {
		t.Skip("Azure Secrets: Azure environment variables not set. Skipping.")
	}

	t.Run("service principals", func(t *testing.T) {
		t.Parallel()

		skipIfMissingEnvVars(t,
			"AZURE_SUBSCRIPTION_ID",
			"AZURE_CLIENT_ID",
			"AZURE_CLIENT_SECRET",
			"AZURE_TENANT_ID",
			"AZURE_TEST_RESOURCE_GROUP",
		)

		b := backend()
		s := new(logical.InmemStorage)
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		clientID := os.Getenv("AZURE_CLIENT_ID")
		clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		appObjectID := os.Getenv("AZURE_APPLICATION_OBJECT_ID")

		config := &logical.BackendConfig{
			Logger: logging.NewVaultLogger(log.Trace),
			System: &logical.StaticSystemView{
				DefaultLeaseTTLVal: defaultLeaseTTLHr,
				MaxLeaseTTLVal:     maxLeaseTTLHr,
			},
			StorageView: s,
		}
		err := b.Setup(context.Background(), config)
		assertErrorIsNil(t, err)

		configData := map[string]interface{}{
			"application_object_id": subscriptionID,
			"client_id":             clientID,
			"client_secret":         clientSecret,
			"tenant_id":             tenantID,
		}

		configResp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "config",
			Data:      configData,
			Storage:   s,
		})
		assertRespNoError(t, configResp, err)

		roleName := "test_role_object_id"

		roleData := map[string]interface{}{
			"application_object_id": appObjectID,
			"ttl":                   "1h",
		}

		roleResp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      fmt.Sprintf("roles/%s", roleName),
			Data:      roleData,
			Storage:   s,
		})
		assertRespNoError(t, roleResp, err)

		credsResp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      fmt.Sprintf("creds/%s", roleName),
			Storage:   s,
		})
		assertRespNoError(t, credsResp, err)

	})
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name  string
		tags  []string
		valid bool
	}{
		{
			name:  "Valid tags",
			tags:  []string{"project:vault_test", "team:engineering"},
			valid: true,
		},
		{
			name:  "Empty tags",
			tags:  []string{},
			valid: true,
		},
		{
			name:  "Duplicate tags",
			tags:  []string{"project:vault_test", "project:vault_test"},
			valid: true,
		},
		{
			name:  "Tags with whitespaces",
			tags:  []string{"environment development"},
			valid: false,
		},
		{
			name:  "Invalid long tag size (must be between 1 and 256 characters)",
			tags:  []string{"abc" + strings.Repeat("d", 256)},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateTags(tt.tags)
			if tt.valid && err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if !tt.valid && err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}

func TestRoleUpdateError(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

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
	b, s := getTestBackendMocked(t, true)

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
	assertErrorIsNil(t, err)

	exp := []string{"r1", "r2", "r3"}
	sort.Strings(resp.Data["keys"].([]string))
	equal(t, exp, resp.Data["keys"])

	// Delete a role and verify list is updated
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/r2",
		Storage:   s,
	})
	assertErrorIsNil(t, err)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   s,
	})
	assertErrorIsNil(t, err)

	exp = []string{"r1", "r3"}
	sort.Strings(resp.Data["keys"].([]string))
	equal(t, exp, resp.Data["keys"])
}

func TestRoleDelete(t *testing.T) {
	b, s := getTestBackendMocked(t, true)
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
	assertErrorIsNil(t, err)

	resp, err = testRoleRead(t, b, s, name)
	if resp != nil || err != nil {
		t.Fatalf("expected nil response and error, actual:%#v and %#v", resp, err.Error())
	}

	resp, err = testRoleRead(t, b, s, nameAlt)
	assertErrorIsNil(t, err)
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
	if data["azure_roles"] != nil {
		data["azure_roles"] = encodeJSON(data["azure_roles"])
	}
	if data["azure_groups"] != nil {
		data["azure_groups"] = encodeJSON(data["azure_groups"])
	}
	data["ttl"] = int64(data["ttl"].(time.Duration))
	data["max_ttl"] = int64(data["max_ttl"].(time.Duration))
	data["explicit_max_ttl"] = int64(data["explicit_max_ttl"].(time.Duration))
}
