// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"reflect"
	"testing"

	"github.com/openbao/openbao/sdk/v2/framework"
)

func TestConfig_Update(t *testing.T) {

	cases := []struct {
		name    string
		new     *Config
		d       *framework.FieldData
		r       *Config
		changed bool
		err     bool
	}{
		{
			"empty",
			&Config{},
			nil,
			&Config{},
			false,
			false,
		},
		{
			"keeps_existing",
			&Config{
				Credentials: "creds",
			},
			nil,
			&Config{
				Credentials: "creds",
			},
			false,
			false,
		},
		{
			"overwrites_changes",
			&Config{
				Credentials: "creds",
			},
			&framework.FieldData{
				Raw: map[string]interface{}{
					"credentials": "foo",
				},
			},
			&Config{
				Credentials: "foo",
			},
			true,
			false,
		},
		{
			"overwrites_and_new",
			&Config{
				Credentials: "creds",
			},
			&framework.FieldData{
				Raw: map[string]interface{}{
					"credentials": "foo",
					"scopes":      "bar",
				},
			},
			&Config{
				Credentials: "foo",
				Scopes:      []string{"bar"},
			},
			true,
			false,
		},
		{
			"no_changes_order",
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			&framework.FieldData{
				Raw: map[string]interface{}{
					"scopes": "foo,bar",
				},
			},
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			false,
			false,
		},
		{
			"no_changes_caps",
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			&framework.FieldData{
				Raw: map[string]interface{}{
					"scopes": "FOO,baR",
				},
			},
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			false,
			false,
		},
		{
			"no_changes_dupes",
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			&framework.FieldData{
				Raw: map[string]interface{}{
					"scopes": "foo, foo, foo, bar",
				},
			},
			&Config{
				Scopes: []string{"bar", "foo"},
			},
			false,
			false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {

			if tc.d != nil {
				var b backend
				tc.d.Schema = b.pathConfig().Fields
			}

			changed, err := tc.new.Update(tc.d)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if changed != tc.changed {
				t.Errorf("expected %t to be %t", changed, tc.changed)
			}

			if v, exp := tc.new.Scopes, tc.r.Scopes; !reflect.DeepEqual(v, exp) {
				t.Errorf("expected %q to be %q", v, exp)
			}

			if v, exp := tc.new.Credentials, tc.r.Credentials; v != exp {
				t.Errorf("expected %q to be %q", v, exp)
			}
		})
	}
}
