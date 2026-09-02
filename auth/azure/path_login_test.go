// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/openbao/openbao/sdk/v2/helper/policyutil"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"

	"github.com/openbao/openbao-plugins/auth/azure/client"
)

func TestLogin_Acceptance(t *testing.T) {
	skipIfMissingEnvVars(t,
		"TENANT_ID",
		"CLIENT_ID",
		"CLIENT_SECRET",
		"SUBSCRIPTION_ID",
		"RESOURCE_GROUP_NAME",
		"ACCESS_TOKEN_JWT",
		"VM_NAME",
	)

	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	subscriptionID := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	accessTokenJWT := os.Getenv("ACCESS_TOKEN_JWT")
	vmName := os.Getenv("VM_NAME")

	b, s := getTestBackend(t)
	configData := map[string]interface{}{
		"resource":      "https://management.azure.com/",
		"tenant_id":     tenantID,
		"client_id":     clientID,
		"client_secret": clientSecret,
	}
	if _, err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatal(err)
	}

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                   roleName,
		"policies":               []string{"dev", "prod"},
		"bound_subscription_ids": subscriptionID,
		"bound_resource_groups":  resourceGroupName,
	}
	testRoleCreate(t, b, s, roleData)

	loginData := map[string]interface{}{
		"role":                roleName,
		"subscription_id":     subscriptionID,
		"resource_group_name": resourceGroupName,
		"vm_name":             vmName,
	}
	if err := testLoginWithJWT(t, b, s, accessTokenJWT, loginData, roleData); err != nil {
		t.Fatal(err)
	}
}

func skipIfMissingEnvVars(t *testing.T, envVars ...string) {
	t.Helper()
	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			t.Skipf("Missing env variable: [%s] - skipping test", envVar)
		}
	}
}

func TestResolveRole(t *testing.T) {
	b, storage := getTestBackend(t)
	role := "testrole"

	roleData := map[string]interface{}{
		"name":                        role,
		"policies":                    []string{"dev", "prod"},
		"bound_service_principal_ids": []string{"*"},
	}
	testRoleCreate(t, b, storage, roleData)

	loginData := map[string]interface{}{
		"role": role,
	}
	loginReq := &logical.Request{
		Operation: logical.ResolveRoleOperation,
		Path:      "login",
		Storage:   storage,
		Data:      loginData,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	}

	resp, err := b.HandleRequest(context.Background(), loginReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v", err, resp)
	}

	if resp.Data["role"] != role {
		t.Fatalf("Role was not as expected. Expected %s, received %s", role, resp.Data["role"])
	}
}

func TestResolveRole_RoleDoesNotExist(t *testing.T) {
	b, storage := getTestBackend(t)
	role := "testrole"

	loginData := map[string]interface{}{
		"role": role,
	}
	loginReq := &logical.Request{
		Operation: logical.ResolveRoleOperation,
		Path:      "login",
		Storage:   storage,
		Data:      loginData,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	}

	resp, err := b.HandleRequest(context.Background(), loginReq)
	if resp == nil && !resp.IsError() {
		t.Fatalf("Response was not an error: err:%v resp:%#v", err, resp)
	}

	errString, ok := resp.Data["error"].(string)
	if !ok {
		t.Fatal("Error not part of response.")
	}

	if !strings.Contains(errString, "invalid role name") {
		t.Fatalf("Error was not due to invalid role name. Error: %s", errString)
	}
}

