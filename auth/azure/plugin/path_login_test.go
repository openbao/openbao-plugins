package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/hashicorp/vault/helper/policyutil"
	"github.com/hashicorp/vault/logical"
)

func TestLogin(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":     roleName,
		"policies": []string{"dev", "prod"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
	}

	loginData := map[string]interface{}{
		"role": roleName,
		"jwt":  testJWT(t, claims),
	}
	testLoginSuccess(t, b, s, loginData, roleData)

	claims["nbf"] = time.Now().Add(60 * time.Second).Unix()
	loginData["jwt"] = testJWT(t, claims)
	testLoginFailure(t, b, s, loginData, roleData)

	claims["nbf"] = time.Now().Add(-60 * time.Second).Unix()
	claims["exp"] = time.Now().Add(-60 * time.Second).Unix()
	loginData["jwt"] = testJWT(t, claims)
	testLoginFailure(t, b, s, loginData, roleData)
}

func TestLogin_BoundServicePrincipalID(t *testing.T) {
	b, s := getTestBackend(t)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":                        roleName,
		"policies":                    []string{"dev", "prod"},
		"bound_service_principal_ids": []string{"spid1", "spid2"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(60 * time.Second).Unix(),
		"nbf": time.Now().Add(-60 * time.Second).Unix(),
		"oid": "spid2",
	}

	loginData := map[string]interface{}{
		"role": roleName,
		"jwt":  testJWT(t, claims),
	}
	testLoginSuccess(t, b, s, loginData, roleData)

	claims["oid"] = "bad id"
	loginData["jwt"] = testJWT(t, claims)
	testLoginFailure(t, b, s, loginData, roleData)
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
		"jwt":  testJWT(t, claims),
	}
	testLoginSuccess(t, b, s, loginData, roleData)

	claims["groups"] = []string{"bad grp"}
	loginData["jwt"] = testJWT(t, claims)
	testLoginFailure(t, b, s, loginData, roleData)

	delete(claims, "groups")
	loginData["jwt"] = testJWT(t, claims)
	testLoginFailure(t, b, s, loginData, roleData)
}

func testLoginSuccess(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, roleData map[string]interface{}) {
	t.Helper()
	if err := testLogin(t, b, s, loginData, roleData); err != nil {
		t.Fatal(err)
	}
}

func testLoginFailure(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, roleData map[string]interface{}) {
	t.Helper()
	if err := testLogin(t, b, s, loginData, roleData); err == nil {
		t.Fatal("no error thown when expected")
	}
}

func testLogin(t *testing.T, b *azureAuthBackend, s logical.Storage, loginData, roleData map[string]interface{}) error {
	t.Helper()
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
