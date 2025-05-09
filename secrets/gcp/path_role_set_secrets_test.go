// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/openbao/openbao-plugins/secrets/gcp/util"
	"golang.org/x/oauth2"
	"google.golang.org/api/iam/v1"
)

func TestSecrets_getRoleSetAccessToken(t *testing.T) {
	rsName := "test-gentoken"
	testGetRoleSetAccessToken(t, rsName, fmt.Sprintf("roleset/%s/token", rsName))
}

func TestSecrets_getRoleSetKey(t *testing.T) {
	rsName := "test-genkey"
	testGetRoleSetKey(t, rsName, fmt.Sprintf("roleset/%s/key", rsName))
}

func TestSecrets_roleSetBadResource(t *testing.T) {
	rsName := "test-bad-resource"
	testGetRoleSetBadResource(t, rsName, fmt.Sprintf("roleset/%s", rsName))
}

// Test deprecated path still works
func TestSecretsDeprecated_getRoleSetAccessToken(t *testing.T) {
	rsName := "test-gentoken"
	testGetRoleSetAccessToken(t, rsName, fmt.Sprintf("token/%s", rsName))
}

// Test deprecated path still works
func TestSecretsDeprecated_getRoleSetKey(t *testing.T) {
	rsName := "test-genkey"
	testGetRoleSetKey(t, rsName, fmt.Sprintf("key/%s", rsName))
}

func testGetRoleSetAccessToken(t *testing.T, rsName, path string) {
	secretType := SecretTypeAccessToken

	td := setupTest(t, "0s", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testRoleSetCreate(t, td, rsName,
		map[string]interface{}{
			"secret_type":  secretType,
			"project":      td.Project,
			"bindings":     bindsRaw,
			"token_scopes": []string{iam.CloudPlatformScope},
		})
	sa := getRoleSetAccount(t, td, rsName)

	// expect error for trying to read key from token roleset
	testGetKeyFail(t, td, fmt.Sprintf("roleset/%s/key", rsName))

	token := testGetToken(t, path, td)

	callC := oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
	)
	checkSecretPermissions(t, td, callC)

	// Cleanup: Delete role set
	testRoleSetDelete(t, td, rsName, sa.Name)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}