func TestLogin(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                        roleName,
		"policies":                    []string{"dev", "prod"},
		"bound_service_principal_ids": []string{"*"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	claims["nbf"] = time.Now().Add(60 * time.Second).Unix()
	testLoginFailure(t, b, s, loginData, claims, roleData)

	claims["nbf"] = time.Now().Add(-60 * time.Second).Unix()
	claims["exp"] = time.Now().Add(-60 * time.Second).Unix()
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_ManagedIdentity(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "eb936495-7356-4a35-af3e-ea68af201f0c"
	resourceGroupName := "azure-func-rg"
	resourceID := "/subscriptions/eb936495-7356-4a35-af3e-ea68af201f0c/resourceGroups/azure-func-rg/providers/Microsoft.Web/sites/my-azure-func"
	roleName := "test-role"

	// setup test response functions that mock the client GetByID response
	nilIdentityRespFunc := func(_ string) (armresources.ClientGetByIDResponse, error) {
		return armresources.ClientGetByIDResponse{}, nil
	}
	userAssignedRespFunc, systemAssignedRespFunc := getResourceByIDResponses(t, principalID)
	noIdentityUserAssignedRespFunc, noIdentitySystemAssignedRespFunc := getResourceByIDResponses(t, "")
	providersRespFunc := getProvidersResponse(t, resourceID)

	testCases := map[string]struct {
		claims      map[string]interface{}
		roleData    map[string]interface{}
		loginData   map[string]interface{}
		clientFunc  func(resourceID string) (armresources.ClientGetByIDResponse, error)
		expectError bool
	}{
		"login happy path user-assigned managed identity": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": resourceGroupName,
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  userAssignedRespFunc,
			expectError: false,
		},
		"login happy path system-assigned managed identity": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": resourceGroupName,
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: false,
		},
		"login fails when no identity data is returned": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  nilIdentityRespFunc,
			expectError: true,
		},
		"login fails when user-assigned identity data is not returned in response": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  noIdentityUserAssignedRespFunc,
			expectError: true,
		},
		"login fails when system-assigned identity data is not returned in response": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  noIdentitySystemAssignedRespFunc,
			expectError: true,
		},
		"login fails when bound_scale_sets is set": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
				"bound_scale_sets":            []string{"bad-vmss"},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"subscription_id":     subscriptionID,
				"resource_id":         resourceID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: true,
		},
		"login fails when missing resource_id": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"subscription_id":     subscriptionID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: true,
		},
		"login fails when missing resource_group_name and subscription_id": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":        roleName,
				"resource_id": resourceID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: true,
		},
		"login fails when missing resource_group_name": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":            roleName,
				"subscription_id": subscriptionID,
				"resource_id":     resourceID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: true,
		},
		"login fails when missing subscription_id": {
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			roleData: map[string]interface{}{
				"name":                        roleName,
				"policies":                    []string{"dev", "prod"},
				"bound_subscription_ids":      []string{subscriptionID},
				"bound_service_principal_ids": []string{principalID},
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"resource_group_name": "rg",
				"resource_id":         resourceID,
			},
			clientFunc:  systemAssignedRespFunc,
			expectError: true,
		},
	}

	for tt, tc := range testCases {
		t.Run(tt, func(t *testing.T) {
			b, s := getTestBackendWithResourceClient(t, tc.clientFunc, providersRespFunc)
			testRoleCreate(t, b, s, tc.roleData)
			if tc.expectError {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, tc.roleData)
			} else {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, tc.roleData)
			}
		})
	}
}

func TestLogin_BoundServicePrincipalID(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                        roleName,
		"policies":                    []string{"dev", "prod"},
		"bound_service_principal_ids": []string{"SpiD1", "sPid2"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": "spid2",
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	claims["oid"] = "bad id"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundGroupID(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":            roleName,
		"policies":        []string{"dev", "prod"},
		"bound_group_ids": []string{"grp1", "grp2"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp":    time.Now().Add(60 * time.Second).Unix(),
		"nbf":    time.Now().Add(-60 * time.Second).Unix(),
		"groups": []string{"grp1"},
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	claims["groups"] = []string{"GrP1"}
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	claims["groups"] = []string{"bad grp"}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	delete(claims, "groups")
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundSubscriptionID(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "1234abcd-1234-abcd-1234-abcd1234ef90"
	c, v, m := getTestBackendFunctions(false)

	g := getTestMSGraphClient()

	b, s := getTestBackendWithComputeClient(t, c, v, m, nil, g)

	vmName := "vm"
	vmssName := "vmss"
	rgName := "rg"
	roleName := "testrole"
	subID := "1234abcd-1234-abcd-1234-abcd1234ef90"
	roleData := map[string]interface{}{
		"name":                   roleName,
		"policies":               []string{"dev", "prod"},
		"bound_subscription_ids": []string{subID},
	}
	testRoleCreate(t, b, s, roleData)

	testCases := []struct {
		name            string
		claims          map[string]interface{}
		loginData       map[string]interface{}
		expectedSuccess bool
	}{
		{
			name: "error with only role provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role": roleName,
			},
		},
		{
			name: "error with only role and subscription_id provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":            roleName,
				"subscription_id": subscriptionID,
			},
		},
		{
			name: "error with only role resource_group_name subscription_id provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
			},
		},
		{
			name: "error with missing xms_az_rid and xms_mirid in token claims",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			name: "success with flexible vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "success with vm_name with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "error with vm_name with no user-assigned managed identities and bad subscription_id",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					"bad-subscription-id", "bad-rg-name", vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     "bad-subscription-id",
				"resource_group_name": "bad-rg-name",
				"vm_name":             vmName,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSuccess {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, roleData)
			} else {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, roleData)
			}
		})
	}
}

