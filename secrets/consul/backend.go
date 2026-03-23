// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package consul

import (
	"context"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const operationPrefixConsul = "consul"

// pluginVersion is used to report a specific version to Vault.
var pluginVersion = ""

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func Backend() *backend {
	var b backend
	b.Backend = &framework.Backend{
		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"config/access",
			},
		},

		Paths: []*framework.Path{
			pathConfigAccess(&b),
			pathListRoles(&b),
			pathRoles(&b),
			pathToken(&b),
		},

		Secrets: []*framework.Secret{
			secretToken(&b),
		},
		BackendType:    logical.TypeLogical,
		RunningVersion: pluginVersion,
	}

	return &b
}

type backend struct {
	*framework.Backend
}
