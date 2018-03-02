package plugin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/hashicorp/vault/helper/logformat"
	"github.com/hashicorp/vault/logical"
	log "github.com/mgutz/logxi/v1"
)

func TestDeriveTenantId(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header, 0),
	}

	expectedTenantID := "55dd1a28-8db3-44f0-86bd-a8e4d7d51771"
	authorizationURI := fmt.Sprintf("https://login.windows.net/%s", expectedTenantID)
	headerValue := fmt.Sprintf(`Bearer authorization_uri="%s", error="invalid_token", error_description="The authentication failed because of missing 'Authorization' header.`, authorizationURI)

	resp.Header.Add("WWW-Authenticate", headerValue)

	actualTenantID, err := deriveTenantID(resp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if expectedTenantID != actualTenantID {
		t.Fatalf("expected: %s, actual: %s", expectedTenantID, actualTenantID)
	}
}

func getTestBackend(t *testing.T) (*azureAuthBackend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24
	config := &logical.BackendConfig{
		Logger: logformat.NewVaultLogger(log.LevelTrace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b := Backend(config)
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}
	return b, config.StorageView
}

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