func TestLogin_BoundResourceGroup(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	badPrincipalID := "badID"
	subscriptionID := "1234abcd-1234-abcd-1234-abcd1234ef90"

	c, v, m := getTestBackendFunctions(false)

	g := getTestMSGraphClient()

	b, s := getTestBackendWithComputeClient(t, c, v, m, nil, g)

	vmssName := "vmss"
	vmName := "vm"
	roleName := "testrole"
	rgName := "rg"
	roleData := map[string]interface{}{
		"name":                  roleName,
		"policies":              []string{"dev", "prod"},
		"bound_resource_groups": []string{rgName},
	}
	testRoleCreate(t, b, s, roleData)

	testCases := []struct {
		name            string
		claims          map[string]interface{}
		loginData       map[string]interface{}
		expectedSuccess bool
	}{
		{
			name: "error with only role provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role": roleName,
			},
		},
		{
			name: "error with only role and subscription_id provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":            roleName,
				"subscription_id": subscriptionID,
			},
		},
		{
			name: "error with only role resource_group_name subscription_id provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
			},
		},
		{
			name: "error with missing xms_az_rid and xms_mirid in token claims",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with flexible vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with vm_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":        time.Now().Add(60 * time.Second).Unix(),
				"nbf":        time.Now().Add(-60 * time.Second).Unix(),
				"oid":        principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is no present
			name: "success with vm_name with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "error with vm_name with no user-assigned managed identities and bad resource group name",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, "bad-rg-name", vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": "bad-rg-name",
				"vm_name":             vmName,
			},
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "error with vm_name and xms_mirid and bad oid in token claims provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": badPrincipalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSuccess {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, roleData)
			} else {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, roleData)
			}
		})
	}
}

func TestLogin_BoundLocation(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "1234abcd-1234-abcd-1234-abcd1234abcd"
	location := "loc"

	c, v, m := getTestBackendFunctions(true)

	g := getTestMSGraphClient()

	b, s := getTestBackendWithComputeClient(t, c, v, m, nil, g)

	vmName := "good"
	vmssName := "good"
	rgName := "rg"
	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":            roleName,
		"policies":        []string{"dev", "prod"},
		"bound_locations": []string{location},
	}
	testRoleCreate(t, b, s, roleData)

	testCases := []struct {
		name            string
		claims          map[string]interface{}
		loginData       map[string]interface{}
		expectedSuccess bool
	}{
		{
			name: "error with only role provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role": roleName,
			},
		},
		{
			name: "error with missing xms_az_rid and xms_mirid in token claims",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with flexible vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with good vmss_name and token of a different VMSS with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", "anothervmss")),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with bad vmss_name",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", "bad")),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           "bad",
			},
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with vm_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":        time.Now().Add(60 * time.Second).Unix(),
				"nbf":        time.Now().Add(-60 * time.Second).Unix(),
				"oid":        principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with good vm_name and token of a different VM with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":        time.Now().Add(60 * time.Second).Unix(),
				"nbf":        time.Now().Add(-60 * time.Second).Unix(),
				"oid":        principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID, subscriptionID, rgName, "anotherVM"),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with bad vm_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":        time.Now().Add(60 * time.Second).Unix(),
				"nbf":        time.Now().Add(-60 * time.Second).Unix(),
				"oid":        principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID, subscriptionID, rgName, "bad"),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             "bad",
			},
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is no present
			name: "success with vm_name with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is no present
			name: "error with good vm_name and token of a different VM with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, "anotherVM"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             vmName,
			},
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is no present
			name: "error with good vm_name and token of a different VM with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, "bad"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vm_name":             "bad",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSuccess {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, roleData)
			} else {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, roleData)
			}
		})
	}
}

func TestLogin_BoundScaleSet(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "1234abcd-1234-abcd-1234-abcd1234ef90"

	c, v, m := getTestBackendFunctions(false)

	g := getTestMSGraphClient()

	b, s := getTestBackendWithComputeClient(t, c, v, m, nil, g)

	vmssName := "goodvmss"
	rgName := "rg"
	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":             roleName,
		"policies":         []string{"dev", "prod"},
		"bound_scale_sets": []string{vmssName},
	}
	testRoleCreate(t, b, s, roleData)

	testCases := []struct {
		name            string
		claims          map[string]interface{}
		loginData       map[string]interface{}
		expectedSuccess bool
	}{
		{
			name: "error with only role provided",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role": roleName,
			},
		},
		{
			name: "error with missing xms_az_rid and xms_mirid in token claims",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			// The Flexible VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with flexible vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The Uniform VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with uniform vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtVMSSRID,
					subscriptionID, rgName, vmssName),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with good vmss_name and token of a different VMSS with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", "anothervmss")),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with bad vmss_name",
			claims: map[string]interface{}{
				"exp": time.Now().Add(60 * time.Second).Unix(),
				"nbf": time.Now().Add(-60 * time.Second).Unix(),
				"oid": principalID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName, fmt.Sprintf("%s_randomInstanceID", "badvmss")),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName,
				"vmss_name":           "badvmss",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSuccess {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, roleData)
			} else {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, roleData)
			}
		})
	}
}

