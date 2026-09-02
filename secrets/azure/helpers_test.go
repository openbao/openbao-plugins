// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-test/deep"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/openbao/openbao/sdk/v2/helper/jsonutil"
)

func init() {
	deep.CompareUnexportedFields = true
}

// assertErrorIsNil tests for non-nil errors
func assertErrorIsNil(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("\nunexpected error: %s", err.Error())
	}
}

// assertKeyExists asserts that the provided key exists within the map
func assertKeyExists(tb testing.TB, m map[string]interface{}, key string) {
	tb.Helper()
	if _, exists := m[key]; !exists {
		tb.Fatalf("key %s does not exist", key)
	}
}

// equal tests for deep equality
func equal(tb testing.TB, exp, act interface{}) {
	tb.Helper()
	if diff := deep.Equal(exp, act); diff != nil {
		tb.Fatal(diff)
	}
}

// encodeJSON is an optimistic JSON encoder that will
// panic on error.
func encodeJSON(data interface{}) string {
	d, err := jsonutil.EncodeJSON(data)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(d))
}

// compactJSON is a JSON compactor that works on strings
// and panics on any error
func compactJSON(s string) string {
	var b bytes.Buffer
	if err := json.Compact(&b, []byte(s)); err != nil {
		panic(err)
	}
	return b.String()
}

// generateUUID is an optimistic UUID source that will
// panic on error.
func generateUUID() string {
	u, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return u
}

// fakeSaveLoad will simulate the JSON encoding/decoding process that secrets will normally go
// through. If not done, some of the secret data will retain richer data types (e.g. pointers)
// than would normally be seen.
func fakeSaveLoad(s *logical.Secret) {
	enc := encodeJSON(s)

	if err := jsonutil.DecodeJSON([]byte(enc), s); err != nil {
		panic(err)
	}
}
