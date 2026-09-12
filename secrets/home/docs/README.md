---
sidebar_label: Overview
description: >-
  The Home secrets engine provides per-identity private key/value storage,
  scoped to the caller's Identity entity rather than to their token.
---

# Home secrets engine

The Home secrets engine stores arbitrary secrets scoped to the caller's
[Identity entity](/docs/concepts/identity). It fulfills a long-standing
community request: the ability to store static secrets that belong to *you* —
your entity — rather than to the ephemeral token you happen to be using.

Think of it as a home directory for your secrets.

## Motivation

The built-in [Cubbyhole](/docs/secrets/cubbyhole) engine scopes storage to the
**current token**. Secrets vanish when the token expires, and they are invisible
to any other token — even tokens belonging to the same user. This is useful for
response-wrapping workflows, but it is a poor fit for persistent personal
secrets such as SSH keys, personal API tokens, or per-user configuration.

A commonly proposed alternative is to use templated policies with the KV engine:

```hcl
path "secret/data/users/{{identity.entity.id}}/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

While this works in simple cases, it pushes entity-scoped isolation into the
policy layer. Every operator must get the template exactly right, the secrets
are technically accessible to anyone with a sufficiently broad policy on
`secret/*`, and the approach does not compose well with complex policy
hierarchies.

The Home engine solves this at the storage layer instead. Each entity's data is
keyed by its entity UUID inside the engine's barrier storage, so no policy can
grant cross-entity access — the engine simply will not serve another entity's
data regardless of the caller's capabilities.

| Property | Cubbyhole | Home |
|---|---|---|
| Storage key | Token ID | Identity entity ID |
| Survives token renewal / rotation | No | **Yes** |
| Shared across tokens for same user | No | **Yes** |
| Requires an entity | No | **Yes** |
| Works with root token | Yes | No (by design) |

## Setup

The Home engine is an external plugin. Before it can be mounted it must be
built, placed in OpenBao's `plugin_directory`, and registered in the catalog.

### Build

```bash
git clone https://github.com/openbao/openbao-home-engine
cd openbao-home-engine
make build        # → bin/home-engine (current platform)
make release      # → cross-compiled binaries in bin/
```

The binary is statically linked (`CGO_ENABLED=0`) and has no external
dependencies.

### Register

```bash
cp bin/home-engine /etc/openbao/plugins/
chmod 755 /etc/openbao/plugins/home-engine

SHA256=$(sha256sum /etc/openbao/plugins/home-engine | cut -d' ' -f1)

bao plugin register \
    -sha256="$SHA256" \
    -command=home-engine \
    secret home
```

### Enable

```bash
bao secrets enable -path=home -description="Per-identity private storage" home
```

## Usage

After the engine is mounted and the caller has a token with a resolved Identity
entity, secrets can be written, read, deleted, and listed just like any other
key/value store.

```bash
# Write
bao write home/my-secret password=hunter2

# Read
bao read home/my-secret

# Delete
bao delete home/my-secret

# List
bao list home/
```

Nested paths are fully supported:

```bash
bao write home/work/aws access_key=AKIA... secret_key=...
bao list  home/work/
bao read  home/work/aws
```

## Policy example

Because isolation is enforced by the engine itself, a single broad policy is
sufficient for all users:

```hcl
path "home/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

There is no need for per-user path prefixes or templated policies. The engine
guarantees that entity A can never read or write entity B's data, regardless of
what ACL policies are in effect.

## How it works

When OpenBao routes a request to the Home engine it populates
`logical.Request.EntityID` with the UUID of the caller's Identity entity. The
engine prepends this UUID to every storage path:

```
<mount-barrier>/
└── <entity-uuid>/
    ├── my-secret
    ├── tokens
    └── work/
        └── aws
```

Two different tokens that resolve to the same entity UUID read and write the
same underlying keys. A token with **no** entity UUID — such as the root
token — is rejected with a permission-denied error.

## Security considerations

- **No cross-entity access.** The entity UUID in the storage path acts as an
  unforgeable isolation boundary enforced by the engine, independent of ACL
  policies.

- **Deleted entities.** If an entity is deleted from the Identity store, the
  corresponding Home data is not automatically cleaned up. An operator can purge
  orphaned data via `bao sys raw` or by disabling and re-enabling the mount.

- **Root tokens.** Root tokens intentionally cannot use this engine because they
  have no entity. This prevents accidental writes that would be invisible to the
  intended user.

- **Plugin integrity.** Register the plugin with the correct SHA-256 checksum.
  OpenBao will refuse to run a binary whose hash does not match the catalog
  entry.

- **Path traversal.** The engine rejects any path containing `..` components to
  prevent directory traversal attacks against the storage backend.

## API

The Home secrets engine has a full [HTTP API](/api-docs/secret/home).

## Authors

John Boero and Claude — buddies 4ever 🤝
