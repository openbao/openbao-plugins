// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/logging"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/openbao/openbao-plugins/secrets/azure/api"
)

const (
	PasswordLength = 36
)

var (
	errDoesNotExist = "does not exist"
	testRole        = map[string]interface{}{
		"azure_roles": encodeJSON([]AzureRole{
			{
				RoleName: "Owner",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
			{
				RoleName: "Contributor",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
		}),
	}

	testPermanentlyDeleteRole = map[string]interface{}{
		"azure_roles": encodeJSON([]AzureRole{
			{
				RoleName: "Owner",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
			{
				RoleName: "Contributor",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
		}),
		"permanently_delete": true,
	}

	testGroupRole = map[string]interface{}{
		"azure_groups": encodeJSON([]AzureGroup{
			{
				GroupName: "foo",
				ObjectID:  "00000000-1111-2222-3333-444444444444FAKE_GROUP-foo",
			},
			{
				GroupName: "baz",
				ObjectID:  "00000000-1111-2222-3333-444444444444FAKE_GROUP-baz",
			},
		}),
	}

	testStaticSPAppObjID = "00000000-0000-0000-0000-000000000000"

	testStaticSPRole = map[string]interface{}{
		"application_object_id": testStaticSPAppObjID,
	}

	testPersistentRole = map[string]interface{}{
		"azure_roles": encodeJSON([]AzureRole{
			{
				RoleName: "Owner",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
			{
				RoleName: "Contributor",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
		}),
		"persist_app": true,
	}
)

// TestSP_WAL_Cleanup tests that any Service Principal that gets created, but
// fails to have roles associated with it, gets cleaned up by the periodic WAL
// function.
func TestSP_WAL_Cleanup(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	mp := newMockProvider()
	// ensure timeout is exceeds the context deadline setup below
	mp.(*mockProvider).ctxTimeout = 6 * time.Second
	b.getProvider = func(context.Context, hclog.Logger, logical.SystemView, *clientSettings) (AzureProvider, error) {
		return mp, nil
	}

	// verify basic cred issuance
	t.Run("Role assign fail", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		// create a short timeout to short-circuit the retry process and trigger the
		// deadline error
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Fatalf("expected deadline error, but got '%s'", err.Error())
		}
		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		assertEmptyWAL(t, b, mp, s)
	})
}

func assertEmptyWAL(t *testing.T, b *azureSecretBackend, emp AzureProvider, s logical.Storage) {
	t.Helper()

	wal, err := framework.ListWAL(context.Background(), s)
	if err != nil {
		t.Fatalf("error listing wal: %s", err)
	}
	req := &logical.Request{
		Storage: s,
	}

	// loop of WAL entries and trigger the rollback method for each, simulating
	// Vault's rollback mechanism
	for _, v := range wal {
		ctx := context.Background()
		entry, err := framework.GetWAL(ctx, s, v)
		if err != nil {
			t.Fatal(err)
		}

		switch entry.Kind {
		case walAppKey:
			// Decode the WAL data
			var app walApp
			d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
				Result:     &app,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = d.Decode(entry.Data)
			if err != nil {
				t.Fatal(err)
			}

			_, err = emp.GetApplication(context.Background(), app.AppObjID)
			if err != nil {
				t.Fatalf("expected to find application (%s), but wasn't found", app.AppObjID)
			}

			err = b.walRollback(ctx, req, entry.Kind, entry.Data)
			if err != nil {
				t.Fatal(err)
			}
			if err := framework.DeleteWAL(ctx, s, v); err != nil {
				t.Fatal(err)
			}

			_, err = emp.GetApplication(context.Background(), app.AppObjID)
			if err == nil {
				t.Fatalf("expected error getting application")
			}
		case walAppRoleAssignment:
			// Decode the WAL data
			d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
				Result:     &entry,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = d.Decode(entry.Data)
			if err != nil {
				t.Fatal(err)
			}

			err = b.walRollback(ctx, req, entry.Kind, entry.Data)
			if err != nil {
				t.Fatal(err)
			}

			if err := framework.DeleteWAL(ctx, s, v); err != nil {
				t.Fatal(err)
			}
		}

	}
}

func TestSPRead(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	// verify basic cred issuance
	t.Run("Basic Role", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		assertErrorIsNil(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		assertClientSecret(t, resp.Data)
	})

	// verify basic cred issuance using group membership
	t.Run("Basic Group", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testGroupRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		assertErrorIsNil(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		assertClientSecret(t, resp.Data)
	})

	// verify role TTLs are reflected in secret
	t.Run("TTLs", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 0*time.Second, resp.Secret.TTL)
		equal(t, 0*time.Second, resp.Secret.MaxTTL)

		roleUpdate := map[string]interface{}{
			"ttl":     20,
			"max_ttl": 30,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 30*time.Second, resp.Secret.MaxTTL)

		roleUpdate = map[string]interface{}{
			"ttl":              20,
			"explicit_max_ttl": 60,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 60*time.Second, resp.Secret.MaxTTL)
	})
}

func TestStaticSPRead(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	// verify basic cred issuance
	t.Run("Basic", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testStaticSPRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		assertErrorIsNil(t, err)

		keyID := resp.Secret.InternalData["key_id"].(string)
		if len(keyID) == 0 {
			t.Fatalf("expected keyId to not be empty")
		}

		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).passwordExists(keyID) {
			t.Fatalf("password was not created")
		}

		assertClientSecret(t, resp.Data)
	})

	// verify role TTLs are reflected in secret
	t.Run("TTLs", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testStaticSPRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 0*time.Second, resp.Secret.TTL)
		equal(t, 0*time.Second, resp.Secret.MaxTTL)

		roleUpdate := map[string]interface{}{
			"ttl":     20,
			"max_ttl": 30,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 30*time.Second, resp.Secret.MaxTTL)

		roleUpdate = map[string]interface{}{
			"ttl":              20,
			"explicit_max_ttl": 60,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 60*time.Second, resp.Secret.MaxTTL)
	})
}

