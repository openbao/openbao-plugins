# Azure secrets engine (API)

This is the API documentation for the OpenBao Azure
secrets engine. For general information about the usage and operation of
the Azure secrets engine, please see the main [Azure secrets documentation][docs].

This documentation assumes the Azure secrets engine is enabled at the `/azure` path
in OpenBao. Since it is possible to mount secrets engines at any path, please
update your API calls accordingly.

## Configure access

Configures the credentials required for the plugin to perform API calls
to Azure. These credentials will be used to query roles and create/delete
service principals. Environment variables will override any parameters set in the config.

| Method | Path            |
| :----- | :-------------- |
| `POST` | `/azure/config` |

- `subscription_id` (`string: <required>`) - The subscription id for the Azure Active Directory.
  This value can also be provided with the AZURE_SUBSCRIPTION_ID environment variable.
- `tenant_id` (`string: <required>`) - The tenant id for the Azure Active Directory.
  This value can also be provided with the AZURE_TENANT_ID environment variable.
- `client_id` (`string:""`) - The OAuth2 client id to connect to Azure. This value can also be provided
  with the AZURE_CLIENT_ID environment variable. See [authentication](index.md#authentication) for more details.
- `client_secret` (`string:""`) - The OAuth2 client secret to connect to Azure. This value can also be
  provided with the AZURE_CLIENT_SECRET environment variable. See [authentication](index.md#authentication) for more details.
- `environment` (`string:""`) - The Azure environment. This value can also be provided with the AZURE_ENVIRONMENT
  environment variable. If not specified, OpenBao will use Azure Public Cloud.
- `password_policy` `(string: "")` - Specifies a [password policy](https://openbao.org/docs/concepts/password-policies/) to
  use when creating dynamic credentials. Defaults to generating an alphanumeric password if not set.
- `root_password_ttl` `(string: 182d)` - Specifies how long the root password is valid for in Azure when
  rotate-root generates a new client secret. Uses [duration format strings](https://openbao.org/docs/concepts/duration-format/).

### Sample payload

```json
{
  "subscription_id": "94ca80...",
  "tenant_id": "d0ac7e...",
  "client_id": "e607c4...",
  "client_secret": "9a6346...",
  "environment": "AzureGermanCloud",
  "password_policy": "azure_policy",
  "root_password_ttl": "48d"
}
```

### Sample request

```shell-session
$ bao write azure/config \
    subscription_id="94ca80..." \
    tenant_id="d0ac7e...",
    client_id="e607c4...",
    client_secret="9a6346...",
    environment="AzureGermanCloud",
    password_policy="azure_policy"
```

## Read config

Return the stored configuration, omitting `client_secret`.

| Method | Path            |
| :----- | :-------------- |
| `GET`  | `/azure/config` |

### Sample request

```shell-session
$ bao read azure/config
```

### Sample response

```json
{
  "data": {
    "subscription_id": "94ca80...",
    "tenant_id": "d0ac7e...",
    "client_id": "e607c4...",
    "environment": "AzureGermanCloud"
  },
  ...
}
```

## Delete config

Deletes the stored Azure configuration and credentials.

| Method   | Path            |
| :------- | :-------------- |
| `DELETE` | `/azure/config` |

### Sample request

```shell-session
$ bao delete azure/config
```

## Rotate root

This endpoint generates a new client secret for the root account defined in the config. The
value generated will only be known by OpenBao.

~> Due to the eventual consistency of Microsoft Azure client secret APIs, the plugin
may briefly stop authenticating to Azure as the password propagates through their
datacenters.

| Method | Path                 |
| :----- | :------------------- |
| `POST` | `/azure/rotate-root` |

### Parameters

There are no parameters to this operation.

### Sample request

```shell-session
$ curl \
  --header "X-Bao-Token: ..." \
  --request POST \
  http://127.0.0.1:8200/v1/azure/rotate-root
```

## Create/Update role

Create or update a OpenBao role. Either `application_object_id` or
`azure_roles` must be provided, and these resources must exist for this
call to succeed. See the Azure secrets [roles docs][roles] for more
information about roles.

| Method | Path                 |
| :----- | :------------------- |
| `POST` | `/azure/roles/:name` |

### Parameters

- `azure_roles` (`string: ""`) - List of Azure roles to be assigned to the generated service
  principal. The array must be in JSON format, properly escaped as a string. See [roles docs][roles]
  for details on role definition.
- `azure_groups` (`string: ""`) - List of Azure groups that the generated service principal will be
  assigned to. The array must be in JSON format, properly escaped as a string. See [groups docs][groups]
  for more details.
- `application_object_id` (`string: ""`) - Application Object ID for an existing service principal that will
  be used instead of creating dynamic service principals. If present, `azure_roles` will be ignored. See
  [roles docs][roles] for details on role definition.
- `persist_app` (`bool: "false"`) – If set to true, persists the created service principal and application for the lifetime of the role.
 Useful for when the Service Principal needs to maintain ownership of objects it creates
- `ttl` (`string: ""`) – Specifies the default TTL for service principals generated using this role.
  Accepts time suffixed strings ("1h") or an integer number of seconds. Defaults to the system/engine default TTL time.
- `max_ttl` (`string: ""`) – Specifies the maximum TTL for service principals generated using this role. Accepts time
  suffixed strings ("1h") or an integer number of seconds. Defaults to the system/engine max TTL time.
- `permanently_delete` (`bool: false`) - Specifies whether to permanently delete Applications and Service Principals that are dynamically
  created by OpenBao. If `application_object_id` is present, `permanently_delete` must be `false`.

### Sample payload

```json
{
  "azure_roles": "[
    {
      \"role_name\": \"Contributor\",
      \"scope\":  \"/subscriptions/<uuid>/resourceGroups/Website\"
    },
    {
      \"role_id\": \"/subscriptions/<uuid>/providers/Microsoft.Authorization/roleDefinitions/<uuid>\",
      \"scope\":  \"/subscriptions/<uuid>\"
    }
  ]",
  "ttl": 3600,
  "max_ttl": "24h"
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request POST \
    --data @payload.json \
    https://127.0.0.1:8200/v1/azure/roles/my-role
```

## List roles

Lists all of the roles that are registered with the plugin.

| Method | Path           |
| :----- | :------------- |
| `LIST` | `/azure/roles` |

### Sample request

```shell-session
$ bao list azure/roles
```

### Sample response

```json
{
  "data": {
    "keys": ["my-role-one", "my-role-two"]
  }
}
```

## Generate credentials

This endpoint generates a new service principal based on the named role.

| Method | Path                 |
| :----- | :------------------- |
| `GET`  | `/azure/creds/:name` |

### Parameters

- `name` (`string: <required>`) - Specifies the name of the role to create credentials against.

### Sample request

```shell-session
$ bao read azure/creds/my-role
```

### Sample response

```json
{
  "data": {
    "client_id": "408bf248-dd4e-4be5-919a-7f6207a307ab",
    "client_secret": "9PfdaDP9qcf98ggw8WSttfVreFcN4q9c4m4x",
    ...
  }
}
```

## Revoking/Renewing secrets

See docs on how to [renew](https://openbao.org/api-docs/system/leases/#renew-lease) and [revoke](https://openbao.org/api-docs/system/leases/#revoke-lease) leases.

[docs]: index.md
[roles]: index.md#roles
[groups]: index.md#azure-groups
