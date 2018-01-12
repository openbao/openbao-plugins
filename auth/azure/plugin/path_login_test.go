package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	oidc "github.com/coreos/go-oidc"
)

// mockKeySet is used in tests to bypass signature validation and return only
// the jwt payload
type mockKeySet struct{}

func (s *mockKeySet) VerifySignature(ctx context.Context, idToken string) ([]byte, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding payload: %v", err)
	}
	return payload, nil
}

func newMockVerifier() *oidc.IDTokenVerifier {
	config := &oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   true,
	}
	ks := new(mockKeySet)
	return oidc.NewVerifier("", ks, config)
}

func TestVerifyLogin(t *testing.T) {

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
