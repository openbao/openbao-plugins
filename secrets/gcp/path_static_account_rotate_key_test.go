// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcp

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/openbao/openbao-plugins/secrets/gcp/util"
	"github.com/openbao/openbao/sdk/v2/logical"
	"golang.org/x/oauth2"
	"google.golang.org/api/iam/v1"
)

func TestStatic_Rotate(t *testing.T) {
	staticName := "test-static-rotate"
	secretType := SecretTypeAccessToken

	td := setupTest(t, "0s", "2h")
	defer cleanupStatic(t, td, staticName, testRoles)

	sa := createStaticAccount(t, td, staticName)
	defer deleteStaticAccount(t, td, sa)

	projRes := fmt.Sprintf(testProjectResourceTemplate, td.Project)

	expectedBinds := ResourceBindings{projRes: testRoles}
	bindsRaw, err := util.BindingsHCL(expectedBinds)
	if err != nil {
		t.Fatalf("unable to convert resource bindings to HCL string: %v", err)
	}
	testStaticCreate(t, td, staticName,
		map[string]interface{}{
			"service_account_email": sa.Email,
			"token_scopes":          []string{iam.CloudPlatformScope},
			"secret_type":           secretType,
			"bindings":              bindsRaw,
		})

	// expect error for trying to read key from token
	testGetKeyFail(t, td, fmt.Sprintf("%s/%s/key", staticAccountPathPrefix, staticName))

	// Obtain current keys
	oldKeys := getServiceAccountKeys(t, td, sa.Name)

	// Get token and check
	token := testGetToken(t, fmt.Sprintf("%s/%s/token", staticAccountPathPrefix, staticName), td)
	callC := oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
	)
	checkSecretPermissions(t, td, callC)

	// Rotate key
	resp, err := td.B.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      fmt.Sprintf("%s/%s/rotate-key", staticAccountPathPrefix, staticName),
		Data:      map[string]interface{}{},
		Storage:   td.S,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	// Get new keys
	newKeys := getServiceAccountKeys(t, td, sa.Name)

	// Check that keys are actually rotated
	if reflect.DeepEqual(oldKeys, newKeys) {
		t.Fatal("expected keys to have been rotated, but they were not")
	}

	// Test token still works
	token = testGetToken(t, fmt.Sprintf("%s/%s/token", staticAccountPathPrefix, staticName), td)
	callC = oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
	)
	checkSecretPermissions(t, td, callC)

	// Cleanup
	testStaticDelete(t, td, staticName)
	verifyProjectBindingsRemoved(t, td, sa.Email, testRoles)
}
