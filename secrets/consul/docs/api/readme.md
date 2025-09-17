# Consul secrets engine (API)

This is the API documentation for the OpenBao Consul secrets engine. For general
information about the usage and operation of the Consul secrets engine, please
see the [OpenBao Consul documentation](../readme.md).

This documentation assumes the Consul secrets engine is mounted at the `/consul`
path in OpenBao. Since it is possible to mount secrets engines at any location,
please update your API calls accordingly.

## Configure access

This endpoint configures the access information for Consul. This access
information is used so that OpenBao can communicate with Consul and generate
Consul tokens.

| Method | Path                    |
| :----- | :---------------------- |
| `POST` | `/consul/config/access` |

### Parameters

- `address` `(string: <required>)` – Specifies the address of the Consul
  instance, provided as `"host:port"` like `"127.0.0.1:8500"`.

- `scheme` `(string: "http")` – Specifies the URL scheme to use ("http" or
  "https").

- `token` `(string: "")` – Specifies the Consul ACL token to use. If this is not
  provided, the plugin will try to bootstrap the ACL system of the Consul
  cluster automatically.

- `ca_cert` `(string: "")` - CA certificate to use when verifying Consul server
  certificate, must be x509 PEM encoded.

- `client_cert` `(string: "")` - Client certificate used for Consul's TLS
  communication, must be x509 PEM encoded and if this is set you need to also
  set `client_key`.

- `client_key` `(string: "")` - Client key used for Consul's TLS communication, must
  be x509 PEM encoded and if this is set you need to also set `client_cert`.

### Sample payload

```json
{
  "address": "consul.example.com:8500",
  "scheme": "https",
  "token": "ffffffff-ffff-ffff-ffff-ffffffffffff"
}
```

### Sample request

```shell-session
$ curl \
    --request POST \
    --header "X-Vault-Token: ..." \
    --data @payload.json \
    http://127.0.0.1:8200/v1/consul/config/access
```

## Read access configuration

This endpoint queries for information about the Consul connection.

| Method | Path                    |
| :----- | :---------------------- |
| `GET`  | `/consul/config/access` |

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/consul/config/access
```

### Sample response

```json
{
  "data": {
    "address": "consul.example.com:8500",
    "scheme": "https"
  }
}
```

## Create/Update role

This endpoint creates or updates the Consul role definition in OpenBao. If the
role does not exist, it will be created. If the role already exists, it will
receive updated attributes. At least one of `consul_roles`, `consul_policies`,
`node_identities`, or `service_identities` is required.

| Method | Path                  |
| :----- | :-------------------- |
| `POST` | `/consul/roles/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to
  create/update. This is part of the request URL.

- `consul_policies` `(array: [])` – The list of Consul policies to assign to the
  generated token.

- `consul_roles` `(array: [])` – The list of Consul roles to assign to the
  generated token.

- `service_identities` `(array: [])` – The list of service identities to assign
  to the generated token.

- `node_identities` `(array: [])` - The list of node identities to assign to the
  generated token. Available in Consul 1.8 or above.

- `consul_namespace` `(string: "default")` - Specifies the Consul namespace in
  which the token is generated. Available in Consul 1.7 and above. Requires
  Consul Enterprise.

- `partition` `(string: "default")` - Specifies the Consul admin partition in
  which the token is generated. Available in Consul 1.11 and above. Requires
  Consul Enterprise.

- `local` `(bool: false)` - Indicates that the token should not be replicated
  globally and instead be local to the current datacenter.

- `ttl` `(duration: 1h)` - Specifies the TTL of tokens generated for this role.
  If not provided, the default OpenBao TTL is used.

- `max_ttl` `(duration: 24h)` - Specifies the max TTL of tokens generated for
  this role. If not provided, the default OpenBao max TTL is used.

### Sample payload

To create a client token with policies "policy1" and "policy2" defined in
Consul:

```json
{
  "consul_policies": "policy1,policy2",
  "ttl": "10m",
  "max_ttl": "1h"
}
```

### Sample request

```shell-session
$ curl \
    --request POST \
    --header "X-Vault-Token: ..." \
    --data @payload.json \
    http://127.0.0.1:8200/v1/consul/roles/example-role
```

## Read role

This endpoint queries for information about a Consul role with the given name.
If no role exists with that name, a 404 is returned.

| Method | Path                  |
| :----- | :-------------------- |
| `GET`  | `/consul/roles/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to query. This
  is part of the request URL.

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/consul/roles/example-role
```

### Sample response

```json
{
  "data": {
    "consul_namespace": "",
    "consul_policies": ["policy1", "policy2"],
    "local": false,
    "max_ttl": 3600,
    "partition": "",
    "ttl": 600
  }
}
```

## List roles

This endpoint lists all existing roles in the secrets engine.

| Method | Path                      |
| :----- | :------------------------ |
| `LIST` | `/consul/roles`           |
| `GET`  | `/consul/roles?list=true` |

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST \
    http://127.0.0.1:8200/v1/consul/roles
```

### Sample response

```json
{
  "data": {
    "keys": ["example-role"]
  }
}
```

## Delete role

This endpoint deletes a Consul role with the given name. Even if the role does
not exist, this endpoint will still return a successful response.

| Method   | Path                  |
| :------- | :-------------------- |
| `DELETE` | `/consul/roles/:name` |

### Parameters

- `name` `(string: <required>)` - Specifies the name of the role to delete. This
  is part of the request URL.

### Sample request

```shell-session
$ curl \
    --request DELETE \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/consul/roles/example-role
```

## Generate credential

This endpoint generates a dynamic Consul token based on the given role
definition.

| Method | Path                  |
| :----- | :-------------------- |
| `GET`  | `/consul/creds/:name` |

### Parameters

- `name` `(string: <required>)` - Specifies the name of an existing role against
  which to create this Consul credential. This is part of the request URL.

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/consul/creds/example-role
```

### Sample response

```json
{
  "data": {
    "accessor": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    "consul_namespace": "",
    "local": false,
    "partition": "",
    "token": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
  }
}
```