func TestLogin_AppID(t *testing.T) {
	subscriptionID := "1234abcd-1234-abcd-1234-abcd1234ef90"
	appID := "123e4567-e89b-12d3-a456-426655440000"
	badAppID := "aeoifkj"
	rgName1 := "rg-1"
	rgName2 := "rg-2"

	cl := func(rg string) armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse {
		if rg == rgName1 {
			return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{}
		} else if rg == rgName2 {
			return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{
				UserAssignedIdentitiesListResult: armmsi.UserAssignedIdentitiesListResult{
					Value: []*armmsi.Identity{
						{
							Properties: &armmsi.UserAssignedIdentityProperties{
								ClientID: &appID,
							},
						},
					},
				},
			}
		} else {
			return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{}
		}
	}

	c, v, m := getTestBackendFunctions(false)

	g := getTestMSGraphClient()

	b, s := getTestBackendWithComputeClient(t, c, v, m, cl, g)

	vmName := "vm"
	vmssName := "vmss"
	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                  roleName,
		"policies":              []string{"dev", "prod"},
		"bound_resource_groups": []string{rgName1, rgName2},
	}
	testRoleCreate(t, b, s, roleData)

	testCases := []struct {
		name            string
		claims          map[string]interface{}
		loginData       map[string]interface{}
		expectedSuccess bool
	}{
		{
			name: "error with only role provided",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
			},
			loginData: map[string]interface{}{
				"role": roleName,
			},
		},
		{
			name: "error with missing xms_az_rid and xms_mirid in token claims",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vmss_name":           vmssName,
			},
		},
		{
			// The Flexible VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with flexible vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName1, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName1, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The Uniform VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
				"xms_az_rid": fmt.Sprintf(fmtVMSSRID,
					subscriptionID, rgName1, vmssName),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName1, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vmss_name":           vmssName,
			},
			expectedSuccess: true,
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with token of a non-matching VMSS with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName1, fmt.Sprintf("%s_randomInstanceID", "anothervmss")),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName1, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VMSS in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "error with bad appid and vmss_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": badAppID,
				"xms_az_rid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName1, fmt.Sprintf("%s_randomInstanceID", vmssName)),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName1, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vmss_name":           vmssName,
			},
		},
		{
			// The VM in this case has user-assigned managed identities
			// so xms_az_rid is present
			name: "success with vm_name with user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":        time.Now().Add(60 * time.Second).Unix(),
				"nbf":        time.Now().Add(-60 * time.Second).Unix(),
				"appid":      appID,
				"xms_az_rid": fmt.Sprintf(fmtRID, subscriptionID, rgName1, vmName),
				"xms_mirid": fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName1, "userAssignedMI"),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "success with vm_name with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": appID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName1, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vm_name":             vmName,
			},
			expectedSuccess: true,
		},
		{
			// The VM in this case has no user-assigned managed identities
			// so xms_az_rid is not present
			name: "error with bad appid vm_name with no user-assigned managed identities",
			claims: map[string]interface{}{
				"exp":   time.Now().Add(60 * time.Second).Unix(),
				"nbf":   time.Now().Add(-60 * time.Second).Unix(),
				"appid": badAppID,
				"xms_mirid": fmt.Sprintf(fmtRID,
					subscriptionID, rgName1, vmName),
			},
			loginData: map[string]interface{}{
				"role":                roleName,
				"subscription_id":     subscriptionID,
				"resource_group_name": rgName1,
				"vm_name":             vmName,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSuccess {
				testLoginSuccess(t, b, s, tc.loginData, tc.claims, roleData)
			} else {
				testLoginFailure(t, b, s, tc.loginData, tc.claims, roleData)
			}
		})
	}
}

func TestLogin_InvalidCharacters(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                        roleName,
		"policies":                    []string{"dev", "prod"},
		"bound_service_principal_ids": []string{"*"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
	}

	loginData := map[string]interface{}{
		"role":            roleName,
		"subscription_id": "1234abcd-1234-1234-abcd-abcd1234abcd",
		"vmss_name":       "vmss",
		"vm_name":         "vm",
		"resource_group":  "rg",
	}
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = ".." // illegal
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "1234abcd-1234-1234-abcd-1234-abcd1234abcd"
	loginData["vmss_name"] = "a/../b"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "vmss"
	loginData["vm_name"] = "../a"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["vm_name"] = "vm"
	loginData["resource_group_name"] = "a/../b"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func testLoginSuccess(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) {
	t.Helper()
	if err := testLoginWithClaims(t, b, s, loginData, claims, roleData); err != nil {
		t.Fatal(err)
	}
}

func testLoginFailure(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) {
	t.Helper()
	if err := testLoginWithClaims(t, b, s, loginData, claims, roleData); err == nil {
		t.Fatal("no error thrown when expected")
	}
}

func testLoginWithClaims(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) error {
	t.Helper()
	return testLoginWithJWT(t, b, s, testJWT(t, claims), loginData, roleData)
}

func testLoginWithJWT(t *testing.T, b *azureAuthBackend, s logical.Storage, jwt string, loginData, roleData map[string]interface{}) error {
	t.Helper()

	loginData["jwt"] = jwt
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "login",
		Data:      loginData,
		Storage:   s,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	})
	if err != nil {
		return fmt.Errorf("err: %v", err)
	}
	if resp.IsError() {
		return resp.Error()
	}
	if resp.Auth == nil {
		return fmt.Errorf("received nil auth data")
	}

	if !policyutil.EquivalentPolicies(resp.Auth.Policies, roleData["policies"].([]string)) {
		return fmt.Errorf("policy mismatch, expected %v but got %v", roleData["policies"].([]string), resp.Auth.Policies)
	}
	return nil
}

