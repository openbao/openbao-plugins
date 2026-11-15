// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"fmt"
	"time"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/roottoken"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func (b *backend) pathSelfServiceEnroll(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, err := b.getSelfServiceAlias(req)
	if err != nil {
		return nil, err
	}

	otp, err := roottoken.GenerateOTP(16)
	if err != nil {
		return nil, err
	}

	token := fmt.Sprintf("v1.%s.%s", otp, alias.Name)
	tokenStoragePath := escapedPathJoin("v1", "enrollment-tokens", alias.Name, token)
	err = req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   tokenStoragePath,
		Value: []byte{},
	})
	if err != nil {
		return nil, err
	}

	resp := b.Secret(SecretTypeEnrollmentToken).Response(map[string]any{
		"token": token,
	}, map[string]any{
		"storage_path": tokenStoragePath,
	})
	resp.Secret.Renewable = false
	resp.Secret.TTL = 60 * time.Second

	return resp, nil
}
