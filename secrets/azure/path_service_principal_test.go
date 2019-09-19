package azuresecrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/hashicorp/go-hclog"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fakeSubscription = "ce7d1612-67c1-4dc6-8d81-4e0a432e696b"
	fakeRoleDef1     = "a527471d-5db9-4cbe-844d-97573d3e68a3"
	fakeRoleDef2     = "458f24bf-eaa3-42aa-a2ab-14e172d0bc5e"
)

var (
	testRole = map[string]interface{}{
		"azure_roles": encodeJSON([]AzureRole{
			AzureRole{
				RoleName: "Owner",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Owner",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
			AzureRole{
				RoleName: "Contributor",
				RoleID:   "/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-Contributor",
				Scope:    "/subscriptions/ce7d1612-67c1-4dc6-8d81-4e0a432e696b",
			},
		}),
	}

	testGroupRole = map[string]interface{}{
		"azure_groups": encodeJSON([]AzureGroup{
			AzureGroup{
				GroupName: "foo",
				ObjectID:  "00000000-1111-2222-3333-444444444444FAKE_GROUP-foo",
			},
			AzureGroup{
				GroupName: "baz",
				ObjectID:  "00000000-1111-2222-3333-444444444444FAKE_GROUP-baz",
			},
		}),
	}

	testStaticSPRole = map[string]interface{}{
		"application_object_id": "00000000-0000-0000-0000-000000000000",
	}
)

func TestSPRead(t *testing.T) {
	b, s := getTestBackend(t, true)

	// verify basic cred issuance
	t.Run("Basic Role", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		nilErr(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		nilErr(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// verify password format
		_, err = uuid.ParseUUID(resp.Data["client_secret"].(string))
		nilErr(t, err)
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

		nilErr(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		nilErr(t, err)

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// verify password format
		_, err = uuid.ParseUUID(resp.Data["client_secret"].(string))
		nilErr(t, err)
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

		nilErr(t, err)

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

		nilErr(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 30*time.Second, resp.Secret.MaxTTL)
	})
}

func TestStaticSPRead(t *testing.T) {
	b, s := getTestBackend(t, true)

	// verify basic cred issuance
	t.Run("Basic", func(t *testing.T) {
		name := generateUUID()
		testRoleCreate(t, b, s, name, testStaticSPRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/" + name,
			Storage:   s,
		})

		nilErr(t, err)

		if resp.IsError() {
			t.Fatalf("expected no response error, actual:%#v", resp.Error())
		}

		// verify client_id format, and that the corresponding app actually exists
		_, err = uuid.ParseUUID(resp.Data["client_id"].(string))
		nilErr(t, err)

		keyID := resp.Secret.InternalData["key_id"].(string)
		if !strings.HasPrefix(keyID, "ffffff") {
			t.Fatalf("expected prefix 'ffffff': %s", keyID)
		}

		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)

		if !client.provider.(*mockProvider).passwordExists(keyID) {
			t.Fatalf("password was not created")
		}

		// verify password format
		_, err = uuid.ParseUUID(resp.Data["client_secret"].(string))
		nilErr(t, err)
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

		nilErr(t, err)

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

		nilErr(t, err)

		equal(t, 20*time.Second, resp.Secret.TTL)
		equal(t, 30*time.Second, resp.Secret.MaxTTL)
	})
}

func TestSPRevoke(t *testing.T) {
	b, s := getTestBackend(t, true)

	t.Run("roles", func(t *testing.T) {
		testRoleCreate(t, b, s, "test_role", testRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/test_role",
			Storage:   s,
		})

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		secret := new(logical.Secret)
		enc, err := jsonutil.EncodeJSON(resp.Secret)
		if err != nil {
			t.Fatalf("expected nil error, actual:%#v", err.Error())
		}
		jsonutil.DecodeJSON(enc, &secret)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    secret,
			Storage:   s,
		})

		nilErr(t, err)

		if resp.IsError() {
			t.Fatalf("receive response error: %v", resp.Error())
		}

		if client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application present but should have been deleted")
		}
	})

	t.Run("groups", func(t *testing.T) {
		testRoleCreate(t, b, s, "test_role", testGroupRole)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "creds/test_role",
			Storage:   s,
		})

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)

		if !client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application was not created")
		}

		// Serialize and deserialize the secret to remove typing, as will really happen.
		secret := new(logical.Secret)
		enc, err := jsonutil.EncodeJSON(resp.Secret)
		if err != nil {
			t.Fatalf("expected nil error, actual:%#v", err.Error())
		}
		jsonutil.DecodeJSON(enc, &secret)

		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    secret,
			Storage:   s,
		})

		nilErr(t, err)

		if resp.IsError() {
			t.Fatalf("receive response error: %v", resp.Error())
		}

		if client.provider.(*mockProvider).appExists(appObjID) {
			t.Fatalf("application present but should have been deleted")
		}
	})
}

