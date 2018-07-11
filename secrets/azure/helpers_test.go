package azuresecrets

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-test/deep"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/jsonutil"
)

func init() {
	deep.CompareUnexportedFields = true
}

// nilErr tests for non-nil errors
func nilErr(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("\nunexpected error: %s", err.Error())
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
