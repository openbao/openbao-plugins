package plugin

import (
	"fmt"
	"net/http"
	"testing"
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