func TestStaticSPRevoke(t *testing.T) {
	b, s := getTestBackend(t, true)

	testRoleCreate(t, b, s, "test_role", testStaticSPRole)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/test_role",
		Storage:   s,
	})

	keyID := resp.Secret.InternalData["key_id"].(string)
	if !strings.HasPrefix(keyID, "ffffff") {
		t.Fatalf("expected prefix 'ffffff': %s", keyID)
	}

	client, err := b.getClient(context.Background(), s)
	nilErr(t, err)

	if !client.provider.(*mockProvider).passwordExists(keyID) {
		t.Fatalf("password was not created")
	}

	// Serialize and deserialize the secret to remove typing, as will really happen.
	secret := new(logical.Secret)
	enc, err := jsonutil.EncodeJSON(resp.Secret)
	if err != nil {
		t.Fatalf("expected nil error, actual:%#v", err.Error())
	}
	json.Unmarshal(enc, &secret)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.RevokeOperation,
		Secret:    secret,
		Storage:   s,
	})

	nilErr(t, err)

	if resp.IsError() {
		t.Fatalf("receive response error: %v", resp.Error())
	}

	if client.provider.(*mockProvider).passwordExists(keyID) {
		t.Fatalf("password present but should have been deleted")
	}
}

func TestSPReadMissingRole(t *testing.T) {
	b, s := getTestBackend(t, true)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/some_role_that_doesnt_exist",
		Storage:   s,
	})

	nilErr(t, err)

	if !resp.IsError() {
		t.Fatal("expected a response error")
	}
}

