---
sidebar_label: API
description: >-
  This is the API documentation for the OpenBao Home secrets engine.
---

# Home secrets engine (API)

This is the API documentation for the OpenBao Home secrets engine. For general
information about the usage and operation of the Home secrets engine, please see
the [OpenBao Home documentation](/docs/secrets/home).

This documentation assumes the Home secrets engine is enabled at the `/home`
path in OpenBao. Since it is possible to enable secrets engines at any location,
please update your API calls accordingly.

**Important:** All endpoints require the caller's token to have a resolved
Identity entity. Requests from tokens with no entity (such as the root token)
will receive a `403` response.

## Read secret

This endpoint retrieves the secret at the specified location.

| Method | Path |
|--------|------|
| `GET`  | `/home/:path` |

### Parameters

- `path` `(string: <required>)` – Specifies the path of the secret to read.
  This is specified as part of the URL.

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/home/my-secret
```

### Sample response

```json
{
  "auth": null,
  "data": {
    "password": "hunter2"
  },
  "lease_duration": 0,
  "lease_id": "",
  "renewable": false
}
```

## List secrets

This endpoint returns a list of secret entries at the specified location.
Folders are suffixed with `/`. The input must be a folder; list on a file will
not return a value.

| Method | Path |
|--------|------|
| `LIST` | `/home/:path` |

### Parameters

- `path` `(string: "")` – Specifies the path of the secrets to list. This is
  specified as part of the URL.

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST \
    http://127.0.0.1:8200/v1/home/
```

### Sample response

The example below shows output for a query path of `home/` when there are
secrets at `home/foo` and `home/foo/bar`; note the difference in the two
entries.

```json
{
  "auth": null,
  "data": {
    "keys": ["foo", "foo/"]
  },
  "lease_duration": 0,
  "lease_id": "",
  "renewable": false
}
```

## Create/Update secret

This endpoint stores a secret at the specified location. If a secret already
exists at the given path it is overwritten.

| Method | Path |
|--------|------|
| `POST` | `/home/:path` |

### Parameters

- `path` `(string: <required>)` – Specifies the path of the secret to
  create/update. This is specified as part of the URL.

- `:key` `(string: "")` – Specifies a key in the request payload, paired with
  an associated value, to be held at the given location. Multiple key/value
  pairs can be specified, and all will be returned on a read operation.

### Sample payload

```json
{
  "password": "hunter2",
  "ttl": "1h"
}
```

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/home/my-secret
```

### Sample response

A successful write returns `204 No Content` with no body.

## Delete secret

This endpoint deletes the secret at the specified location.

| Method   | Path |
|----------|------|
| `DELETE` | `/home/:path` |

### Parameters

- `path` `(string: <required>)` – Specifies the path of the secret to delete.
  This is specified as part of the URL.

### Sample request

```shell-session
$ curl \
    --header "X-Vault-Token: ..." \
    --request DELETE \
    http://127.0.0.1:8200/v1/home/my-secret
```

### Sample response

A successful delete returns `204 No Content` with no body.

## Error responses

### No Identity entity

If the caller's token does not have a resolved Identity entity (e.g. the root
token), all endpoints will return:

```json
{
  "errors": [
    "home: caller has no Identity entity; the home engine requires an entity to derive a stable storage path. Root tokens and tokens without an associated entity cannot use this engine"
  ]
}
```

**Status:** `403 Forbidden`

### Path traversal

If the requested path contains `..` components, all endpoints will return:

```json
{
  "errors": [
    "home: path \"../etc/shadow\" contains illegal '..' component"
  ]
}
```

**Status:** `400 Bad Request`