func TestVerifyClaims(t *testing.T) {
	b, _ := getTestBackend(t)

	payload := map[string]interface{}{
		"nbf": time.Now().Add(-10 * time.Second).Unix(),
		"exp": time.Now().Add(10 * time.Second).Unix(),
	}
	idToken, err := b.provider.TokenVerifier().Verify(context.Background(), testJWT(t, payload))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	claims := new(additionalClaims)
	if err := idToken.Claims(claims); err != nil {
		t.Fatalf("err: %v", err)
	}
	role := new(azureRole)
	err = claims.verifyRole(role)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	claims.NotBefore = jsonTime(time.Now().Add(10 * time.Second))
	err = claims.verifyRole(role)
	if err == nil {
		t.Fatal("expected claim verification error")
	}

	claims = new(additionalClaims)
	if err = idToken.Claims(claims); err != nil {
		t.Fatalf("err: %v", err)
	}

	testCases := map[string]struct {
		bgIds  []string
		bspIds []string
		claims additionalClaims
		error  string
	}{
		"Should error since both fields can't be globbed together": {
			bgIds:  []string{"*"},
			bspIds: []string{"*"},
			claims: *claims,
			error:  "both cannot be '*'",
		},
		"Should error since claim GroupID not in role GroupIDs": {
			bgIds:  []string{"test-group-1"},
			bspIds: []string{"*"},
			claims: additionalClaims{
				NotBefore: claims.NotBefore,
				ObjectID:  claims.ObjectID,
				AppID:     claims.AppID,
				GroupIDs:  []string{"test-group-2"},
			},
			error: "groups not authorized",
		},
		"Should pass. Claim GroupID added": {
			bgIds:  []string{"test-group-1", "test-group2"},
			bspIds: []string{"*"},
			claims: additionalClaims{
				NotBefore: claims.NotBefore,
				ObjectID:  claims.ObjectID,
				AppID:     claims.AppID,
				GroupIDs:  []string{"test-group-2"},
			},
			error: "",
		},
		"Should error since claims OID not in role SPIDs": {
			bgIds:  []string{"*"},
			bspIds: []string{"spId1"},
			claims: additionalClaims{
				NotBefore: claims.NotBefore,
				ObjectID:  "test-oid",
				AppID:     claims.AppID,
				GroupIDs:  claims.GroupIDs,
			},
			error: "service principal not authorized",
		},
		"Should pass. Claim OID added": {
			bgIds:  []string{"*"},
			bspIds: []string{"spId1", "test-oid"},
			claims: additionalClaims{
				NotBefore: claims.NotBefore,
				ObjectID:  "test-oid",
				AppID:     claims.AppID,
				GroupIDs:  claims.GroupIDs,
			},
			error: "",
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			role.BoundGroupIDs = testCase.bgIds
			role.BoundServicePrincipalIDs = testCase.bspIds
			claims = &testCase.claims

			err = claims.verifyRole(role)

			if err != nil && testCase.error != "" && !strings.Contains(err.Error(), testCase.error) {
				t.Fatalf("expected an error %s, got %v", testCase.error, err)
			}
		})
	}
}

func TestGetAPIVersionForResource(t *testing.T) {
	subscriptionID := "eb936495-7356-4a35-af3e-ea68af201f0c"
	resourceID := "/subscriptions/eb936495-7356-4a35-af3e-ea68af201f0c/resourceGroups/azure-func-rg/providers/Microsoft.Web/sites/my-azure-func"

	providersRespFunc := getProvidersResponse(t, resourceID)
	b, _ := getTestBackendWithResourceClient(t, nil, providersRespFunc)
	apiVersion, err := b.getAPIVersionForResource(context.Background(), subscriptionID, resourceID)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expectedVer := "2022-08-01"
	if apiVersion != expectedVer {
		t.Fatalf("unexpected apiVersion returned, got %s, want %s", apiVersion, expectedVer)
	}

	// reset the provider and call getAPIVersionForResource again to ensure
	// we can get the API version from the cache
	b.provider = &mockProvider{}
	apiVersion, err = b.getAPIVersionForResource(context.Background(), subscriptionID, resourceID)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if apiVersion != expectedVer {
		t.Fatalf("unexpected apiVersion returned, got %s, want %s", apiVersion, expectedVer)
	}
}

