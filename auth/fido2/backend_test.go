// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathJoin(t *testing.T) {
	parts := []string{"a", "b", "c/d", "x/../y", "%", "5%20d"}
	joined := escapedPathJoin(parts...)

	for i, part := range parts {
		var head string
		head, joined, _ = strings.Cut(joined, "/")
		assert.Equal(t, url.PathEscape(part), head, "part %d", i) // this should not be true in the general case (as e.g. + will not be encoded), see case below
		t.Logf("%d expect %q == %q // remaining %q %v", i, head, part, joined, parts)
	}

	assert.Empty(t, joined)

	nonEncoded := []string{"a", "asdf", "+", "$", "?&@$*&#"}
	assert.Equal(t, path.Join(nonEncoded...), escapedPathJoin(nonEncoded...)) // all chars without special meaning in storage paths => should be equivalent path.Join
}