func TestPersistentAppSPRead(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	// verify basic cred issuance
	t.Run("Basic", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testPersistentRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		assertErrorIsNil(t, err)

		keyID := resp.Secret.InternalData["key_id"].(string)
		if len(keyID) == 0 {
			t.Fatalf("expected keyId to not be empty")
		}

		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).passwordExists(keyID) {
			t.Fatalf("password was not created")
		}

		assertClientSecret(t, resp.Data)
	})

	// verify role TTLs are reflected in secret
	t.Run("TTLs", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testPersistentRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 0*time.Second, resp.Secret.TTL)
		equal(t, 0*time.Second, resp.Secret.MaxTTL)

		roleUpdate := map[string]interface{}{
			"ttl":     20,
			"max_ttl": 30,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 30*time.Second, resp.Secret.MaxTTL)

		roleUpdate = map[string]interface{}{
			"ttl":              20,
			"explicit_max_ttl": 60,
		}
		testRoleCreate(t, b, s, name, roleUpdate)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 60*time.Second, resp.Secret.MaxTTL)
	})
}

func TestSPRevoke(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	t.Run("roles", func(t *testing.T) {
		testRoleCreate(t, b, s, "test_role", testRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/test_role",
			Storage:   s,
		})
		assertErrorIsNil(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		fakeSaveLoad(resp.Secret)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    resp.Secret,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("receive response error: %v", resp.Error())
		}

		if client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application present but should have been deleted")
		}

		if !client.provider.(*mockProvider).deletedObjectExists(appObjID) {
			t.Fatalf("application is missing from deleted objects but should have been 'soft deleted'")
		}
	})

	t.Run("permanently_delete_roles", func(t *testing.T) {
		testRoleCreate(t, b, s, "test_role", testPermanentlyDeleteRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/test_role",
			Storage:   s,
		})
		assertErrorIsNil(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		fakeSaveLoad(resp.Secret)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    resp.Secret,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("receive response error: %v", resp.Error())
		}

		if client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application present but should have been deleted")
		}

		if client.provider.(*mockProvider).deletedObjectExists(appObjID) {
			t.Fatalf("application is present in deleted objects but should have been permanently deleted")
		}
	})

	t.Run("groups", func(t *testing.T) {
		testRoleCreate(t, b, s, "test_role", testGroupRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/test_role",
			Storage:   s,
		})
		assertErrorIsNil(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		fakeSaveLoad(resp.Secret)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    resp.Secret,
			Storage:   s,
		})

		assertErrorIsNil(t, err)

		if resp.IsError() {
			t.Fatalf("receive response error: %v", resp.Error())
		}

		if len(resp.Warnings) > 0 {
			t.Fatalf("response contains warnings but should not have")
		}

		if client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application present but should have been deleted")
		}
	})
}