// getResourceByIDResponses is a test helper to get the functions that return
// the azure arm resource client responses. If principalID is an empty string
// then no identity data will be set in the response.
func getResourceByIDResponses(t *testing.T, principalID string) (
	func(_ string) (armresources.ClientGetByIDResponse, error),
	func(_ string) (armresources.ClientGetByIDResponse, error),
) {
	t.Helper()
	u := armresources.ClientGetByIDResponse{
		armresources.GenericResource{
			Identity: &armresources.Identity{
				UserAssignedIdentities: map[string]*armresources.IdentityUserAssignedIdentitiesValue{},
			},
		},
	}
	s := armresources.ClientGetByIDResponse{
		armresources.GenericResource{
			Identity: &armresources.Identity{},
		},
	}
	if principalID != "" {
		identityValue := map[string]*armresources.IdentityUserAssignedIdentitiesValue{
			"mockuserassignedmsi": {
				PrincipalID: &principalID,
			},
		}

		u.GenericResource.Identity.UserAssignedIdentities = identityValue
		s.GenericResource.Identity.PrincipalID = &principalID
	}

	userAssignedRespFunc := func(_ string) (armresources.ClientGetByIDResponse, error) {
		return u, nil
	}
	systemAssignedRespFunc := func(_ string) (armresources.ClientGetByIDResponse, error) {
		return s, nil
	}

	return userAssignedRespFunc, systemAssignedRespFunc
}

// getProvidersResponse is a test helper to get the function that returns
// the azure arm resource providers client response.
func getProvidersResponse(t *testing.T, resourceID string) func(_ string) (armresources.ProvidersClientGetResponse, error) {
	t.Helper()

	resourceType, err := arm.ParseResourceType(resourceID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	ver0 := "2022-11-01-preview"
	ver1 := "2022-08-01"
	ver2 := "2022-03-01"
	u := armresources.ProvidersClientGetResponse{
		armresources.Provider{
			ResourceTypes: []*armresources.ProviderResourceType{
				{
					APIVersions: []*string{
						&ver0,
						&ver1,
						&ver2,
					},
					ResourceType: &resourceType.Type,
				},
			},
		},
	}
	providersRespFunc := func(_ string) (armresources.ProvidersClientGetResponse, error) {
		return u, nil
	}
	return providersRespFunc
}

func testJWT(t *testing.T, payload map[string]interface{}) string {
	headers := map[string]interface{}{
		"alg": oidc.RS256,
	}
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	fixedHeader := base64.RawURLEncoding.EncodeToString(headersJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	fixedSignature := base64.RawURLEncoding.EncodeToString([]byte("signature"))

	return fmt.Sprintf("%s.%s.%s", fixedHeader, encodedPayload, fixedSignature)
}

func getTestBackendFunctions(withLocation bool) (
	func(_ string) (armcompute.VirtualMachinesClientGetResponse, error),
	func(_ string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error),
	func(_ string) (armmsi.UserAssignedIdentitiesClientGetResponse, error),
) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"

	if !withLocation {
		c := func(_ string) (armcompute.VirtualMachinesClientGetResponse, error) {
			id := armcompute.VirtualMachineIdentity{
				PrincipalID: &principalID,
			}
			return armcompute.VirtualMachinesClientGetResponse{
				armcompute.VirtualMachine{
					Identity: &id,
				},
			}, nil
		}
		v := func(_ string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error) {
			id := armcompute.VirtualMachineScaleSetIdentity{
				PrincipalID: &principalID,
			}
			return armcompute.VirtualMachineScaleSetsClientGetResponse{armcompute.VirtualMachineScaleSet{
				Identity: &id,
			}}, nil
		}

		m := func(_ string) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
			userAssignedIdentityProperties := armmsi.UserAssignedIdentityProperties{
				PrincipalID: &principalID,
			}
			return armmsi.UserAssignedIdentitiesClientGetResponse{armmsi.Identity{
				Properties: &userAssignedIdentityProperties,
			}}, nil
		}

		return c, v, m
	} else {
		location := "loc"

		c := func(vmName string) (armcompute.VirtualMachinesClientGetResponse, error) {
			id := armcompute.VirtualMachineIdentity{
				PrincipalID: &principalID,
			}
			switch vmName {
			case "good":
				return armcompute.VirtualMachinesClientGetResponse{armcompute.VirtualMachine{
					Identity: &id,
					Location: &location,
				}}, nil
			case "bad":
				badLoc := "bad"
				return armcompute.VirtualMachinesClientGetResponse{armcompute.VirtualMachine{
					Identity: &id,
					Location: &badLoc,
				}}, nil
			}
			return armcompute.VirtualMachinesClientGetResponse{}, nil
		}
		v := func(vmssName string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error) {
			id := armcompute.VirtualMachineScaleSetIdentity{
				PrincipalID: &principalID,
			}
			switch vmssName {
			case "good":
				return armcompute.VirtualMachineScaleSetsClientGetResponse{armcompute.VirtualMachineScaleSet{
					Identity: &id,
					Location: &location,
				}}, nil
			case "bad":
				badLoc := "bad"
				return armcompute.VirtualMachineScaleSetsClientGetResponse{armcompute.VirtualMachineScaleSet{
					Identity: &id,
					Location: &badLoc,
				}}, nil
			}
			return armcompute.VirtualMachineScaleSetsClientGetResponse{}, nil
		}

		m := func(_ string) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
			userAssignedIdentityProperties := armmsi.UserAssignedIdentityProperties{
				PrincipalID: &principalID,
			}
			return armmsi.UserAssignedIdentitiesClientGetResponse{armmsi.Identity{
				Properties: &userAssignedIdentityProperties,
			}}, nil
		}

		return c, v, m
	}
}