func testGetRoleSetKey(t *testing.T, rsName, path string) {
	secretType := SecretTypeKey

	td := setupTest(t, "0s", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testRoleSetCreate(t, td, rsName,
		map[string]interface{}{
			"secret_type": secretType,
			"project":     td.Project,
			"bindings":    bindsRaw,
		})
	sa := getRoleSetAccount(t, td, rsName)

	// expect error for trying to read token from key roleset
	testGetTokenFail(t, td, fmt.Sprintf("roleset/%s/token", rsName))

	creds, resp := testGetKey(t, path, td)
	secret := resp.Secret

	// Confirm calls with key work
	keyHttpC := oauth2.NewClient(context.Background(), creds.TokenSource)
	checkSecretPermissions(t, td, keyHttpC)

	keyName := secret.InternalData["key_name"].(string)
	if keyName == "" {
		t.Fatalf("expected internal data to include key name")
	}

	_, err = td.IamAdmin.Projects.ServiceAccounts.Keys.Get(keyName).Do()
	if err != nil {
		t.Fatalf("could not get key from given internal 'key_name': %v", err)
	}

	testRenewSecretKey(t, td, secret)
	testRevokeSecretKey(t, td, secret)

	verifyServiceAccountKeyDeleted(t, td.IamAdmin, keyName)

	// Cleanup: Delete role set
	testRoleSetDelete(t, td, rsName, sa.Name)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}

func testGetRoleSetBadResource(t *testing.T, rsName, path string) {
	secretType := SecretTypeKey

	td := setupTest(t, "0s", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, rsName)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	resp, _ := testRoleSetCreateRaw(t, td, rsName,
		map[string]interface{}{
			"secret_type": secretType,
			"project":     td.Project,
			"bindings":    bindsRaw,
		})

	if !resp.IsError() {
		t.Fatal("expected error, got none")
	}
}

func TestSecrets_GenerateKeyConfigTTL(t *testing.T) {
	secretType := SecretTypeKey
	rsName := "test-genkey"
	path := fmt.Sprintf("roleset/%s/key", rsName)

	td := setupTest(t, "1h", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testRoleSetCreate(t, td, rsName,
		map[string]interface{}{
			"secret_type": secretType,
			"project":     td.Project,
			"bindings":    bindsRaw,
		})
	sa := getRoleSetAccount(t, td, rsName)

	// expect error for trying to read token from key roleset
	testGetTokenFail(t, td, fmt.Sprintf("roleset/%s/token", rsName))

	creds, resp := testGetKey(t, path, td)
	if int(resp.Secret.LeaseTotal().Hours()) != 1 {
		t.Fatalf("expected lease duration %d, got %d", 1, int(resp.Secret.LeaseTotal().Hours()))
	}

	// Confirm calls with key work
	keyHttpC := oauth2.NewClient(context.Background(), creds.TokenSource)
	checkSecretPermissions(t, td, keyHttpC)

	keyName := resp.Secret.InternalData["key_name"].(string)
	if keyName == "" {
		t.Fatalf("expected internal data to include key name")
	}

	_, err = td.IamAdmin.Projects.ServiceAccounts.Keys.Get(keyName).Do()
	if err != nil {
		t.Fatalf("could not get key from given internal 'key_name': %v", err)
	}

	testRenewSecretKey(t, td, resp.Secret)
	testRevokeSecretKey(t, td, resp.Secret)

	verifyServiceAccountKeyDeleted(t, td.IamAdmin, keyName)

	// Cleanup: Delete role set
	testRoleSetDelete(t, td, rsName, sa.Name)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}

func TestSecrets_GenerateKeyTTLOverride(t *testing.T) {
	secretType := SecretTypeKey
	rsName := "test-genkey"

	td := setupTest(t, "1h", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testRoleSetCreate(t, td, rsName,
		map[string]interface{}{
			"secret_type": secretType,
			"project":     td.Project,
			"bindings":    bindsRaw,
		})
	sa := getRoleSetAccount(t, td, rsName)

	// expect error for trying to read token from key roleset
	testGetTokenFail(t, td, fmt.Sprintf("roleset/%s/token", rsName))

	// call the POST endpoint of /gcp/roleset/:roleset:/key with TTL
	creds, resp := testPostKey(t, td, fmt.Sprintf("roleset/%s/key", rsName), "60s")
	if int(resp.Secret.LeaseTotal().Seconds()) != 60 {
		t.Fatalf("expected lease duration %d, got %d", 60, int(resp.Secret.LeaseTotal().Seconds()))
	}

	// Confirm calls with key work
	keyHttpC := oauth2.NewClient(context.Background(), creds.TokenSource)
	checkSecretPermissions(t, td, keyHttpC)

	keyName := resp.Secret.InternalData["key_name"].(string)
	if keyName == "" {
		t.Fatalf("expected internal data to include key name")
	}

	_, err = td.IamAdmin.Projects.ServiceAccounts.Keys.Get(keyName).Do()
	if err != nil {
		t.Fatalf("could not get key from given internal 'key_name': %v", err)
	}

	testRenewSecretKey(t, td, resp.Secret)
	testRevokeSecretKey(t, td, resp.Secret)

	verifyServiceAccountKeyDeleted(t, td.IamAdmin, keyName)

	// Cleanup: Delete role set
	testRoleSetDelete(t, td, rsName, sa.Name)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}

// TestSecrets_GenerateKeyMaxTTLCheck verifies the MaxTTL is set for the
// configured backend
func TestSecrets_GenerateKeyMaxTTLCheck(t *testing.T) {
	secretType := SecretTypeKey
	rsName := "test-genkey"

	td := setupTest(t, "1h", "2h")
	defer cleanupRoleset(t, td, rsName, testRoles)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	// Create new role set
	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testRoleSetCreate(t, td, rsName,
		map[string]interface{}{
			"secret_type": secretType,
			"project":     td.Project,
			"bindings":    bindsRaw,
		})
	sa := getRoleSetAccount(t, td, rsName)

	// expect error for trying to read token from key roleset
	testGetTokenFail(t, td, fmt.Sprintf("roleset/%s/token", rsName))

	// call the POST endpoint of /gcp/roleset/:roleset/key with updated TTL
	creds, resp := testPostKey(t, td, fmt.Sprintf("roleset/%s/key", rsName), "60s")
	if int(resp.Secret.LeaseTotal().Seconds()) != 60 {
		t.Fatalf("expected lease duration %d, got %d", 60, int(resp.Secret.LeaseTotal().Seconds()))
	}

	if int(resp.Secret.LeaseOptions.MaxTTL.Hours()) != 2 {
		t.Fatalf("expected max lease %d, got %d", 2, int(resp.Secret.LeaseOptions.MaxTTL.Hours()))
	}

	// Confirm calls with key work
	keyHttpC := oauth2.NewClient(context.Background(), creds.TokenSource)
	checkSecretPermissions(t, td, keyHttpC)

	keyName := resp.Secret.InternalData["key_name"].(string)
	if keyName == "" {
		t.Fatalf("expected internal data to include key name")
	}

	_, err = td.IamAdmin.Projects.ServiceAccounts.Keys.Get(keyName).Do()
	if err != nil {
		t.Fatalf("could not get key from given internal 'key_name': %v", err)
	}

	testRenewSecretKey(t, td, resp.Secret)
	testRevokeSecretKey(t, td, resp.Secret)

	verifyServiceAccountKeyDeleted(t, td.IamAdmin, keyName)

	// Cleanup: Delete role set
	testRoleSetDelete(t, td, rsName, sa.Name)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}
