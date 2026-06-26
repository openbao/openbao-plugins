package github

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
)

// testAccPreCheck checks if required environment variables are set for acceptance tests
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("GITHUB_TOKEN"); v == "" {
		t.Skip("GITHUB_TOKEN must be set for acceptance tests")
	}

	if v := os.Getenv("GITHUB_USER"); v == "" {
		t.Skip("GITHUB_USER must be set for acceptance tests")
	}

	if v := os.Getenv("GITHUB_ORG"); v == "" {
		t.Skip("GITHUB_ORG must be set for acceptance tests")
	}

	if v := os.Getenv("GITHUB_BASEURL"); v == "" {
		t.Skip("GITHUB_BASEURL must be set for acceptance tests (use 'https://api.github.com' if you don't know what you're doing)")
	}
}

// createBackend creates a backend for testing
func createBackend(t *testing.T) logical.Backend {
	defaultLeaseTTLVal := time.Hour * 24
	maxLeaseTTLVal := time.Hour * 24 * 32
	b, err := Factory(context.Background(), &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
	})
	if err != nil {
		t.Fatalf("Unable to create backend: %s", err)
	}
	return b
}

// writeConfig writes configuration to the backend
func writeConfig(t *testing.T, b logical.Backend, storage logical.Storage, data map[string]interface{}) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Data:      data,
		Storage:   storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Error writing config: %v", err)
	}
	if resp != nil && resp.IsError() {
		t.Fatalf("Error writing config: %v", resp.Error())
	}
}

// writeTeamMapping writes team mapping to the backend
func writeTeamMapping(t *testing.T, b logical.Backend, storage logical.Storage, team string, policy string) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "map/teams/" + team,
		Data: map[string]interface{}{
			"value": policy,
		},
		Storage: storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Error writing team mapping: %v", err)
	}
	if resp != nil && resp.IsError() {
		t.Fatalf("Error writing team mapping: %v", resp.Error())
	}
}

// writeUserMapping writes user mapping to the backend
func writeUserMapping(t *testing.T, b logical.Backend, storage logical.Storage, user string, policy string) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "map/users/" + user,
		Data: map[string]interface{}{
			"value": policy,
		},
		Storage: storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Error writing user mapping: %v", err)
	}
	if resp != nil && resp.IsError() {
		t.Fatalf("Error writing user mapping: %v", resp.Error())
	}
}

// performLogin performs a login operation and returns the response
func performLogin(t *testing.T, b logical.Backend, storage logical.Storage, token string) *logical.Response {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "login",
		Data: map[string]interface{}{
			"token": token,
		},
		Storage:         storage,
		Unauthenticated: true,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Error performing login: %v", err)
	}
	return resp
}

// checkAuth verifies that the auth response contains the expected policies
func checkAuth(t *testing.T, resp *logical.Response, expectedPolicies []string) {
	if resp == nil || resp.Auth == nil {
		t.Fatal("No auth in response")
	}

	expected := make([]string, len(expectedPolicies))
	copy(expected, expectedPolicies)
	sort.Strings(expected)

	actual := make([]string, len(resp.Auth.Policies))
	copy(actual, resp.Auth.Policies)
	sort.Strings(actual)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Invalid policies: expected %#v, got %#v", expected, actual)
	}
}

func TestBackend_Config(t *testing.T) {
	testAccPreCheck(t)

	b, storage := setupTestBackendForConfig(t)
	token := os.Getenv("GITHUB_TOKEN")
	org := os.Getenv("GITHUB_ORG")

	t.Run("DefaultTTL", func(t *testing.T) {
		testDefaultTTLConfig(t, b, storage, org, token)
	})

	t.Run("CustomTTL", func(t *testing.T) {
		testCustomTTLConfig(t, b, storage, org, token)
	})

	t.Run("ExceedingMaxTTL", func(t *testing.T) {
		testExceedingMaxTTLConfig(t, b, storage, org, token)
	})
}

// setupTestBackendForConfig creates a backend with specific TTL settings for config testing
func setupTestBackendForConfig(t *testing.T) (logical.Backend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 24
	maxLeaseTTLVal := time.Hour * 24 * 2

	b, err := Factory(context.Background(), &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
	})
	if err != nil {
		t.Fatalf("Unable to create backend: %s", err)
	}

	storage := &logical.InmemStorage{}
	return b, storage
}

// testDefaultTTLConfig tests backend configuration with default TTL values
func testDefaultTTLConfig(t *testing.T, b logical.Backend, storage logical.Storage, org, token string) {
	// Write config with no TTL specified
	writeConfig(t, b, storage, map[string]interface{}{
		"organization": org,
		"ttl":          "",
		"max_ttl":      "",
	})

	// Perform login and verify default TTL is applied
	resp := performLoginWithCheck(t, b, storage, token)

	expectedTTL := 24 * time.Hour
	if resp.Auth.TTL != expectedTTL {
		t.Fatalf("TTL mismatched. Expected: %v Actual: %v", expectedTTL, resp.Auth.TTL)
	}
}

// testCustomTTLConfig tests backend configuration with custom TTL values
func testCustomTTLConfig(t *testing.T, b logical.Backend, storage logical.Storage, org, token string) {
	// Write config with custom TTL
	writeConfig(t, b, storage, map[string]interface{}{
		"organization": org,
		"ttl":          "1h",
		"max_ttl":      "2h",
	})

	// Perform login and verify custom TTL is applied
	resp := performLoginWithCheck(t, b, storage, token)

	expectedTTL := time.Hour
	if resp.Auth.TTL != expectedTTL {
		t.Fatalf("TTL mismatched. Expected: %v Actual: %v", expectedTTL, resp.Auth.TTL)
	}
}