func getTestMSGraphClient() func() (client.MSGraphClient, error) {
	return func() (client.MSGraphClient, error) {
		return nil, nil
	}
}

func Test_additionalClaims_verifyVM(t *testing.T) {
	type fields struct {
		NotBefore                    jsonTime
		ObjectID                     string
		AppID                        string
		GroupIDs                     []string
		XMSAzureResourceID           string
		XMSManagedIdentityResourceID string
	}
	type args struct {
		vmName string
	}

	appID := "123e4567-e89b-12d3-a456-426655440000"
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "eb936495-7356-4a35-af3e-ea68af201f0c"
	rgName := "rg"
	vmName := "vm"
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "error if xms_mirid and xms_az_rid are empty",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
			},
			args: args{
				vmName: vmName,
			},
			wantErr: assert.Error,
		},
		{
			name: "error if vm_name does not match when only xms_mirid exists",
			fields: fields{
				NotBefore:                    jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:                     principalID,
				AppID:                        appID,
				GroupIDs:                     []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
			},
			args: args{
				vmName: "wrong-vm",
			},
			wantErr: assert.Error,
		},
		{
			name: "error if vm_name does not match when xms_az_rid and xms_mirid exist",
			fields: fields{
				NotBefore:          jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:           principalID,
				AppID:              appID,
				GroupIDs:           []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				vmName: "wrong-vm",
			},
			wantErr: assert.Error,
		},
		{
			name: "happy if vm_name matches xms_mirid",
			fields: fields{
				NotBefore:                    jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:                     principalID,
				AppID:                        appID,
				GroupIDs:                     []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
			},
			args: args{
				vmName: vmName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy if vm_name matches xms_az_rid",
			fields: fields{
				NotBefore:          jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:           principalID,
				AppID:              appID,
				GroupIDs:           []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				vmName: vmName,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &additionalClaims{
				NotBefore:                    tt.fields.NotBefore,
				ObjectID:                     tt.fields.ObjectID,
				AppID:                        tt.fields.AppID,
				GroupIDs:                     tt.fields.GroupIDs,
				XMSAzureResourceID:           tt.fields.XMSAzureResourceID,
				XMSManagedIdentityResourceID: tt.fields.XMSManagedIdentityResourceID,
			}
			tt.wantErr(t, c.verifyVM(tt.args.vmName), fmt.Sprintf("verifyVM(%v)", tt.args.vmName))
		})
	}
}

func Test_additionalClaims_verifyVMSS(t *testing.T) {
	type fields struct {
		NotBefore                    jsonTime
		ObjectID                     string
		AppID                        string
		GroupIDs                     []string
		XMSAzureResourceID           string
		XMSManagedIdentityResourceID string
	}
	type args struct {
		vmssName string
	}

	appID := "123e4567-e89b-12d3-a456-426655440000"
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "eb936495-7356-4a35-af3e-ea68af201f0c"
	rgName := "rg"
	vmssName := "vmss"
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "error if xms_mirid and xms_az_rid are empty",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
			},
			args: args{
				vmssName: vmssName,
			},
			wantErr: assert.Error,
		},
		{
			name: "error if flexible vmss_name does not match when only xms_mirid exists",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
			},
			args: args{
				vmssName: "wrong-vmss",
			},
			wantErr: assert.Error,
		},
		{
			name: "error if flexible vmss_name does not match when xms_az_rid and xms_mirid exist",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				vmssName: "wrong-vm",
			},
			wantErr: assert.Error,
		},
		{
			name: "happy if flexible vmss_name matches xms_mirid",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
			},
			args: args{
				vmssName: vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy if uniform vmss_name matches xms_mirid",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtVMSSRID, subscriptionID, rgName,
					vmssName),
			},
			args: args{
				vmssName: vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy if flexible vmss_name matches xms_az_rid",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				vmssName: vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy if uniform vmss_name matches xms_az_rid",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtVMSSRID, subscriptionID, rgName,
					vmssName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				vmssName: vmssName,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &additionalClaims{
				NotBefore:                    tt.fields.NotBefore,
				ObjectID:                     tt.fields.ObjectID,
				AppID:                        tt.fields.AppID,
				GroupIDs:                     tt.fields.GroupIDs,
				XMSAzureResourceID:           tt.fields.XMSAzureResourceID,
				XMSManagedIdentityResourceID: tt.fields.XMSManagedIdentityResourceID,
			}
			tt.wantErr(t, c.verifyVMSS(tt.args.vmssName), fmt.Sprintf("verifyVMSS(%v)", tt.args.vmssName))
		})
	}
}

