# Google Cloud auth method (API)

This is the API documentation for the OpenBao Google Cloud auth method. To learn
more about the usage and operation, see the
[OpenBao Google Cloud method documentation](index.md).

This documentation assumes the plugin method is mounted at the
`/auth/gcp` path in OpenBao. Since it is possible to enable auth methods
at any location, please update your API calls accordingly.

## Configure

Configures the credentials required for the plugin to perform API calls
to Google Cloud. These credentials will be used to query the status of IAM
entities and get service account or other Google public certificates
to confirm signed JWTs passed in during login.

| Method | Path               |
| :----- | :----------------- |
| `POST` | `/auth/gcp/config` |

### Parameters

- `credentials` `(string: "")` - A JSON string containing the contents of a GCP
  service account credentials file. The service account associated with the credentials
  file must have the following [permissions](index.md#required-gcp-permissions).
  If this value is empty, OpenBao will try to use [Application Default Credentials][gcp-adc]
  from the machine on which the OpenBao server is running.

- `iam_alias` `(string: "role_id")` - Must be either `unique_id` or `role_id`.
  If `unique_id` is specified, the service account's unique ID will be used for
  alias names during login. If `role_id` is specified, the ID of the OpenBao role
  will be used. Only used if role `type` is `iam`.

- `iam_metadata` `(string: "default")` - The metadata to include on the token
  returned by the `login` endpoint. This metadata will be added to both audit logs,
  and on the `iam_alias`. By default, it includes `project_id`, `role`,
  `service_account_id`, and `service_account_email`. To include no metadata,
  set to `""` via the CLI or `[]` via the API. To use only particular fields, select
  the explicit fields. To restore to defaults, send only a field of `default`.
  **Only select fields that will have a low rate of change** for your `iam_alias` because
  each change triggers a storage write and can have a performance impact at scale.
  Only used if role `type` is `iam`.

- `gce_alias` `(string: "role_id")` - Must be either `instance_id` or `role_id`.
  If `instance_id` is specified, the GCE instance ID will be used for alias names
  during login. If `role_id` is specified, the ID of the OpenBao role will be used.
  Only used if role `type` is `gce`.

- `gce_metadata` `(string: "default")` - The metadata to include on the token
  returned by the `login` endpoint. This metadata will be added to both audit logs,
  and on the `gce_alias`. By default, it includes `instance_creation_timestamp`,
  `instance_id`, `instance_name`, `project_id`, `project_number`, `role`,
  `service_account_id`, `service_account_email`, and `zone`. To include no metadata,
  set to `""` via the CLI or `[]` via the API. To use only particular fields, select
  the explicit fields. To restore to defaults, send only a field of `default`.
  **Only select fields that will have a low rate of change** for your `gce_alias` because
  each change triggers a storage write and can have a performance impact at scale.
  Only used if role `type` is `gce`.

- `custom_endpoint` `(map<string|string>: <optional>)` - Specifies overrides to
  [service endpoints](https://cloud.google.com/apis/design/glossary#api_service_endpoint)
  used when making API requests. This allows specific requests made during authentication
  to target alternative service endpoints for use in [Private Google Access](https://cloud.google.com/vpc/docs/configure-private-google-access)
  environments.

  Overrides are set at the subdomain level using the following keys:
  - `api` - Replaces the service endpoint used in API requests to `https://www.googleapis.com`.
  - `iam` - Replaces the service endpoint used in API requests to `https://iam.googleapis.com`.
  - `crm` - Replaces the service endpoint used in API requests to `https://cloudresourcemanager.googleapis.com`.
  - `compute` - Replaces the service endpoint used in API requests to `https://compute.googleapis.com`.

  The endpoint value provided for a given key has the form of `scheme://host:port`.
  The `scheme://` and `:port` portions of the endpoint value are optional.

### Sample payload

```json
{
  "credentials": "{ \"type\": \"service_account\", \"project_id\": \"project-123456\", ...}"
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/auth/gcp/config
```

## Read config

Returns the configuration, if any, including credentials.

| Method | Path               |
| :----- | :----------------- |
| `GET`  | `/auth/gcp/config` |

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    http://127.0.0.1:8200/v1/auth/gcp/config
```

### Sample response

```json
{
  "data": {
    "client_email": "service-account@project-123456.iam.gserviceaccount.com",
    "client_id": "123456789101112131415",
    "private_key_id": "97fd7ba59a96e1f3830296aedb4f50879e4d5382",
    "project_id": "project-123456"
  }
}
```

## Create/Update role

Registers a role in the method. Role types have specific entities
that can perform login operations against this endpoint. Constraints specific
to the role type must be set on the role. These are applied to the authenticated
entities attempting to login.

| Method | Path                   |
| :----- | :--------------------- |
| `POST` | `/auth/gcp/role/:name` |

### Parameters

- `name` `(string: <required>)` - The name of the role.

- `type` `(string: <required>)` - The type of this role. Certain fields
  correspond to specific roles and will be rejected otherwise. Please see below
  for more information.

- `bound_service_accounts` `(array: <required for iam>)` - An array of
  service account emails or IDs that login is restricted to,
  either directly or through an associated instance. If set to
  `*`, all service accounts are allowed (you can bind this further using
  `bound_projects`.)

- `bound_projects` `(array: [])` - An array of GCP project IDs. Only entities
  belonging to this project can authenticate under the role.

- `add_group_aliases` `(bool: false)` - If true, any auth token
  generated under this token will have associated group aliases, namely
  `project-$PROJECT_ID`, `folder-$PROJECT_ID`, and `organization-$ORG_ID`
  for the entities project and all its folder or organization ancestors. This
  requires OpenBao to have IAM permission `resourcemanager.projects.get`.

- `token_ttl` `(integer: 0 or string: "")` - The incremental lifetime for
  generated tokens. This current value of this will be referenced at renewal
  time.
- `token_max_ttl` `(integer: 0 or string: "")` - The maximum lifetime for
  generated tokens. This current value of this will be referenced at renewal
  time.
- `token_policies` `(array: [] or comma-delimited string: "")` - List of
  token policies to encode onto generated tokens. Depending on the auth method, this
  list may be supplemented by user/group/other values.
- `policies` `(array: [] or comma-delimited string: "")` - DEPRECATED: Please
  use the `token_policies` parameter instead. List of token policies to encode
  onto generated tokens. Depending on the auth method, this list may be
  supplemented by user/group/other values.

- `token_bound_cidrs` `(array: [] or comma-delimited string: "")` - List of
  CIDR blocks; if set, specifies blocks of IP addresses which can authenticate
  successfully, and ties the resulting token to these blocks as well.
- `token_explicit_max_ttl` `(integer: 0 or string: "")` - If set, will encode
  an [explicit max
  TTL](https://openbao.org/docs/concepts/tokens/#token-time-to-live-periodic-tokens-and-explicit-max-ttls)
  onto the token. This is a hard cap even if `token_ttl` and `token_max_ttl`
  would otherwise allow a renewal.
- `token_no_default_policy` `(bool: false)` - If set, the `default` policy will
  not be set on generated tokens; otherwise it will be added to the policies set
  in `token_policies`.
- `token_num_uses` `(integer: 0)` - The maximum number of times a generated
  token may be used (within its lifetime); 0 means unlimited.
  If you require the token to have the ability to create child tokens,
  you will need to set this value to 0.
- `token_period` `(integer: 0 or string: "")` - The maximum allowed [period](https://openbao.org/docs/concepts/tokens/#token-time-to-live-periodic-tokens-and-explicit-max-ttls) value when a periodic token is requested from this role.
- `token_type` `(string: "")` - The type of token that should be generated. Can
  be `service`, `batch`, or `default` to use the mount's tuned default (which
  unless changed will be `service` tokens). For token store roles, there are two
  additional possibilities: `default-service` and `default-batch` which specify
  the type to return unless the client requests a different type at generation
  time.

#### `iam`-only parameters

The following parameters are only valid when the role is of type `"iam"`:

- `max_jwt_exp` `(string: "15m")` - The number of seconds past the time of
  authentication that the login param JWT must expire within. For example, if a
  user attempts to login with a token that expires within an hour and this is
  set to 15 minutes, OpenBao will return an error prompting the user to create a
  new signed JWT with a shorter `exp`. The GCE metadata tokens currently do not
  allow the `exp` claim to be customized.

- `allow_gce_inference` `(bool: true)` - A flag to determine if this role should
  allow GCE instances to authenticate by inferring service accounts from the
  GCE identity metadata token.

#### `gce`-only parameters

The following parameters are only valid when the role is of type `"gce"`:

- `bound_zones` `(array: [])`: The list of zones that a GCE instance must belong
  to in order to be authenticated. If `bound_instance_groups` is provided, it is
  assumed to be a zonal group and the group must belong to this zone.

- `bound_regions` `(array: [])`: The list of regions that a GCE instance must
  belong to in order to be authenticated. If `bound_instance_groups` is
  provided, it is assumed to be a regional group and the group must belong to
  this region. If `bound_zones` are provided, this attribute is ignored.

- `bound_instance_groups` `(array: [])`: The instance groups that an authorized
  instance must belong to in order to be authenticated. If specified, either
  `bound_zones` or `bound_regions` must be set too.

- `bound_labels` `(array: [])`: A comma-separated list of GCP labels formatted
  as "key:value" strings that must be set on authorized GCE instances. Because
  GCP labels are not currently ACL'd, we recommend that this be used in
  conjunction with other restrictions.

### Sample payload

Example `iam` role:

```json
{
  "type": "iam",
  "project_id": "project-123456",
  "policies": ["prod"],
  "ttl": "30m",
  "max_ttl": "24h",
  "max_jwt_exp": "5m",
  "bound_service_accounts": ["dev-1@project-123456.iam.gserviceaccount.com"]
}
```

Example `gce` role:

```json
{
  "type": "gce",
  "project_id": "project-123456",
  "policies": ["prod"],
  "bound_zones": ["us-east1-b", "eu-west2-a"],
  "ttl": "30m",
  "max_ttl": "24h",
  "bound_service_accounts": ["dev-1@project-123456.iam.gserviceaccount.com"]
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/auth/gcp/role/my-role
```

## Edit service accounts on IAM role

Edit service accounts for an existing IAM role in the method.
This allows you to add or remove service accounts from the list of
service accounts on the role.

| Method | Path                                    |
| :----- | :-------------------------------------- |
| `POST` | `/auth/gcp/role/:name/service-accounts` |

### Parameters

- `name` `(string: <required>)` - The name of an existing `iam` type role. This
  will return an error if role is not an `iam` type role.

- `add` `(array: [])` - The list of service accounts to add to the role's
  service accounts.

- `remove` `(array: [])` - The list of service accounts to remove from the
  role's service accounts.

### Sample payload

```json
{
  "add": ["dev-1@project-123456.iam.gserviceaccount.com", "123456789"],
  "remove": ["dev-2@project-123456.iam.gserviceaccount.com"]
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/auth/gcp/role/my-role
```

## Edit labels on GCE role

Edit labels for an existing GCE role in the backend. This allows you to add or
remove labels (keys, values, or both) from the list of keys on the role.

| Method | Path                          |
| :----- | :---------------------------- |
| `POST` | `/auth/gcp/role/:name/labels` |

### Parameters

- `name` `(string: <required>)` - The name of an existing `gce` role. This will
  return an error if role is not a `gce` type role.

- `add` `(array: [])` - The list of `key:value` labels to add to the GCE role's
  bound labels.

- `remove` `(array: [])` - The list of label _keys_ to remove from the role's
  bound labels. If any of the specified keys do not exist, no error is returned
  (idempotent).

### Sample payload

```json
{
  "add": ["foo:bar", "env:dev", "key:value"],
  "remove": ["key1", "key2"]
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/auth/gcp/role/my-role
```

## Read role

Returns the previously registered role configuration.

| Method | Path                   |
| :----- | :--------------------- |
| `GET`  | `/auth/gcp/role/:name` |

### Parameters

- `name` `(string: <required>)` - The name of the role to read.

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    http://127.0.0.1:8200/v1/auth/gcp/role/my-role
```

### Sample response

```json
{
  "data": {
    "bound_labels": {
      "env": "dev",
      "foo": "bar",
      "key": "value"
    },
    "bound_service_accounts": ["dev-1@project-123456.iam.gserviceaccount.com"],
    "bound_zones": ["eu-west2-a", "us-east1-b"],
    "gce_alias": "instance_id",
    "max_ttl": 86400,
    "policies": ["prod"],
    "project_id": "project-123456",
    "role_id": "6bbfab2b-ca32-6044-4829-4515728d87b1",
    "type": "gce",
    "ttl": 1800
  }
}
```

## List roles

Lists all the roles that are registered with the plugin.

| Method | Path              |
| :----- | :---------------- |
| `LIST` | `/auth/gcp/roles` |

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request LIST \
    http://127.0.0.1:8200/v1/auth/gcp/roles
```

### Sample response

```json
{
  "data": {
    "keys": ["my-role", "my-other-role"]
  }
}
```

## Delete role

Deletes the previously registered role.

| Method   | Path                   |
| :------- | :--------------------- |
| `DELETE` | `/auth/gcp/role/:role` |

### Parameters

- `role` `(string: <required>)` - The name of the role to delete.

### Sample request

```shell-session
$ curl \
    --header "X-Bao-Token: ..." \
    --request DELETE \
    http://127.0.0.1:8200/v1/auth/gcp/role/my-role
```

## Login

Login to retrieve a OpenBao token. This endpoint takes a signed JSON Web Token
(JWT) and a role name for some entity. It verifies the JWT signature with Google
Cloud to authenticate that entity and then authorizes the entity for the given
role.

| Method | Path              |
| :----- | :---------------- |
| `POST` | `/auth/gcp/login` |

### Sample payload

- `role` `(string: <required>)` - The name of the role against which the login
  is being attempted.

- `jwt` `(string: <required>)` - A Signed [JSON Web Token][jwt].

  - For `iam` type roles, this is a JWT signed with the
    [`signJwt` method][signjwt-method] or a self-signed JWT.

  - For `gce` type roles, this is an [identity metadata token][instance-token].

### Sample payload

```json
{
  "role": "my-role",
  "jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Sample request

```shell-session
$ curl \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/auth/gcp/login
```

### Sample response

```json
{
  "auth": {
    "client_token": "f33f8c72-924e-11f8-cb43-ac59d697597c",
    "accessor": "0e9e354a-520f-df04-6867-ee81cae3d42d",
    "policies": ["default", "dev", "prod"],
    "metadata": {
      "project_id": "my-project",
      "role": "my-role",
      "service_account_email": "dev1@project-123456.iam.gserviceaccount.com",
      "service_account_id": "111111111111111111111"
    },
    "lease_duration": 2764800,
    "renewable": true
  }
}
```

[gcp-adc]: https://developers.google.com/identity/protocols/application-default-credentials
[jwt]: https://tools.ietf.org/html/rfc7519
[signjwt-method]: https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/signJwt
[instance-token]: https://cloud.google.com/compute/docs/instances/verifying-instance-identity#request_signature
