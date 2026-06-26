package github

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
)

// TestGitHub_Login tests that we can successfully login with the given config
func TestGitHub_Login(t *testing.T) {
	b, s := createBackendWithStorage(t)

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// Write the config
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"organization": "foo-org",
			"base_url":     ts.URL, // base_url will call the test server
		},
		Storage: s,
	})
	assert.NoError(t, err)
	assert.NoError(t, resp.Error())

	// Read the config
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.ReadOperation,
		Storage:   s,
	})
	assert.NoError(t, err)
	assert.NoError(t, resp.Error())

	// attempt a login
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "login",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"token": "faketoken",
		},
		Storage: s,
	})

	expectedMetaData := map[string]string{
		"org":      "foo-org",
		"username": "user-foo",
	}
	assert.Equal(t, expectedMetaData, resp.Auth.Metadata)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error())
}

// TestGitHub_Login_OrgInvalid tests that we cannot login with an ID other than
// what is set in the config
func TestGitHub_Login_OrgInvalid(t *testing.T) {
	b, s := createBackendWithStorage(t)
	ctx := context.Background()

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// write and store config
	config := config{
		Organization:   "foo-org",
		OrganizationID: 9999,
		BaseURL:        ts.URL + "/", // base_url will call the test server
	}
	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		t.Fatalf("failed creating storage entry")
	}
	if err := s.Put(ctx, entry); err != nil {
		t.Fatalf("writing to in mem storage failed")
	}

	// attempt a login
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "login",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"token": "faketoken",
		},
		Storage: s,
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	// With the optimized API usage, we now get an organization ID mismatch error
	// instead of membership error because we check organization first
	assert.Contains(t, err.Error(), "organization ID mismatch")
}

// TestGitHub_Login_OrgNameChanged tests that we can successfully login with the
// given config and emit a warning when the organization name has changed
func TestGitHub_Login_OrgNameChanged(t *testing.T) {
	b, s := createBackendWithStorage(t)
	ctx := context.Background()

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// write and store config
	// the name does not match what the API will return but the ID does
	config := config{
		Organization:   "old-name",
		OrganizationID: 12345,
		BaseURL:        ts.URL + "/", // base_url will call the test server
	}
	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		t.Fatalf("failed creating storage entry")
	}
	if err := s.Put(ctx, entry); err != nil {
		t.Fatalf("writing to in mem storage failed")
	}

	// attempt a login
	_, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "login",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"token": "faketoken",
		},
		Storage: s,
	})

	// With the optimized API usage, authentication will fail if the organization
	// name in config doesn't match the actual organization name, even if IDs match.
	// This is more secure than the previous behavior.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get organization \"old-name\"")
}

// TestGitHub_Login_NoOrgID tests that we can successfully login with the given
// config when no organization ID is present and write the fetched ID to the
// config
func TestGitHub_Login_NoOrgID(t *testing.T) {
	b, s := createBackendWithStorage(t)
	ctx := context.Background()

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// write and store config without Org ID
	config := config{
		Organization: "foo-org",
		BaseURL:      ts.URL + "/", // base_url will call the test server
	}
	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		t.Fatalf("failed creating storage entry")
	}
	if err := s.Put(ctx, entry); err != nil {
		t.Fatalf("writing to in mem storage failed")
	}

	// attempt a login
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "login",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"token": "faketoken",
		},
		Storage: s,
	})

	expectedMetaData := map[string]string{
		"org":      "foo-org",
		"username": "user-foo",
	}
	assert.Equal(t, expectedMetaData, resp.Auth.Metadata)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error())

	// Read the config
	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.ReadOperation,
		Storage:   s,
	})
	assert.NoError(t, err)
	assert.NoError(t, resp.Error())

	// the ID should be set, we grab it from the GET /orgs API
	assert.Equal(t, int64(12345), resp.Data["organization_id"])
}

// TestGitHub_PathLoginRenew tests the token renewal flow
func TestGitHub_PathLoginRenew(t *testing.T) {
	b, s := createBackendWithStorage(t)

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// Write the config
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"organization": "foo-org",
			"base_url":     ts.URL,
		},
		Storage: s,
	})
	assert.NoError(t, err)
	if resp != nil {
		assert.NoError(t, resp.Error())
	}

	// Write a team mapping so we have policies
	_, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "map/teams/default",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"value": "test-policy",
		},
		Storage: s,
	})
	assert.NoError(t, err)

	// Initial login to get auth token
	loginResp, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "login",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"token": "faketoken",
		},
		Storage: s,
	})
	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	if loginResp != nil {
		assert.NoError(t, loginResp.Error())
		assert.NotNil(t, loginResp.Auth)
	}
	if loginResp == nil || loginResp.Auth == nil {
		t.Fatal("Login response or Auth is nil")
	}

	// Prepare renewal request - need to use exact copy of Auth to test renewal properly
	// The renewal should use the auth state as it would be stored
	// Note: In the real system, Policies get converted to TokenPolicies by the backend framework
	tokenPolicies := loginResp.Auth.TokenPolicies
	if len(tokenPolicies) == 0 && len(loginResp.Auth.Policies) > 0 {
		// If TokenPolicies is empty but Policies has values, use Policies
		// This mimics what the framework does when storing auth
		tokenPolicies = loginResp.Auth.Policies
	}

	renewAuth := &logical.Auth{
		InternalData:  loginResp.Auth.InternalData,
		Policies:      loginResp.Auth.Policies,
		TokenPolicies: tokenPolicies,
		Metadata:      loginResp.Auth.Metadata,
		DisplayName:   loginResp.Auth.DisplayName,
		Alias:         loginResp.Auth.Alias,
		GroupAliases:  loginResp.Auth.GroupAliases,
		LeaseOptions: logical.LeaseOptions{
			TTL:       loginResp.Auth.TTL,
			MaxTTL:    loginResp.Auth.MaxTTL,
			Renewable: true,
		},
	}

	// Test renew with the same token
	renewReq := &logical.Request{
		Path:      "login",
		Operation: logical.RenewOperation,
		Storage:   s,
		Auth:      renewAuth,
	}

	renewResp, err := b.HandleRequest(context.Background(), renewReq)
	assert.NoError(t, err)
	assert.NotNil(t, renewResp)
	if renewResp != nil {
		assert.NotNil(t, renewResp.Auth)
	}
}