func Test_additionalClaims_verifyResourceGroup(t *testing.T) {
	type fields struct {
		NotBefore                    jsonTime
		ObjectID                     string
		AppID                        string
		GroupIDs                     []string
		XMSAzureResourceID           string
		XMSManagedIdentityResourceID string
	}
	type args struct {
		resourceGroupName string
		vmName            string
		vmssName          string
		resourceID        string
	}
	appID := "123e4567-e89b-12d3-a456-426655440000"
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	subscriptionID := "eb936495-7356-4a35-af3e-ea68af201f0c"
	rgName := "rg"
	vmssName := "vmss"
	vmName := "vm"
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "error if vm vmss_name and resource_id are empty",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
			},
			args: args{
				resourceGroupName: rgName,
			},
			wantErr: assert.Error,
		},
		{
			name: "happy with matching resource_id",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
			},
			args: args{
				resourceGroupName: rgName,
				resourceID: fmt.Sprintf(fmtResourceGroupID, subscriptionID, rgName,
					"providers/Microsoft.Web",
					"sites",
					"my-azure-func"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "error with missing xms_az_rid xms_mirid when vmss is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.Error,
		},
		{
			name: "happy with matching xms_mirid when flexible vmss is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy with matching xms_mirid when uniform vmss is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtVMSSRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy with matching xms_az_rid when flexible vmss is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName,
					fmt.Sprintf("%s_instanceID", vmssName)),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy with matching xms_az_rid when uniform vmss is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtVMSSRID, subscriptionID, rgName,
					vmssName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "error with non-matching xms_az_rid xms_mirid when vmss_name is provided",
			fields: fields{
				NotBefore: jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:  principalID,
				AppID:     appID,
				GroupIDs:  []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, "wrong-rg-name",
					fmt.Sprintf("%s_instanceID", vmssName)),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, "wrong-rg-name", "userAssignedIdentity"),
			},
			args: args{
				resourceGroupName: rgName,
				vmssName:          vmssName,
			},
			wantErr: assert.Error,
		},
		{
			name: "happy with matching xms_mirid when vm_name is provided",
			fields: fields{
				NotBefore:                    jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:                     principalID,
				AppID:                        appID,
				GroupIDs:                     []string{"test-group-1"},
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
			},
			args: args{
				resourceGroupName: rgName,
				vmName:            vmName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "happy with matching xms_az_rid when vm_name is provided",
			fields: fields{
				NotBefore:          jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:           principalID,
				AppID:              appID,
				GroupIDs:           []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, rgName, vmName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, rgName, "userAssignedIdentity"),
			},
			args: args{
				resourceGroupName: rgName,
				vmName:            vmName,
			},
			wantErr: assert.NoError,
		},
		{
			name: "error with non-matching xms_az_rid xms_mirid when vm_name is provided",
			fields: fields{
				NotBefore:          jsonTime(time.Now().Add(60 * time.Second)),
				ObjectID:           principalID,
				AppID:              appID,
				GroupIDs:           []string{"test-group-1"},
				XMSAzureResourceID: fmt.Sprintf(fmtRID, subscriptionID, "wrong-rg-name", vmName),
				XMSManagedIdentityResourceID: fmt.Sprintf(fmtRIDWithUserAssignedIdentities,
					subscriptionID, "wrong-rg-name", "userAssignedIdentity"),
			},
			args: args{
				resourceGroupName: rgName,
				vmName:            vmName,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &additionalClaims{
				NotBefore:                    tt.fields.NotBefore,
				ObjectID:                     tt.fields.ObjectID,
				AppID:                        tt.fields.AppID,
				GroupIDs:                     tt.fields.GroupIDs,
				XMSAzureResourceID:           tt.fields.XMSAzureResourceID,
				XMSManagedIdentityResourceID: tt.fields.XMSManagedIdentityResourceID,
			}
			tt.wantErr(t, c.verifyResourceGroup(tt.args.resourceGroupName,
				tt.args.vmName,
				tt.args.vmssName,
				tt.args.resourceID),
				fmt.Sprintf("verifyResourceGroup(%v, %v, %v, %v)",
					tt.args.resourceGroupName,
					tt.args.vmName,
					tt.args.vmssName,
					tt.args.resourceID))
		})
	}
}