func TestStaticSPRevoke(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	testRoleCreate(t, b, s, "test_role", testStaticSPRole)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/test_role",
		Storage:   s,
	})
	assertErrorIsNil(t, err)

	keyID := resp.Secret.InternalData["key_id"].(string)
	if len(keyID) == 0 {
		t.Fatalf("expected keyId to not be empty")
	}

	client, err := b.getClient(context.Background(), s)
	assertErrorIsNil(t, err)

	if !client.provider.(*mockProvider).passwordExists(keyID) {
		t.Fatalf("password was not created")
	}

	// Serialize and deserialize the secret to remove typing, as will really happen.
	fakeSaveLoad(resp.Secret)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.RevokeOperation,
		Secret:    resp.Secret,
		Storage:   s,
	})

	assertErrorIsNil(t, err)

	if resp.IsError() {
		t.Fatalf("receive response error: %v", resp.Error())
	}

	if client.provider.(*mockProvider).passwordExists(keyID) {
		t.Fatalf("password present but should have been deleted")
	}
}

func TestSPReadMissingRole(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/some_role_that_doesnt_exist",
		Storage:   s,
	})

	assertErrorIsNil(t, err)

	if !resp.IsError() {
		t.Fatal("expected a response error")
	}
}

func TestCredentialReadProviderError(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	testRoleCreate(t, b, s, "test_role", testRole)

	client, err := b.getClient(context.Background(), s)
	assertErrorIsNil(t, err)
	client.provider.(*mockProvider).failNextCreateApplication = true

	_, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/test_role",
		Storage:   s,
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestRoleAssignmentWALRollback tests rolling back any
// role assignments that may have taken place prior to
// a subsequent failure resulting in the need to rollback
// an App or SP. This test requires valid, sufficiently-privileged
// Azure credentials in env variables.
func TestRoleAssignmentWALRollback(t *testing.T) {
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
		resourceGroup := os.Getenv("AZURE_TEST_RESOURCE_GROUP")

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
			"subscription_id": subscriptionID,
			"client_id":       clientID,
			"client_secret":   clientSecret,
			"tenant_id":       tenantID,
		}

		configResp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "config",
			Data:      configData,
			Storage:   s,
		})
		assertRespNoError(t, configResp, err)

		roleName := "test_role_rawalrollback"

		roleData := map[string]interface{}{
			"azure_roles": fmt.Sprintf(`[
			{
				"role_name": "Storage Blob Data Owner",
				"scope":  "/subscriptions/%s/resourceGroups/%s"
			},
			{
				"role_name": "Reader",
				"scope":  "/subscriptions/%s/resourceGroups/%s"
			}]`, subscriptionID, resourceGroup, subscriptionID, resourceGroup),
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

		appID := credsResp.Data["client_id"].(string)

		// Use the underlying provider to access clients directly for testing
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)
		provider := client.provider.(*provider)
		spObjID := findServicePrincipalID(t, provider.spClient, appID)

		assertServicePrincipalExistence(t, provider.spClient, spObjID, true)

		// Verify that the role assignments were created. Get the assignment
		// info from Azure and verify it matches the Reader role.
		raIDs := credsResp.Secret.InternalData["role_assignment_ids"].([]string)
		equal(t, 2, len(raIDs))

		ra, err := provider.raClient.GetByID(context.Background(), raIDs[0], nil)
		assertErrorIsNil(t, err)

		roleDefs, err := provider.ListRoleDefinitions(context.Background(), fmt.Sprintf("subscriptions/%s", subscriptionID), "")
		assertErrorIsNil(t, err)

		defID := *ra.Properties.RoleDefinitionID
		found := false
		for _, def := range roleDefs {
			if *def.ID == defID && *def.Properties.RoleName == "Storage Blob Data Owner" {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("'Storage Blob Data Owner' role assignment not found")
		}

		// Parse the assignment IDs
		var assignmentIDs []string
		for _, raID := range raIDs {
			t := strings.Split(raID, "/")
			tRa := t[len(t)-1]
			assignmentIDs = append(assignmentIDs, strings.Replace(tRa, " ", "", -1))
		}

		// Remove one of the RA IDs to simulate a failure to assign a role
		if err := client.unassignRoles(context.Background(), []string{raIDs[0]}); err != nil {
			t.Fatalf("error unassigning Role: %s", err.Error())
		}

		rEntry, err := s.Get(context.Background(), fmt.Sprintf("%s/%s", "roles", roleName))
		if err != nil {
			t.Fatalf("error getting role from storage: %s", err.Error())
		}

		if rEntry == nil {
			t.Fatalf("role entry was nil: %s", err.Error())
		}

		// Decode returned Role Entry
		role := new(roleEntry)
		if err := rEntry.DecodeJSON(role); err != nil {
			t.Fatalf("unable to decode role entry: %s", err.Error())
		}

		// Manually Create Role Assignment WAL
		rWALID, err := framework.PutWAL(context.Background(), s, walAppRoleAssignment, &walAppRoleAssign{
			SpID:          spObjID,
			AssignmentIDs: assignmentIDs,
			AzureRoles:    role.AzureRoles,
			Expiration:    time.Now().Add(maxWALAge),
		})
		if err != nil {
			t.Fatalf("error creating role assignment WAL: %s", err.Error())
		}

		// Retrieve WAL
		entry, err := framework.GetWAL(context.Background(), s, rWALID)
		if err != nil {
			t.Fatalf("error retrieving role assignment WAL: %s", err.Error())
		}

		// Decode the WAL data
		var appRoleAssign walAppRoleAssign
		d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
			Result:     &appRoleAssign,
		})
		if err != nil {
			t.Fatalf("error decoding WAL data: %s", err.Error())
		}
		err = d.Decode(entry.Data)
		if err != nil {
			t.Fatalf("error decoding WAL data: %s", err.Error())
		}

		req := &logical.Request{
			Storage: s,
		}

		// Initiate Role Assignment Rollback
		err = b.walRollback(context.Background(), req, entry.Kind, entry.Data)
		if err != nil {
			t.Fatalf("error rolling back WAL: %s", err.Error())
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		fakeSaveLoad(credsResp.Secret)

		// Revoke the Service Principal by sending back the secret we just received
		req = &logical.Request{
			Secret:  credsResp.Secret,
			Storage: s,
		}

		_, err = b.spRevoke(context.Background(), req, nil)
		if err != nil {
			t.Fatalf("error revoking service principal: %s", err.Error())
		}

		// Verify that SP get is an error after delete. Expected there
		// to be a delay and that this step would take some time/retries,
		// but that seems not to be the case.
		assertServicePrincipalExistence(t, provider.spClient, spObjID, false)
	})
}

