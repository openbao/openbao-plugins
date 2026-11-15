// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// LIST

func (b *backend) pathSelfServiceList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, err := b.getSelfServiceAlias(req)
	if err != nil {
		return nil, err
	}

	return b.keysList(ctx, req, alias.Name)
}

func (b *backend) pathAdminList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	return b.keysList(ctx, req, alias.(string))
}

func (b *backend) keysList(ctx context.Context, req *logical.Request, alias string) (*logical.Response, error) {
	key := escapedPathJoin("v1", "credentials", alias) + "/"
	keys, err := req.Storage.List(ctx, key)
	if err != nil {
		return nil, err
	}

	details := make(map[string]map[string]any, len(keys))
	for _, id := range keys {
		storageEntry, err := req.Storage.Get(ctx, path.Join(key, id))
		if err != nil {
			return nil, err
		}

		var name string
		if storageEntry == nil {
			name = id
		} else {
			var entry credentialStorageEntry
			err = storageEntry.DecodeJSON(&entry)
			if err != nil {
				return nil, err
			}
			if entry.DisplayName == "" {
				name = id
			} else {
				name = entry.DisplayName
			}
		}

		details[id] = map[string]any{
			"name": name,
		}
	}

	return &logical.Response{
		Data: map[string]any{
			"keys":    keys,
			"details": details,
		},
	}, nil
}

// READ

func (b *backend) pathSelfServiceRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, err := b.getSelfServiceAlias(req)
	if err != nil {
		return nil, err
	}

	return b.keysRead(ctx, req, data, alias.Name)
}

func (b *backend) pathAdminRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	return b.keysRead(ctx, req, data, alias.(string))
}

func (b *backend) keysRead(ctx context.Context, req *logical.Request, data *framework.FieldData, alias string) (*logical.Response, error) {
	id, ok := data.GetOk("id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "id is required")
	}

	storageEntry, err := req.Storage.Get(ctx, escapedPathJoin("v1", "credentials", alias, id.(string)))
	if err != nil {
		return nil, err
	}

	if storageEntry == nil {
		return nil, logical.CodedError(http.StatusNotFound, "key not found")
	}

	var name string
	var entry credentialStorageEntry
	err = storageEntry.DecodeJSON(&entry)
	if err != nil {
		return nil, err
	}
	if entry.DisplayName == "" {
		name = id.(string)
	} else {
		name = entry.DisplayName
	}

	return &logical.Response{
		Data: map[string]any{
			"name":       name,
			"public_key": entry.CredentialBytes,
		},
	}, nil
}

// PATCH

func (b *backend) pathSelfServicePatch(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, err := b.getSelfServiceAlias(req)
	if err != nil {
		return nil, err
	}

	return b.keysPatch(ctx, req, data, alias.Name)
}

func (b *backend) pathAdminPatch(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	return b.keysPatch(ctx, req, data, alias.(string))
}

func (b *backend) keysPatch(ctx context.Context, req *logical.Request, data *framework.FieldData, alias string) (*logical.Response, error) {
	id, ok := data.GetOk("id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "id is required")
	}

	name, ok := data.GetOk("name")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "empty patch")
	}

	err := logical.WithTransaction(ctx, req.Storage, func(s logical.Storage) error {
		key := escapedPathJoin("v1", "credentials", alias, id.(string))
		storageEntry, err := s.Get(ctx, key)
		if err != nil {
			return err
		}

		if storageEntry == nil {
			return logical.CodedError(http.StatusNotFound, "credential not found")
		}

		var entry credentialStorageEntry
		err = storageEntry.DecodeJSON(&entry)
		if err != nil {
			return err
		}
		entry.DisplayName = name.(string)

		storageEntry.Value, err = json.Marshal(entry)
		if err != nil {
			return err
		}

		return s.Put(ctx, storageEntry)
	})

	if err != nil {
		return nil, asCodedError(err)
	}

	return nil, nil
}

// DELETE

func (b *backend) pathSelfServiceDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, err := b.getSelfServiceAlias(req)
	if err != nil {
		return nil, err
	}

	return b.keysDelete(ctx, req, data, alias.Name)
}

func (b *backend) pathAdminDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	alias, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	return b.keysDelete(ctx, req, data, alias.(string))
}

func (b *backend) keysDelete(ctx context.Context, req *logical.Request, data *framework.FieldData, alias string) (*logical.Response, error) {
	id, ok := data.GetOk("id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "id is required")
	}

	key := escapedPathJoin("v1", "credentials", alias, id.(string))
	err := req.Storage.Delete(ctx, key)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// helper

func (b *backend) getSelfServiceAlias(req *logical.Request) (*logical.Alias, error) {
	entity, err := b.System().EntityInfo(req.EntityID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, logical.CodedError(http.StatusNotFound, "entity %q not found", req.EntityID)
	}

	aliasIdx := slices.IndexFunc(entity.Aliases, func(alias *logical.Alias) bool {
		//return alias.MountAccessor == req.MountAccessor
		return strings.HasPrefix(alias.MountAccessor, "auth_openbao-plugin-auth-fido2") // FIXME: workaround until https://github.com/openbao/openbao/pull/4434 is released
	})

	if aliasIdx == -1 {
		return nil, fmt.Errorf("no alias available for self-service, please ask your authentication admin to create an alias for mount-accessor %q and entity %q", req.MountAccessor, req.EntityID)
	}

	return entity.Aliases[aliasIdx], nil
}
