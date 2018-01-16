package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/helper/policyutil"
	"github.com/hashicorp/vault/logical"
)

func TestLogin(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "test-tenant-id",
		"resource":  "https://vault.hashicorp.com",
	}
	testConfigCreate(t, b, s, configData)

	roleName := "testrole"
	roleData := map[string]interface{}{
		"name":     roleName,
		"policies": []string{"dev", "prod"},
	}
	testRoleCreate(t, b, s, roleData)

	claims := map[string]interface{}{
		"exp": time.Now().Add(30 * time.Second).Unix(),
		"nbf": time.Now().Add(-30 * time.Second).Unix(),
	}

	loginData := map[string]interface{}{
		"role": roleName,
		"jwt":  testJWT(t, claims),
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "login",
		Data:      loginData,
		Storage:   s,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	t.Logf("%#v\n", resp)
	if resp.Auth == nil {
		t.Fatal("received nil auth data")
	}

	if !policyutil.EquivalentPolicies(resp.Auth.Policies, roleData["policies"].([]string)) {
		t.Fatalf("policy mismatch, expected %v but got %v", roleData["policies"].([]string), resp.Auth.Policies)
	}
}

func TestVerifyClaims(t *testing.T) {
	v := newMockVerifier()

	payload := map[string]interface{}{
		"nbf": time.Now().Add(-10 * time.Second).Unix(),
	}
	idToken, err := v.Verify(context.Background(), testJWT(t, payload))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = verifyClaims(idToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	payload["nbf"] = time.Now().Add(10 * time.Second).Unix()
	idToken, err = v.Verify(context.Background(), testJWT(t, payload))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = verifyClaims(idToken)
	if err == nil {
		t.Fatal("expected claim verification error")
	}
}

func testJWT(t *testing.T, payload map[string]interface{}) string {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	fixedHeader := base64.RawURLEncoding.EncodeToString([]byte("{}"))
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	fixedSignature := base64.RawURLEncoding.EncodeToString([]byte("signature"))

	return fmt.Sprintf("%s.%s.%s", fixedHeader, encodedPayload, fixedSignature)
}