// TestUnassignRoleFailures ensures that the plugin can
// cleanly resolve any errors received when unassigning roles
func TestUnassignRoleFailures(t *testing.T) {
	b, s := getTestBackendMocked(t, true)

	mp := newMockProvider()
	mp.(*mockProvider).failUnassignRoles = true
	mp.(*mockProvider).unassignRolesFailureParams = failureParams{
		expectError: true,
	}
	b.getProvider = func(context.Context, hclog.Logger, logical.SystemView, *clientSettings) (AzureProvider, error) {
		return mp, nil
	}

	c, err := b.getClient(context.Background(), s)
	if err != nil {
		t.Fatalf("error getting client: %s", err.Error())
	}

	// verify error is received
	t.Run("Role unassign fail", func(t *testing.T) {
		testAssignmentIDs := []string{"test-1", "test-2"}
		// Remove one of the RA IDs to simulate a failure to assign a role
		if err := c.unassignRoles(context.Background(), testAssignmentIDs); err == nil {
			t.Fatalf("expected error but got nil")
		}
	})

	mp.(*mockProvider).unassignRolesFailureParams = failureParams{
		expectError: false,
		statusCode:  204,
	}

	// verify the error is properly handled and ignored in 204 case
	t.Run("Role unassign error handled for 204", func(t *testing.T) {
		testAssignmentIDs := []string{"test-1", "test-2"}
		// Remove one of the RA IDs to simulate a failure to assign a role
		if err := c.unassignRoles(context.Background(), testAssignmentIDs); err != nil {
			t.Fatalf("error unassigning Role: %s", err.Error())
		}
	})

	mp.(*mockProvider).unassignRolesFailureParams = failureParams{
		expectError: false,
		statusCode:  404,
	}

	// verify the error is properly handled and ignored in 404 case
	t.Run("Role unassign error handled for 404", func(t *testing.T) {
		testAssignmentIDs := []string{"test-1", "test-2"}
		// Remove one of the RA IDs to simulate a failure to assign a role
		if err := c.unassignRoles(context.Background(), testAssignmentIDs); err != nil {
			t.Fatalf("error unassigning Role: %s", err.Error())
		}
	})

}

