package azureauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/coreos/go-oidc"
	"github.com/hashicorp/vault/sdk/helper/policyutil"
	"github.com/hashicorp/vault/sdk/logical"
)

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
	c, v, m := getTestBackendFunctions(false)

	b, s := getTestBackendWithComputeClient(t, c, v, m)

	roleName := "testrole"
	subID := "subID"
	roleData := map[string]interface{}{
		"name":                   roleName,
		"policies":               []string{"dev", "prod"},
		"bound_subscription_ids": []string{subID},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": principalID,
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = subID
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["resource_group_name"] = "rg"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "vmss"
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["vm_name"] = "vm"
	delete(loginData, "vmss_name")
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "bad sub"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundResourceGroup(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	c, v, m := getTestBackendFunctions(false)

	b, s := getTestBackendWithComputeClient(t, c, v, m)

	roleName := "testrole"
	rg := "rg"
	roleData := map[string]interface{}{
		"name":                  roleName,
		"policies":              []string{"dev", "prod"},
		"bound_resource_groups": []string{rg},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": principalID,
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "sub"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["resource_group_name"] = rg
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "vmss"
	testLoginSuccess(t, b, s, loginData, claims, roleData)
	delete(loginData, "vmss_name")

	loginData["vm_name"] = "vm"
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["resource_group_name"] = "bad rg"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundResourceGroupWithUserAssignedID(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	badPrincipalID := "badID"
	c, v, m := getTestBackendFunctions(false)

	b, s := getTestBackendWithComputeClient(t, c, v, m)

	roleName := "testrole"
	rg := "rg"
	roleData := map[string]interface{}{
		"name":                  roleName,
		"policies":              []string{"dev", "prod"},
		"bound_resource_groups": []string{rg},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": principalID,
	}
	badClaims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": badPrincipalID,
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "sub"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["resource_group_name"] = rg
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "vmss"
	testLoginSuccess(t, b, s, loginData, claims, roleData)
	delete(loginData, "vmss_name")

	loginData["vm_name"] = "vm"
	testLoginSuccess(t, b, s, loginData, claims, roleData)
	testLoginFailure(t, b, s, loginData, badClaims, roleData)

	loginData["resource_group_name"] = "bad rg"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundLocation(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	location := "loc"
	c, v, m := getTestBackendFunctions(true)

	b, s := getTestBackendWithComputeClient(t, c, v, m)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":            roleName,
		"policies":        []string{"dev", "prod"},
		"bound_locations": []string{location},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": principalID,
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "sub"
	loginData["resource_group_name"] = "rg"

	loginData["vmss_name"] = "good"
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "bad"
	testLoginFailure(t, b, s, loginData, claims, roleData)

	delete(loginData, "vmss_name")

	loginData["vm_name"] = "good"
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["vm_name"] = "bad"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundScaleSet(t *testing.T) {
	principalID := "123e4567-e89b-12d3-a456-426655440000"
	c, v, m := getTestBackendFunctions(false)

	b, s := getTestBackendWithComputeClient(t, c, v, m)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":             roleName,
		"policies":         []string{"dev", "prod"},
		"bound_scale_sets": []string{"goodvmss"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": principalID,
	}

	loginData := map[string]interface{}{
		"role": roleName,
	}
	testLoginFailure(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "sub"
	loginData["resource_group_name"] = "rg"

	loginData["vmss_name"] = "goodvmss"
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["vmss_name"] = "badvmss"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func testLoginSuccess(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) {
	t.Helper()
	if err := testLogin(t, b, s, loginData, claims, roleData); err != nil {
		t.Fatal(err)
	}
}

func testLoginFailure(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) {
	t.Helper()
	if err := testLogin(t, b, s, loginData, claims, roleData); err == nil {
		t.Fatal("no error thown when expected")
	}
}

func testLogin(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, claims, roleData map[string]interface{}) error {
	t.Helper()
	loginData["jwt"] = testJWT(t, claims)
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
		return fmt.Errorf(resp.Error().Error())
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
	idToken, err := b.provider.Verifier().Verify(context.Background(), testJWT(t, payload))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	claims := new(additionalClaims)
	if err := idToken.Claims(claims); err != nil {
		t.Fatalf("err: %v", err)
	}
	role := new(azureRole)
	err = b.verifyClaims(claims, role)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	claims.NotBefore = jsonTime(time.Now().Add(10 * time.Second))
	err = b.verifyClaims(claims, role)
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
				claims.NotBefore,
				claims.ObjectID,
				[]string{"test-group-2"},
			},
			error: "groups not authorized",
		},
		"Should pass. Claim GroupID added": {
			bgIds:  []string{"test-group-1", "test-group2"},
			bspIds: []string{"*"},
			claims: additionalClaims{
				claims.NotBefore,
				claims.ObjectID,
				[]string{"test-group-2"},
			},
			error: "",
		},
		"Should error since claims OID not in role SPIDs": {
			bgIds:  []string{"*"},
			bspIds: []string{"spId1"},
			claims: additionalClaims{
				claims.NotBefore,
				"test-oid",
				claims.GroupIDs,
			},
			error: "service principal not authorized",
		},
		"Should pass. Claim OID added": {
			bgIds:  []string{"*"},
			bspIds: []string{"spId1", "test-oid"},
			claims: additionalClaims{
				claims.NotBefore,
				"test-oid",
				claims.GroupIDs,
			},
			error: "",
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			role.BoundGroupIDs = testCase.bgIds
			role.BoundServicePrincipalIDs = testCase.bspIds
			claims = &testCase.claims

			err = b.verifyClaims(claims, role)

			if err != nil && testCase.error != "" && !strings.Contains(err.Error(), testCase.error) {
				t.Fatalf("expected an error %s, got %v", testCase.error, err)
			}
		})
	}
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
