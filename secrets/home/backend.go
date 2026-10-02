// Package backend implements the Home secrets engine for OpenBao.
//
// The Home engine works like the built-in Cubbyhole engine, but instead of
// scoping storage to the current *token*, it scopes storage to the caller's
// *Identity entity*. This means:
//
//   - Secrets written via any token belonging to the same entity are visible
//     across all of those tokens.
//   - The storage path is derived from req.EntityID (the resolved Identity
//     entity UUID), not from req.ClientToken.
//   - If the caller has no Identity entity (e.g. a root token) the request is
//     rejected, because there is no stable identity to anchor storage to.
//
// The engine is intended to be registered as an external plugin and mounted
// at any path (the conventional choice is "home/").
package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	// backendHelp is the top-level help text returned by bao path-help home/
	backendHelp = `
The Home secrets engine provides per-identity private key/value storage.

It behaves like the built-in Cubbyhole engine, but storage is keyed by the
caller's Identity entity ID rather than by the caller's token. This means
that any token associated with the same Identity entity shares the same
storage space, and that the data survives token rotation.

A caller must have a resolved Identity entity to use this engine. Requests
from tokens with no entity (such as the root token) are rejected.

Paths written here are only accessible to callers that share the same
entity, subject to normal ACL policy.
`
)

// HomeBackend holds any per-mount state. Because storage is derived entirely
// from the incoming request's EntityID, there is no persistent configuration.
type HomeBackend struct {
	*framework.Backend
}

// Factory is the logical.Factory function registered in the plugin catalog.
// OpenBao calls this when a new mount of the "home" plugin type is created.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := &HomeBackend{}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        strings.TrimSpace(backendHelp),
		Paths:       b.paths(),
		Secrets:     []*framework.Secret{},

		// RunningVersion is reported back to OpenBao so the plugin version
		// shows up in `bao plugin list` / `bao secrets list`.
		RunningVersion: "v0.1.0",
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

// paths returns all framework.Path definitions handled by this backend.
func (b *HomeBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			// Match any path beneath the mount point.
			Pattern: framework.MatchAllRegex("path"),

			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:        framework.TypeString,
					Description: "Specifies the path within the home storage.",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead,
					Summary:  "Read a secret from the caller's home storage.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Write a secret to the caller's home storage.",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Create a secret in the caller's home storage.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete,
					Summary:  "Delete a secret from the caller's home storage.",
				},
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "List secrets under a path in the caller's home storage.",
				},
			},

			ExistenceCheck: b.handleExistenceCheck,

			HelpSynopsis: "Read/write/delete/list private secrets scoped to your Identity entity.",
			HelpDescription: `
Read, write, delete, and list secrets that are private to the caller's
Identity entity. All tokens associated with the same entity share access
to the same storage. If the caller has no entity (e.g. the root token)
the request will be rejected.
`,
		},
	}
}

// storagePath derives the barrier-relative storage path for the given request.
// The path layout is: <entityID>/<userPath>
//
// The EntityID comes from req.EntityID, which OpenBao populates with the
// resolved Identity entity UUID for the caller's token.
func storagePath(req *logical.Request, userPath string) (string, error) {
	if req.EntityID == "" {
		return "", errors.New(
			"home: caller has no Identity entity; " +
				"the home engine requires an entity to derive a stable storage path. " +
				"Root tokens and tokens without an associated entity cannot use this engine",
		)
	}

	// Sanitise the user-supplied path component to prevent path traversal.
	userPath = strings.Trim(userPath, "/")
	if strings.Contains(userPath, "..") {
		return "", fmt.Errorf("home: path %q contains illegal '..' component", userPath)
	}

	if userPath == "" {
		return req.EntityID + "/", nil
	}
	return req.EntityID + "/" + userPath, nil
}

// handleRead reads a key from the entity's home storage.
func (b *HomeBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	userPath, _ := data.Get("path").(string)

	spath, err := storagePath(req, userPath)
	if err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	// Trailing slash means the caller is trying to read a "directory"; redirect
	// to a list so they get a useful message.
	if strings.HasSuffix(spath, "/") {
		return logical.ErrorResponse("cannot read a path that ends with '/'; use LIST to enumerate keys"), nil
	}

	entry, err := req.Storage.Get(ctx, spath)
	if err != nil {
		return nil, fmt.Errorf("home: storage read error: %w", err)
	}
	if entry == nil {
		return nil, nil // 404
	}

	var rawData map[string]interface{}
	if err := entry.DecodeJSON(&rawData); err != nil {
		return nil, fmt.Errorf("home: failed to decode stored value: %w", err)
	}

	return &logical.Response{Data: rawData}, nil
}

// handleWrite creates or replaces a key in the entity's home storage.
func (b *HomeBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	userPath, _ := data.Get("path").(string)

	spath, err := storagePath(req, userPath)
	if err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	if strings.HasSuffix(spath, "/") {
		return logical.ErrorResponse("cannot write to a path that ends with '/'"), nil
	}

	// req.Data holds the raw key/value pairs supplied by the caller.
	if len(req.Data) == 0 {
		return logical.ErrorResponse("missing data to write"), nil
	}

	entry, err := logical.StorageEntryJSON(spath, req.Data)
	if err != nil {
		return nil, fmt.Errorf("home: failed to encode data: %w", err)
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, fmt.Errorf("home: storage write error: %w", err)
	}

	return nil, nil
}

// handleDelete removes a key from the entity's home storage.
func (b *HomeBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	userPath, _ := data.Get("path").(string)

	spath, err := storagePath(req, userPath)
	if err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	if strings.HasSuffix(spath, "/") {
		return logical.ErrorResponse("cannot delete a path that ends with '/'"), nil
	}

	if err := req.Storage.Delete(ctx, spath); err != nil {
		return nil, fmt.Errorf("home: storage delete error: %w", err)
	}

	return nil, nil
}

// handleList enumerates keys under a path prefix in the entity's home storage.
func (b *HomeBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	userPath, _ := data.Get("path").(string)

	spath, err := storagePath(req, userPath)
	if err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	// Ensure the prefix we scan ends with "/" so that List() returns children.
	if !strings.HasSuffix(spath, "/") {
		spath += "/"
	}

	keys, err := req.Storage.List(ctx, spath)
	if err != nil {
		return nil, fmt.Errorf("home: storage list error: %w", err)
	}

	return logical.ListResponse(keys), nil
}

// handleExistenceCheck lets the framework determine whether an existing entry
// is present, which drives whether Create vs Update ACL capabilities are
// checked. It implements framework.ExistenceFunc: func(context.Context,
// *logical.Request, *FieldData) (bool, error).
func (b *HomeBackend) handleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	userPath, _ := data.Get("path").(string)

	spath, err := storagePath(req, userPath)
	if err != nil {
		// Treat as "not found" so the framework falls through to the handler,
		// which will surface the proper error.
		return false, nil
	}

	entry, err := req.Storage.Get(ctx, spath)
	if err != nil {
		return false, err
	}

	return entry != nil, nil
}