// This is an integration test against the live Azure service. It requires
// valid, sufficiently-privileged Azure credentials in env variables.
func TestCredentialInteg_msgraph(t *testing.T) {
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
			"AZURE_GROUP_NAME",
			"AZURE_TEST_RESOURCE_GROUP",
		)

		b := backend()
		s := new(logical.InmemStorage)
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		resourceGroup := os.Getenv("AZURE_TEST_RESOURCE_GROUP")
		clientID := os.Getenv("AZURE_CLIENT_ID")
		clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		groupName := os.Getenv("AZURE_GROUP_NAME")

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
			"subscription_id": subscriptionID,
			"client_id":       clientID,
			"client_secret":   clientSecret,
			"tenant_id":       tenantID,
		}

		configResp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "config",
			Data:      configData,
			Storage:   s,
		})
		assertRespNoError(t, configResp, err)

		roleName := "test_role_msgraph"

		roleData := map[string]interface{}{
			"azure_roles": fmt.Sprintf(`[
			{
				"role_name": "Storage Blob Data Owner",
				"scope":  "/subscriptions/%s/resourceGroups/%s"
			},
			{
				"role_name": "Reader",
				"scope":  "/subscriptions/%s/resourceGroups/%s"
			}]`, subscriptionID, resourceGroup, subscriptionID, resourceGroup),
			"azure_groups": fmt.Sprintf(`[
			{
				"group_name": "%s"
			}]`, groupName),
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

		appID := credsResp.Data["client_id"].(string)

		// Use the underlying provider to access clients directly for testing
		client, err := b.getClient(context.Background(), s)
		assertErrorIsNil(t, err)
		provider := client.provider.(*provider)
		spObjID := findServicePrincipalID(t, provider.spClient, appID)

		assertServicePrincipalExistence(t, provider.spClient, spObjID, true)

		// Verify that the role assignments were created. Get the assignment
		// info from Azure and verify it matches the Reader role.
		raIDs := credsResp.Secret.InternalData["role_assignment_ids"].([]string)
		equal(t, 2, len(raIDs))

		ra, err := provider.raClient.GetByID(context.Background(), raIDs[0], nil)
		assertErrorIsNil(t, err)

		roleDefs, err := provider.ListRoleDefinitions(context.Background(), fmt.Sprintf("subscriptions/%s", subscriptionID), "")
		assertErrorIsNil(t, err)

		defID := *ra.Properties.RoleDefinitionID
		found := false
		for _, def := range roleDefs {
			if *def.ID == defID && *def.Properties.RoleName == "Storage Blob Data Owner" {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("'Storage Blob Data Owner' role assignment not found")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		fakeSaveLoad(credsResp.Secret)

		// Revoke the Service Principal by sending back the secret we just received
		req := &logical.Request{
			Secret:  credsResp.Secret,
			Storage: s,
		}

		_, err = b.spRevoke(context.Background(), req, nil)
		if err != nil {
			t.Fatalf("error revoking service principal: %s", err.Error())
		}

		// Verify that SP get is an error after delete. Expected there
		// to be a delay and that this step would take some time/retries,
		// but that seems not to be the case.
		assertServicePrincipalExistence(t, provider.spClient, spObjID, false)
	})
}