// testExceedingMaxTTLConfig tests backend configuration with TTL exceeding system max
func testExceedingMaxTTLConfig(t *testing.T, b logical.Backend, storage logical.Storage, org, token string) {
	// Write config with TTL exceeding max TTL
	writeConfig(t, b, storage, map[string]interface{}{
		"organization": org,
		"ttl":          "50h",
		"max_ttl":      "50h",
	})

	// Perform login and verify TTL is capped at system max
	resp := performLoginWithCheck(t, b, storage, token)

	// This should be capped at the system max TTL
	expectedTTL := 48 * time.Hour // System max TTL
	if resp.Auth.TTL != expectedTTL {
		t.Fatalf("TTL mismatched. Expected: %v Actual: %v", expectedTTL, resp.Auth.TTL)
	}
}

// performLoginWithCheck performs a login request with full error checking
func performLoginWithCheck(t *testing.T, b logical.Backend, storage logical.Storage, token string) *logical.Response {
	resp := performLogin(t, b, storage, token)

	if resp == nil {
		t.Fatal("Expected a response but got nil")
	}
	if resp.IsError() {
		t.Fatalf("Login failed: %v", resp.Error())
	}

	return resp
}

func TestBackend_basic(t *testing.T) {
	testAccPreCheck(t)

	b := createBackend(t)
	storage := &logical.InmemStorage{}
	token := os.Getenv("GITHUB_TOKEN")
	org := os.Getenv("GITHUB_ORG")
	user := os.Getenv("GITHUB_USER")
	baseURL := os.Getenv("GITHUB_BASEURL")

	// Test 1: Basic configuration with lowercase organization
	t.Run("BasicConfigLowercase", func(t *testing.T) {
		// Write config
		writeConfig(t, b, storage, map[string]interface{}{
			"organization":   org,
			"token_policies": []string{"abc"},
		})

		// Write team mappings
		writeTeamMapping(t, b, storage, "default", "fakepol")
		writeTeamMapping(t, b, storage, "oWnErs", "fakepol")

		// Perform login and check auth
		resp := performLogin(t, b, storage, token)
		checkAuth(t, resp, []string{"default", "abc", "fakepol"})
	})

	// Test 2: Configuration with uppercase organization
	t.Run("BasicConfigUppercase", func(t *testing.T) {
		// Write config with uppercase organization
		writeConfig(t, b, storage, map[string]interface{}{
			"organization":   strings.ToUpper(org),
			"token_policies": []string{"abc"},
		})

		// Write team mappings
		writeTeamMapping(t, b, storage, "default", "fakepol")
		writeTeamMapping(t, b, storage, "oWnErs", "fakepol")

		// Perform login and check auth
		resp := performLogin(t, b, storage, token)
		checkAuth(t, resp, []string{"default", "abc", "fakepol"})
	})

	// Test 3: Configuration with base URL
	t.Run("ConfigWithBaseURL", func(t *testing.T) {
		// Write config with base URL
		writeConfig(t, b, storage, map[string]interface{}{
			"organization": org,
			"base_url":     baseURL,
		})

		// Write team mappings
		writeTeamMapping(t, b, storage, "default", "fakepol")
		writeTeamMapping(t, b, storage, "oWnErs", "fakepol")

		// Perform login and check auth
		resp := performLogin(t, b, storage, token)
		checkAuth(t, resp, []string{"default", "abc", "fakepol"})
	})

	// Test 4: User policy mapping
	t.Run("UserPolicyMapping", func(t *testing.T) {
		// Write config
		writeConfig(t, b, storage, map[string]interface{}{
			"organization":   org,
			"token_policies": []string{"abc"},
		})

		// Write team mappings
		writeTeamMapping(t, b, storage, "default", "fakepol")

		// Write user mapping
		writeUserMapping(t, b, storage, user, "userpolicy")

		// Perform login and check auth (should include user policy)
		resp := performLogin(t, b, storage, token)
		checkAuth(t, resp, []string{"default", "abc", "fakepol", "userpolicy"})
	})
}

// TestFactory tests the Factory function
func TestFactory(t *testing.T) {
	defaultLeaseTTLVal := time.Hour * 24
	maxLeaseTTLVal := time.Hour * 24 * 32

	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
	}

	// Test successful backend creation
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("Factory() returned error: %v", err)
	}
	if b == nil {
		t.Fatal("Factory() returned nil backend")
	}

	// Cast to our backend type to verify structure
	backend, ok := b.(*backend)
	if !ok {
		t.Fatalf("backend should be of type *backend, got %T", b)
	}
	if backend.TeamMap == nil {
		t.Error("backend.TeamMap should not be nil")
	}
	if backend.UserMap == nil {
		t.Error("backend.UserMap should not be nil")
	}
}

// TestFactory_WithConfig tests Factory with minimal valid config
func TestFactory_WithConfig(t *testing.T) {
	// Test with minimal valid config (no System view)
	config := &logical.BackendConfig{
		Logger: nil,
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("Factory() returned error: %v", err)
	}
	if b == nil {
		t.Fatal("Factory() returned nil backend")
	}
}
