# Development

## How to test

Start server

```bash
make auth-fido2 --always-make
export BAO_FIDO2_DEV_UI=$PWD/auth/fido2/ui/static/
bao server -dev -dev-root-token-id=root --dev-plugin-dir ./bin/
```

> [!NOTE]
> Usually the HTM, JavaScript and CSS files are bundled in the executable. This
> is annoying during development. Setting the `BAO_FIDO2_DEV_UI` env will cause
> the assets to be served from disk instead.

Enable and configure Plugin

```bash
bao auth enable -path=passkey -plugin-name=openbao-plugin-auth-fido2_linux_amd64_v1 plugin
bao write auth/passkey/config relying_party_id=localhost relying_party_origins=http://localhost:8200
```

Create another auth method, user and prepare for self-enrollment testing

```bash
bao auth enable userpass
bao write auth/userpass/users/alice password=bob4ever token_policies=read-kv
ENTITY_ID=$(bao login -format=json -method=userpass username=alice password=bob4ever | jq -r .auth.entity_id)

cat <<EOF | bao policy write super-user -
path "*" {
    capabilities = ["read", "create", "update", "patch", "delete", "list", "scan"]
}
EOF

bao write identity/entity/id/$ENTITY_ID policies=super-user

FIDO2_ACCESSOR=$(bao read --field=accessor sys/auth/passkey)
bao write identity/entity-alias \
    canonical_id=$ENTITY_ID \
    mount_accessor=$FIDO2_ACCESSOR \
    name=alice
```

Open the browser at <http://localhost:8200/> and login as alice (password
"bob4ever"). Now go to
<http://localhost:8200/v1/auth/passkey/internal/ui/register.html> and select
"Enroll Now" in the "Self-Enrollment" section.

In a new session, go to
<http://localhost:8200/v1/auth/passkey/internal/ui/login.html> and login.

## Helpful Links

- [https://ctap.dev/](ctap.dev) An online "Passkey Playground"
- As always [MDN has some good stuff on
  Passkeys](https://developer.mozilla.org/en-US/docs/Web/Security/Authentication/Passkeys)
