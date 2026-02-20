# openbao-home-engine

An [OpenBao](https://openbao.org) external secrets-engine plugin that provides
**per-identity private key/value storage** — a spiritual successor to the
built-in Cubbyhole engine, but scoped to the caller's **Identity entity** rather
than to the caller's token.

---

## Motivation

The built-in [Cubbyhole](https://openbao.org/docs/secrets/cubbyhole/) engine is
scoped to the **current token**: secrets are destroyed when the token expires,
and they are invisible to any other token — even tokens belonging to the same
user. This is intentional for response-wrapping workflows, but it is a poor fit
for user-facing "personal secrets" (e.g. SSH keys, personal API tokens,
per-user configuration).

The **Home engine** solves this by scoping storage to the caller's
[Identity entity](https://openbao.org/docs/concepts/identity/) instead.
Advantages:

| Property | Cubbyhole | Home |
|---|---|---|
| Storage key | Token ID | Identity entity ID |
| Survives token renewal / rotation | ❌ | ✅ |
| Shared across tokens for same user | ❌ | ✅ |
| Requires an entity | No | **Yes** |
| Works with root token | Yes | ❌ (by design) |

---

## Requirements

- OpenBao ≥ 2.x (SDK v2)
- Go 1.22+
- The caller's token must have a resolved Identity entity (any non-root token
  created through an auth method that creates entities will satisfy this)

---

## Building

```bash
git clone https://github.com/openbao/openbao-home-engine
cd openbao-home-engine

# Build for the current platform
make build
# → bin/home-engine

# Cross-compile release binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
make release
```

---

## Installation

1. **Copy the binary** to OpenBao's `plugin_directory` (set in `config.hcl`):

   ```bash
   cp bin/home-engine /etc/openbao/plugins/
   chmod 755 /etc/openbao/plugins/home-engine
   ```

2. **Register the plugin** in the catalog:

   ```bash
   SHA256=$(sha256sum /etc/openbao/plugins/home-engine | cut -d' ' -f1)

   bao plugin register \
       -sha256="$SHA256" \
       -command=home-engine \
       secret home
   ```

3. **Enable a mount**:

   ```bash
   bao secrets enable -path=home -description="Per-identity private storage" home
   ```

---

## Usage

### Write a secret

```bash
bao kv put home/my-ssh-key private_key=@~/.ssh/id_ed25519
bao kv put home/tokens github_pat=ghp_xxx openai_key=sk-xxx
```

### Read a secret

```bash
bao kv get home/my-ssh-key
bao kv get home/tokens
```

### Delete a secret

```bash
bao kv delete home/my-ssh-key
```

### List secrets

```bash
bao kv list home/
```

### Nested paths are supported

```bash
bao kv put home/work/aws access_key=AKIA... secret_key=...
bao kv list home/work/
bao kv get  home/work/aws
```

---

## Policy example

Because each user's data is isolated by their entity ID in the storage layer,
a single broad policy is safe to grant to all users:

```hcl
# Allow any authenticated user with an entity to manage their own home storage.
path "home/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

You do **not** need per-user path prefixes in the policy — the engine enforces
isolation automatically.

---

## How it works

When OpenBao routes a request to the Home engine, it populates
`logical.Request.EntityID` with the UUID of the Identity entity associated with
the caller's token. The engine prepends this UUID to every storage path:

```
<mount-uuid>/  (barrier view provided by OpenBao)
└── <entity-uuid>/
    ├── my-ssh-key
    ├── tokens
    └── work/
        └── aws
```

Two different tokens with the same entity UUID will therefore read and write the
same underlying keys. A token with **no** entity UUID (e.g. the root token) is
rejected with a permission-denied error.

---

## Security considerations

* **No cross-entity access**: Entity A can never read or write entity B's data.
  The entity UUID in the storage path acts as an unforgeable isolation boundary
  enforced by the engine, independent of ACL policies.
* **Deleted entities**: If an entity is deleted in the Identity store, the
  corresponding storage is not automatically cleaned up. An operator can purge
  orphaned data via `bao sys raw` or by disabling and re-enabling the mount.
* **Root tokens**: Root tokens intentionally cannot use this engine because they
  have no entity. This prevents accidental writes that would be invisible to the
  intended user.
* **Plugin integrity**: Register the plugin with the correct SHA-256 checksum.
  OpenBao will refuse to run a binary whose hash does not match the catalog entry.

---

## Running the tests

```bash
make test
```

The tests use OpenBao SDK's `logical.InmemStorage` and exercise write/read,
entity isolation, shared access for the same entity, delete, list, nested paths,
and the no-entity rejection.

---

## License

Mozilla Public License 2.0 — same as OpenBao itself.
