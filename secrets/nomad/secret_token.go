// Copyright (c) HashiCorp, Inc.
// Copyright (c) Jan Martens <jan@martens.eu.org>
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	SecretTokenType = "token"
)

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretTokenType,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "Request token",
			},
		},

		Renew:  b.secretTokenRenew,
		Revoke: b.secretTokenRevoke,
	}
}

func (b *backend) secretTokenRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	lease, err := b.LeaseConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		lease = &configLease{}
	}
	resp := &logical.Response{Secret: req.Secret}
	resp.Secret.TTL = lease.TTL
	resp.Secret.MaxTTL = lease.MaxTTL
	return resp, nil
}

func (b *backend) secretTokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	c, err := b.client(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if c == nil {
		return nil, fmt.Errorf("error getting Nomad client")
	}

	accessorIDRaw, ok := req.Secret.InternalData["accessor_id"]
	if !ok {
		return nil, fmt.Errorf("accessor_id is missing on the lease")
	}
	accessorID, ok := accessorIDRaw.(string)
	if !ok {
		return nil, errors.New("unable to convert accessor_id")
	}
	_, err = c.ACLTokens().Delete(accessorID, nil)
	if err != nil {
		statusError := api.UnexpectedResponseError{}

		if errors.As(err, &statusError) &&
			statusError.StatusCode() == 400 &&
			// Don't just rely on the status code, a 400 could have many causes (e.g. load balancer has briefly no backend)
			// So we additionally match the exact response body.
			// This might break in future versions of Nomad, but at least it's safe.
			statusError.Body() == fmt.Sprintf("Cannot delete nonexistent tokens: %s", accessorID) {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}

	return nil, nil
}