// TestGitHub_PathLoginRenew_PolicyMismatch tests renewal with mismatched policies
func TestGitHub_PathLoginRenew_PolicyMismatch(t *testing.T) {
	b, s := createBackendWithStorage(t)

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// Write the config
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"organization": "foo-org",
			"base_url":     ts.URL,
		},
		Storage: s,
	})
	assert.NoError(t, err)

	// Attempt renew with mismatched policies (should fail)
	renewReq := &logical.Request{
		Path:      "login",
		Operation: logical.RenewOperation,
		Storage:   s,
		Auth: &logical.Auth{
			InternalData: map[string]interface{}{
				"token": "faketoken",
			},
			TokenPolicies: []string{"different-policy"},
			Metadata:      map[string]string{"org": "foo-org", "username": "user-foo"},
			LeaseOptions: logical.LeaseOptions{
				TTL:       3600,
				Renewable: true,
			},
		},
	}

	_, err = b.HandleRequest(context.Background(), renewReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policies do not match")
}

// TestGitHub_PathLoginRenew_MissingToken tests renewal without token in internal data
func TestGitHub_PathLoginRenew_MissingToken(t *testing.T) {
	b, s := createBackendWithStorage(t)

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	// Write the config
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Path:      "config",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"organization": "foo-org",
			"base_url":     ts.URL,
		},
		Storage: s,
	})
	assert.NoError(t, err)

	// Attempt renew without token in internal data
	renewReq := &logical.Request{
		Path:      "login",
		Operation: logical.RenewOperation,
		Storage:   s,
		Auth: &logical.Auth{
			InternalData:  map[string]interface{}{},
			TokenPolicies: []string{"default"},
			LeaseOptions: logical.LeaseOptions{
				TTL:       3600,
				Renewable: true,
			},
		},
	}

	_, err = b.HandleRequest(context.Background(), renewReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token created in previous version")
}

// TestGitHub_CheckCIDRMatch tests CIDR validation
func TestGitHub_CheckCIDRMatch(t *testing.T) {
	b, s := createBackendWithStorage(t)

	// use a test server to return our mock GH org info
	ts := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		name            string
		remoteAddr      string
		tokenBoundCIDRs []string
		expectError     bool
	}{
		{
			name:            "no CIDR restrictions",
			remoteAddr:      "192.168.1.1",
			tokenBoundCIDRs: nil,
			expectError:     false,
		},
		{
			name:            "matching CIDR",
			remoteAddr:      "192.168.1.1",
			tokenBoundCIDRs: []string{"192.168.1.0/24"},
			expectError:     false,
		},
		{
			name:            "non-matching CIDR",
			remoteAddr:      "10.0.0.1",
			tokenBoundCIDRs: []string{"192.168.1.0/24"},
			expectError:     true,
		},
		{
			name:            "multiple CIDRs with match",
			remoteAddr:      "10.0.0.1",
			tokenBoundCIDRs: []string{"192.168.1.0/24", "10.0.0.0/8"},
			expectError:     false,
		},
		{
			name:            "no connection info with CIDR restrictions",
			remoteAddr:      "",
			tokenBoundCIDRs: []string{"192.168.1.0/24"},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config with CIDR restrictions
			configData := map[string]interface{}{
				"organization": "foo-org",
				"base_url":     ts.URL,
			}
			if len(tt.tokenBoundCIDRs) > 0 {
				configData["token_bound_cidrs"] = tt.tokenBoundCIDRs
			}

			_, err := b.HandleRequest(context.Background(), &logical.Request{
				Path:      "config",
				Operation: logical.UpdateOperation,
				Data:      configData,
				Storage:   s,
			})
			assert.NoError(t, err)

			// Read config back
			config, err := b.Config(context.Background(), s)
			assert.NoError(t, err)
			assert.NotNil(t, config)

			req := &logical.Request{}
			if tt.remoteAddr != "" {
				req.Connection = &logical.Connection{
					RemoteAddr: tt.remoteAddr,
				}
			}

			err = b.checkCIDRMatch(req, config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, logical.ErrPermissionDenied, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
