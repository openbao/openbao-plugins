package azureauth

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/version"
)

func TestJsonTime(t *testing.T) {
	type a struct {
		Time jsonTime `json:"time"`
	}

	now := time.Now().Round(time.Second)
	test1Raw := fmt.Sprintf(`{"time":%d}`, now.Unix())
	test1 := new(a)
	if err := json.Unmarshal([]byte(test1Raw), test1); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !now.Equal(time.Time(test1.Time)) {
		t.Fatalf("expected: %s, got: %s", now, time.Time(test1.Time))
	}

	test2Raw := fmt.Sprintf(`{"time":%f}`, float64(now.Unix()))
	test2 := new(a)
	if err := json.Unmarshal([]byte(test2Raw), test2); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !now.Equal(time.Time(test2.Time)) {
		t.Fatalf("expected: %s, got: %s", now, time.Time(test2.Time))
	}
}

func TestUserAgent(t *testing.T) {
	// VersionMetadata contains the version of Vault, typically "ent" or "prem" etc
	// for enterprise versions, and "" for OSS version. Dev versions will contain
	// "dev"
	// GUID
	// 15cd22ce-24af-43a4-aa83-4c1a36a4b177  Vault OSS
	//
	// b2c13ec1-60e8-4733-9a76-88dbb2ce2471  Vault Ent

	testCases := map[string]struct {
		meta     string
		expected string
	}{
		"none": {
			meta:     "",
			expected: ossVaultGUID,
		},
		"dev": {
			meta:     "dev",
			expected: ossVaultGUID,
		},
		"ent": {
			meta:     "ent",
			expected: entVaultGUID,
		},
		"prem": {
			meta:     "prem.hsm",
			expected: entVaultGUID,
		},
		"unknown": {
			meta:     "glhf",
			expected: entVaultGUID,
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			version.VersionMetadata = tc.meta
			userAgentStr := userAgent()
			if !strings.Contains(userAgentStr, tc.expected) {
				t.Fatalf("expected userAgent string to contain (%s), got: %s", tc.expected, userAgentStr)
			}
		})
	}
}