func skipIfMissingEnvVars(t *testing.T, envVars ...string) {
	t.Helper()
	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			t.Skipf("Missing env variable: [%s] - skipping test", envVar)
		}
	}
}

func assertClientSecret(tb testing.TB, data map[string]interface{}) {
	assertKeyExists(tb, data, "client_secret")
	actualPassword, ok := data["client_secret"].(string)
	if !ok {
		tb.Fatalf("client_secret is not a string")
	}
	if len(actualPassword) != PasswordLength {
		tb.Fatalf("client_secret is not the correct length: expected %d but was %d", PasswordLength, len(actualPassword))
	}
}

func findServicePrincipalID(t *testing.T, client api.ServicePrincipalClient, appID string) (spID string) {
	t.Helper()

	switch spClient := client.(type) {
	case *api.MSGraphClient:
		pathVals := &url.Values{}
		pathVals.Set("$filter", fmt.Sprintf("appId eq '%s'", appID))

		spList, err := spClient.ListServicePrincipals(context.Background(), appID)
		assertErrorIsNil(t, err)

		if len(spList) == 0 {
			t.Fatalf("Failed to find service principals from application ID")
		}

		for _, sp := range spList {
			if sp.AppID == appID {
				return sp.ID
			}
		}
	default:
		t.Fatalf("Unrecognized service principal client type: %T", spClient)
	}

	t.Fatalf("Failed to find service principal with application ID: %s", appID)
	return "" // Because compilers
}

func assertServicePrincipalExistence(t *testing.T, client api.ServicePrincipalClient, spID string, exists bool) {
	t.Helper()

	switch spClient := client.(type) {
	case *api.MSGraphClient:
		sp, err := spClient.GetServicePrincipalByID(context.Background(), spID)
		if exists {
			assertErrorIsNil(t, err)

			if sp.ID == "" {
				t.Fatalf("Failed to find service principal")
			}
		} else {
			if !strings.Contains(err.Error(), errDoesNotExist) || sp.ID != "" {
				t.Fatalf("Found service principal when it shouldn't exist")
			}
		}
	default:
		t.Fatalf("Unrecognized service principal client type: %T", spClient)
	}
}

func assertRespNoError(t *testing.T, resp *logical.Response, err error) {
	t.Helper()

	assertErrorIsNil(t, err)

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
}
