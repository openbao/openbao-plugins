package azureauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	oidc "github.com/coreos/go-oidc"
	"github.com/hashicorp/vault/helper/policyutil"
	"github.com/hashicorp/vault/logical"
)

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
	principalID := "prinID"
	c := func(vmName string) (compute.VirtualMachine, error) {
		id := compute.VirtualMachineIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachine{
			Identity: &id,
		}, nil
	}
	v := func(vmssName string) (compute.VirtualMachineScaleSet, error) {
		id := compute.VirtualMachineScaleSetIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachineScaleSet{
			Identity: &id,
		}, nil
	}

	b, s := getTestBackendWithComputeClient(t, c, v)

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
	testLoginFailure(t, b, s, loginData, claims, roleData)
	delete(loginData, "vmss_name")
	testLoginSuccess(t, b, s, loginData, claims, roleData)

	loginData["subscription_id"] = "bad sub"
	testLoginFailure(t, b, s, loginData, claims, roleData)
}

func TestLogin_BoundResourceGroup(t *testing.T) {
	principalID := "prinID"
	c := func(vmName string) (compute.VirtualMachine, error) {
		id := compute.VirtualMachineIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachine{
			Identity: &id,
		}, nil
	}
	v := func(vmName string) (compute.VirtualMachineScaleSet, error) {
		id := compute.VirtualMachineScaleSetIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachineScaleSet{
			Identity: &id,
		}, nil
	}
	b, s := getTestBackendWithComputeClient(t, c, v)

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

func TestLogin_BoundLocation(t *testing.T) {
	principalID := "prinID"
	location := "loc"
	c := func(vmName string) (compute.VirtualMachine, error) {
		id := compute.VirtualMachineIdentity{
			PrincipalID: &principalID,
		}
		switch vmName {
		case "good":
			return compute.VirtualMachine{
				Identity: &id,
				Location: &location,
			}, nil
		case "bad":
			badLoc := "bad"
			return compute.VirtualMachine{
				Identity: &id,
				Location: &badLoc,
			}, nil
		}
		return compute.VirtualMachine{}, nil
	}
	v := func(vmssName string) (compute.VirtualMachineScaleSet, error) {
		id := compute.VirtualMachineScaleSetIdentity{
			PrincipalID: &principalID,
		}
		switch vmssName {
		case "good":
			return compute.VirtualMachineScaleSet{
				Identity: &id,
				Location: &location,
			}, nil
		case "bad":
			badLoc := "bad"
			return compute.VirtualMachineScaleSet{
				Identity: &id,
				Location: &badLoc,
			}, nil
		}
		return compute.VirtualMachineScaleSet{}, nil
	}

	b, s := getTestBackendWithComputeClient(t, c, v)

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
	principalID := "prinID"
	c := func(vmName string) (compute.VirtualMachine, error) {
		id := compute.VirtualMachineIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachine{
			Identity: &id,
		}, nil
	}
	v := func(vmssName string) (compute.VirtualMachineScaleSet, error) {
		id := compute.VirtualMachineScaleSetIdentity{
			PrincipalID: &principalID,
		}
		return compute.VirtualMachineScaleSet{
			Identity: &id,
		}, nil
	}

	b, s := getTestBackendWithComputeClient(t, c, v)

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

	err = b.verifyClaims(claims, new(azureRole))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	claims.NotBefore = jsonTime(time.Now().Add(10 * time.Second))
	err = b.verifyClaims(claims, new(azureRole))
	if err == nil {
		t.Fatal("expected claim verification error")
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

	fixedHeader := base64.RawURLEncoding.EncodeToString([]byte(headersJSON))
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	fixedSignature := base64.RawURLEncoding.EncodeToString([]byte("signature"))

	return fmt.Sprintf("%s.%s.%s", fixedHeader, encodedPayload, fixedSignature)
}