func TestCredentialReadProviderError(t *testing.T) {
	b, s := getTestBackend(t, true)

	testRoleCreate(t, b, s, "test_role", testRole)

	client, err := b.getClient(context.Background(), s)
	nilErr(t, err)
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

// TestCredentialInteg is an integration test against the live Azure service. It requires
// valid, sufficiently-privileged Azure credentials in env variables.
func TestCredentialInteg(t *testing.T) {
	if os.Getenv("VAULT_ACC") != "1" {
		t.SkipNow()
	}

	if os.Getenv("AZURE_CLIENT_SECRET") == "" {
		t.Skip("Azure Secrets: Azure environment variables not set. Skipping.")
	}

	t.Run("SP", func(t *testing.T) {
		t.Parallel()

		b := backend()
		s := new(logical.InmemStorage)
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

		config := &logical.BackendConfig{
			Logger: logging.NewVaultLogger(log.Trace),
			System: &logical.StaticSystemView{
				DefaultLeaseTTLVal: defaultLeaseTTLHr,
				MaxLeaseTTLVal:     maxLeaseTTLHr,
			},
			StorageView: s,
		}
		err := b.Setup(context.Background(), config)
		nilErr(t, err)

		// Add a Vault role that will provide creds with Azure "Reader" permissions
		// Resources groups "vault-azure-secrets-test1" and "vault-azure-secrets-test2"
		// should already exist in the test infrastructure. (The test can be simplified
		// to just use scope "/subscriptions/%s" if need be.)
		rolename := "test_role"
		role := map[string]interface{}{
			"azure_roles": fmt.Sprintf(`[
			{
				"role_name": "Reader",
				"scope":  "/subscriptions/%s/resourceGroups/vault-azure-secrets-test1"
			},
			{
				"role_name": "Reader",
				"scope":  "/subscriptions/%s/resourceGroups/vault-azure-secrets-test2"
			}]`, subscriptionID, subscriptionID),
		}
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      fmt.Sprintf("roles/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		// Request credentials
		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      fmt.Sprintf("creds/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		appID := resp.Data["client_id"].(string)

		// Use the underlying provider to access clients directly for testing
		client, err := b.getClient(context.Background(), s)
		nilErr(t, err)
		provider := client.provider.(*provider)

		// recover the SP Object ID, which is not used by the application but
		// is helpful for verification testing
		spList, err := provider.spClient.List(context.Background(), "")
		nilErr(t, err)

		var spObjID string
		for spList.NotDone() {
			for _, v := range spList.Values() {
				if to.String(v.AppID) == appID {
					spObjID = to.String(v.ObjectID)
					goto FOUND
				}
			}
			spList.Next()
		}
		t.Fatal("Couldn't find SP Object ID")

	FOUND:
		// verify the new SP can be accessed
		_, err = provider.spClient.Get(context.Background(), spObjID)
		if err != nil {
			t.Fatalf("Expected nil error on GET of new SP, got: %#v", err)
		}

		// Verify that the role assignments were created. Get the assignment
		// info from Azure and verify it matches the Reader role.
		raIDs := resp.Secret.InternalData["role_assignment_ids"].([]string)
		equal(t, 2, len(raIDs))

		ra, err := provider.raClient.GetByID(context.Background(), raIDs[0])
		nilErr(t, err)

		roleDefs, err := client.provider.ListRoles(context.Background(), fmt.Sprintf("subscriptions/%s", subscriptionID), "")
		nilErr(t, err)

		defID := *ra.RoleAssignmentPropertiesWithScope.RoleDefinitionID
		found := false
		for _, def := range roleDefs {
			if *def.ID == defID && *def.RoleName == "Reader" {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("'Reader' role assignment not found")
		}

		// Revoke the Service Principal by sending back the secret we just
		// received, with a little type tweaking to make it work.
		resp.Secret.InternalData["role_assignment_ids"] = []interface{}{
			resp.Secret.InternalData["role_assignment_ids"].([]string)[0],
		}

		req := &logical.Request{
			Secret:  resp.Secret,
			Storage: s,
		}

		b.spRevoke(context.Background(), req, nil)

		// Verify that SP get is an error after delete. Expected there
		// to be a delay and that this step would take some time/retries,
		// but that seems not to be the case.
		_, err = provider.spClient.Get(context.Background(), spObjID)

		if err == nil {
			t.Fatal("Expected error reading deleted SP")
		}
	})

	t.Run("Static SP", func(t *testing.T) {
		t.Parallel()

		b := backend()
		s := new(logical.InmemStorage)

		config := &logical.BackendConfig{
			Logger: logging.NewVaultLogger(log.Trace),
			System: &logical.StaticSystemView{
				DefaultLeaseTTLVal: defaultLeaseTTLHr,
				MaxLeaseTTLVal:     maxLeaseTTLHr,
			},
			StorageView: s,
		}
		err := b.Setup(context.Background(), config)
		nilErr(t, err)

		// Add a Vault role that will provide creds with Azure "Reader" permissions
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

		rolename := "test_role"
		role := map[string]interface{}{
			"azure_roles": fmt.Sprintf(`[{
			"role_name": "Reader",
			"scope":  "/subscriptions/%s"
		}]`, subscriptionID),
		}
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      fmt.Sprintf("roles/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		// Request credentials
		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      fmt.Sprintf("creds/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		origResp := resp

		appObjID := resp.Secret.InternalData["app_object_id"].(string)
		appID := resp.Data["client_id"].(string)

		// Create a new role that will add passwords to the previously
		// created application when creds are requested.

		rolename = "test_role2"
		role = map[string]interface{}{
			"application_object_id": appObjID,
		}
		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      fmt.Sprintf("roles/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		// Request credentials
		resp, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      fmt.Sprintf("creds/%s", rolename),
			Data:      role,
			Storage:   s,
		})
		nilErr(t, err)

		if resp != nil && resp.IsError() {
			t.Fatal(resp.Error())
		}

		// Test the added password by creating a new Azure provider with these
		// creds and attempting an operation with it.
		clientConfig := azureConfig{}

		settings, err := b.getClientSettings(context.Background(), &clientConfig)
		if err != nil {
			t.Fatal(err)
		}

		settings.ClientID = appID
		settings.ClientSecret = resp.Data["client_secret"].(string)

		success := false

		// The new app may not be propagated immediately, so retry for ~30s.
		for i := 0; i < 6; i++ {
			// New credentials are only tested during an actual operation, not provider creation.
			// This step should never fail.
			p, err := newAzureProvider(settings)
			if err != nil {
				t.Fatal(err)
			}

			_, err = p.GetApplication(context.Background(), appObjID)
			if err == nil {
				success = true
				break
			}
			time.Sleep(5 * time.Second)
		}

		if !success {
			t.Fatal("unable to validate with credentials. Last error: " + err.Error())
		}

		// Revoke the Service Principal by sending back the secret we just
		// received, with a little type tweaking to make it work.
		origResp.Secret.InternalData["role_assignment_ids"] = []interface{}{
			origResp.Secret.InternalData["role_assignment_ids"].([]string)[0],
		}

		req := &logical.Request{
			Secret:  origResp.Secret,
			Storage: s,
		}

		_, err = b.spRevoke(context.Background(), req, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}
