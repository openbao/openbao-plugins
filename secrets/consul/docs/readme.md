# Consul secrets engine

Consul is a service networking platform featuring service discovery, service
mesh, API gateway and configuration management. The Consul secrets engine for
OpenBao generates [Consul][consul] ACL tokens dynamically based on pre-existing
Consul ACL policies.

This page will show a quick start for this secrets engine. For detailed
documentation on every path, use `bao path-help` after mounting the secrets
engine.

> [!TIP]
> See the Consul Agent [config documentation][consul-bootstrap-acl] for details
> on how to enable Consul's ACL system.

> [!NOTE]
> This documentation assumes you are using at least Consul version 1.4 (released
> 2018). Older version are not supported.

## Setup

Most secrets engines must be configured in advance before they can perform their
functions. These steps are usually completed by an operator or configuration
management tool.

1. (Optional) If you're only looking to set up a quick test environment, you can
    start a Consul Agent in dev mode in a separate terminal window.

    ```shell-session
    $ consul agent -dev -hcl "acl { enabled = true }"
    ```

1. Enable the Consul secrets engine:

    ```shell-session
    $ bao secrets enable consul
    Success! Enabled the consul secrets engine at: consul/
    ```

    By default, the secrets engine will mount at the name of the engine. To
    enable the secrets engine at a different path, use the `-path` argument.

1. Configure OpenBao to connect and authenticate to Consul.

    OpenBao can bootstrap the Consul ACL system automatically if it is enabled
    and hasn't already been bootstrapped. If you have already bootstrapped the
    ACL system, then you will need to provide OpenBao with a management token.
    This can either be the bootstrap token or another management token you've
    created yourself.

    - Configuring OpenBao without previously bootstrapping the Consul ACL system:

        ```shell-session
        $ bao write consul/config/access \
            address="127.0.0.1:8500"
        Success! Data written to: consul/config/access
        ```

        OpenBao will silently store the bootstrap token as the configuration
        token when it performs the automatic bootstrap; it will not be presented
        to the user. If you need another management token, you will need to
        generate one by writing a OpenBao role with the `global-management`
        policy and then reading new credentials back from it.

    - Configuring OpenBao after manually bootstrapping the Consul ACL system:

        ```shell-session
        $ CONSUL_HTTP_TOKEN="<bootstrap-token>" consul acl token create -policy-name="global-management"
        AccessorID:   aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
        SecretID:     ffffffff-ffff-ffff-ffff-ffffffffffff
        Description:
        Local:        false
        Create Time:  2025-01-01 10:10:10.124356789 +0000 UTC
        Policies:
            00000000-0000-0000-0000-000000000001 - global-management
        ```

        ```shell-session
        $ bao write consul/config/access \
            address="127.0.0.1:8500" \
            token="ffffffff-ffff-ffff-ffff-ffffffffffff"
        Success! Data written to: consul/config/access
        ```

## Usage

### Configure Roles

Configure a role that maps a name in OpenBao to a Consul ACL policy. You can
either provide a list of policies or roles, or a set of service or node
identities. When users generate credentials, they are generated against this
role.

- Attach [a Consul node identity][consul-node-identity] to the role
  (requires at least Consul 1.8):

    ```shell-session
    $ bao write consul/roles/my-role \
        node_identities="server-1:dc1" \
        node_identities="server-2:dc1"
    Success! Data written to: consul/roles/my-role
    ```

- Attach [a Consul role][consul-role] to the role (requires at least Consul
  1.5):

    ```shell-session
    $ bao write consul/roles/my-role consul_roles="api-server"
    Success! Data written to: consul/roles/my-role
    ```

- Attach [a Consul service identity][consul-service-identity] to the role
   (requires at least Consul 1.5):

   ```shell-session
   $ bao write consul/roles/my-role \
   service_identities="myservice-1:dc1,dc2" \
   service_identities="myservice-2:dc1"
   Success! Data written to: consul/roles/my-role
   ```

- Attach [a Consul policy][consul-policy]:

   ```shell-session
   $ bao write consul/roles/my-role consul_policies="readonly"
   Success! Data written to: consul/roles/my-role
   ```

You may further limit a role's access by adding the optional parameters
`consul_namespace` and `partition` (both require a Consul Enterprise license).
Please refer to Consul's [namespace documentation][consul-namespaces] and [admin
partition documentation][consul-partitions] for further information about these
features.

- For Consul version 1.11 and above, link an admin partition to a role:

   ```shell-session
   $ bao write consul/roles/my-role consul_roles="admin-management" partition="admin1"
   Success! Data written to: consul/roles/my-role
   ```

- For Consul versions 1.7 and above, link a Consul namespace to the role:

   ```shell-session
   $ bao write consul/roles/my-role consul_roles="namespace-management" consul_namespace="ns1"
   Success! Data written to: consul/roles/my-role
   ```

To learn about further configurations options like TTL and and local only tokens
use `bao path-help consul/roles/`.

### Generate Tokens

After the secrets engine is configured, at least one role is defined and a
user/machine has a OpenBao token with the proper permission, it can generate
credentials.

Generate a new credential by reading from the `/creds` endpoint with the name
of the role:

```shell-session
$ bao read consul/creds/my-role
Key                 Value
---                 -----
lease_id            consul/creds/my-role/0aB0aB0aB0aB0aB0aB0aB0aB
lease_duration      768h
lease_renewable     true
accessor            cccccccc-cccc-cccc-cccc-cccccccccccc
consul_namespace    n/a
local               false
partition           n/a
token               bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
```

Here we can see that OpenBao has generated a new Consul ACL token for us.
We can verify this, by reading it in Consul (by it's accessor):

```shell-session
$ consul acl token read -accessor-id cccccccc-cccc-cccc-cccc-cccccccccccc
AccessorID:       cccccccc-cccc-cccc-cccc-cccccccccccc
SecretID:         bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
Description:      Vault worker root 1735726200124356789
Local:            false
Create Time:      2025-01-01 10:10:10.124356789 +0000 UTC
Policies:
   dddddddd-dddd-dddd-dddd-dddddddddddd - readonly
```


> [!WARNING]
> **Expired token rotation:** Once a token's TTL expires, then Consul
> operations will no longer be allowed with it. This requires you to have an
> external process to rotate tokens. At this time, the recommended approach for
> operators is to rotate the tokens manually by creating a new token using the
> `bao read consul/creds/my-role` command. Once the token is synchronized with
> Consul, apply the token to the agents using the Consul API or CLI.

## API

The Consul secrets engine has a full HTTP API. Please see the [Consul secrets
engine API](./api/readme.md) for more details.

[consul]: https://developer.hashicorp.com/consul
[consul-bootstrap-acl]: https://developer.hashicorp.com/consul/docs/secure/acl/bootstrap
[consul-namespaces]: https://developer.hashicorp.com/consul/docs/enterprise/namespaces
[consul-node-identity]: https://developer.hashicorp.com/consul/commands/acl/token/create#node-identity
[consul-partitions]: https://developer.hashicorp.com/consul/docs/enterprise/admin-partitions
[consul-policy]: https://developer.hashicorp.com/consul/docs/secure/acl/policy
[consul-role]: https://developer.hashicorp.com/consul/docs/secure/acl/role
[consul-service-identity]: https://developer.hashicorp.com/consul/commands/acl/token/create#service-identity
