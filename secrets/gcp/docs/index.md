# Google Cloud secrets engine

The Google Cloud OpenBao secrets engine dynamically generates Google Cloud service
account keys and OAuth tokens based on IAM policies. This enables users to gain
access to Google Cloud resources without needing to create or manage a dedicated
service account.

The benefits of using this secrets engine to manage Google Cloud IAM service accounts are:

- **Automatic cleanup of GCP IAM service account keys** - each Service Account
  key is associated with a OpenBao lease. When the lease expires (either during
  normal revocation or through early revocation), the service account key is
  automatically revoked.

- **Quick, short-term access** - users do not need to create new GCP Service
  Accounts for short-term or one-off access (such as batch jobs or quick
  introspection).

- **Multi-cloud and hybrid cloud applications** - users authenticate to OpenBao
  using a central identity service (such as LDAP) and generate GCP credentials
  without the need to create or manage a new Service Account for that user.

## Setup

Most secrets engines must be configured in advance before they can perform their
functions. These steps are usually completed by an operator or configuration
management tool.

1.  Enable the Google Cloud secrets engine:

    ```shell-session
    $ bao secrets enable gcp
    Success! Enabled the gcp secrets engine at: gcp/
    ```

    By default, the secrets engine will mount at the name of the engine. To
    enable the secrets engine at a different path, use the `-path` argument.

1.  Configure the secrets engine with account credentials, or leave blank or unwritten
    to use Application Default Credentials.

    ```shell-session
    $ bao write gcp/config credentials=@my-credentials.json
    Success! Data written to: gcp/config
    ```

    If you are running OpenBao from inside [Google Compute Engine][gce] or [Google
    Kubernetes Engine][gke], the instance or pod service account can be used in
    place of specifying the credentials JSON file.
    For more information on authentication, see the [authentication section](#authentication) below.

1. Configure rolesets or static accounts. See the relevant sections below.

## Rolesets

A roleset consists of a OpenBao managed GCP Service account along with a set of IAM bindings
defined for that service account. The name of the service account is generated based on the time
of creation or update. You should not depend on the name of the service account being
fixed and should manage all IAM bindings for the service account through the `bindings` parameter
when creating or updating the roleset.

For more information on the differences between rolesets and static accounts, see the
[things to note](#things-to-note) section below.

### Roleset policy considerations

Starting with OpenBao 1.8.0, existing permissive policies containing globs 
for the GCP Secrets Engine may grant additional privileges due to the introduction 
of `/gcp/roleset/:roleset/token` and `/gcp/roleset/:roleset/key` endpoints.

The following policy grants a user the ability to read all rolesets, but would 
also allow them to generate tokens and keys. This type of policy is not recommended:

```hcl
# DO NOT USE
path "/gcp/roleset/*" {
    capabilities = ["read"]
}
```

The following example demonstrates how a wildcard can instead be used in a roleset policy to 
adhere to the principle of least privilege:

```hcl
path "/gcp/roleset/+" {
    capabilities = ["read"]
}
```

For more more information on policy syntax, see the 
[policy documentation](https://openbao.org/docs/concepts/policies/#policy-syntax).

### Examples

To configure a roleset that generates OAuth2 access tokens (preferred):

```shell-session
$ bao write gcp/roleset/my-token-roleset \
    project="my-project-id" \
    secret_type="access_token"  \
    token_scopes="https://www.googleapis.com/auth/cloud-platform" \
    bindings=-<<EOF
      resource "//cloudresourcemanager.googleapis.com/projects/my-project-id" {
        roles = ["roles/viewer"]
      }
    EOF
```

To configure a roleset that generates GCP Service Account keys:

```shell-session
$ bao write gcp/roleset/my-key-roleset \
    project="my-project" \
    secret_type="service_account_key"  \
    bindings=-<<EOF
      resource "//cloudresourcemanager.googleapis.com/projects/my-project" {
        roles = ["roles/viewer"]
      }
    EOF
```

Alternatively, provide a file for the `bindings` argument like so:

```shell-session
$ bao write gcp/roleset/my-roleset
    bindings=@mybindings.hcl
    ...
```

For more information on role bindings and sample role bindings, please see
the [bindings](#bindings) section below.

For more information on the differences between OAuth2 access tokens and
Service Account keys, see the [things to note](#things-to-note) section
below.

For more information on creating and managing rolesets, see the
[GCP secrets engine API docs][api] docs.

## Static accounts

Static accounts are GCP service accounts that are created outside of OpenBao and then provided to
OpenBao to generate access tokens or keys. You can also use OpenBao to optionally manage IAM bindings
for the service account.

For more information on the differences between rolesets and static accounts, see the
[things to note](#things-to-note) section below.

### Examples

Before configuring a static account, you need to create a
[Google Cloud Service Account][service-accounts]. Take note of the email address of the service
account you have created. Service account emails are of the format
`<service-account-id>@<project-id>.iam.gserviceaccount.com`.

To configure a static account that generates OAuth2 access tokens (preferred):

```shell-session
$ bao write gcp/static-account/my-token-account \
    service_account_email="account@my-project.iam.gserviceaccount.com" \
    secret_type="access_token"  \
    token_scopes="https://www.googleapis.com/auth/cloud-platform" \
    bindings=-<<EOF
      resource "//cloudresourcemanager.googleapis.com/projects/my-project" {
        roles = ["roles/viewer"]
      }
    EOF
```

To configure a static account that generates GCP Service Account keys:

```shell-session
$ bao write gcp/static-account/my-key-account \
    service_account_email="account@my-project.iam.gserviceaccount.com" \
    secret_type="service_account_key"  \
    bindings=-<<EOF
      resource "//cloudresourcemanager.googleapis.com/projects/my-project" {
        roles = ["roles/viewer"]
      }
    EOF
```

Alternatively, provide a file for the `bindings` argument like so:

```shell-session
$ bao write gcp/static-account/my-account
    bindings=@mybindings.hcl
    ...
```

For more information on role bindings and sample role bindings, please see
the [bindings](#bindings) section below.

For more information on the differences between OAuth2 access tokens and
Service Account keys, see the [things to note](#things-to-note) section
below.

For more information on creating and managing static accounts, see the
[GCP secrets engine API docs][api] docs.

## Impersonated accounts

Impersonated accounts are a way to generate an OAuth2 [access token](#access-tokens) that is granted
the permissions and accesses of another given service account. These access
tokens do not have the same 10-key limit as service account keys do, yet they
retain their short-lived nature. By default, their TTL in GCP is 1 hour, but
this may be configured to be up to 12 hours as explained in Google's 
[short-lived credentials documentation](https://cloud.google.com/iam/docs/create-short-lived-credentials-delegated#sa-credentials-oauth).

For more information regarding service account impersonation in GCP, consider starting
with their documentation [available here](https://cloud.google.com/iam/docs/impersonating-service-accounts).

### Examples

To configure a OpenBao role that impersonates the administrator on the Google
Cloud project with the cloud platform and compute scopes:

```shell-session
$ bao write gcp/impersonated-account/my-token-impersonate \
    service_account_email="projectAdmin@my-project.iam.gserviceaccount.com" \
    token_scopes="https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/compute" \
    ttl="6h"
```

## Usage

After the secrets engine is configured and a user/machine has a OpenBao token with
the proper permission, it can generate credentials. Depending on how the OpenBao role
was configured, you can generate OAuth2 tokens or service account keys.

### Access tokens

To generate OAuth2 [access tokens](https://cloud.google.com/docs/authentication/token-types#access),
read from the [`gcp/.../token`](api.md#generate-secret-iam-service-account-creds-oauth2-access-token)
API. If using a roleset or static account, it must have been created with a
[`secret_type`](api.md#secret_type) of `access_token`. Impersonated accounts will
generate OAuth2 tokens by default.

**Roleset:**
```shell-session
$ bao read gcp/roleset/my-token-roleset/token

Key                Value
---                -----
expires_at_seconds    1537402548
token                 ya29.c.ElodBmNPwHUNY5gcBpnXcE4ywG4w1k...
token_ttl             3599
```

**Static account:**
```shell-session
$ bao read gcp/static-account/my-token-account/token

Key                Value
---                -----
expires_at_seconds    1672231587
token                 ya29.c.b0Aa9VdykAdYoW9S1ImtPZykF_oTi9...
token_ttl             3599
```

**Impersonated account:**
```shell-session
$ bao read gcp/impersonated-account/my-token-impersonate/token

Key                Value
---                -----
expires_at_seconds    1671667844
token                 ya29.c.b0AT7lpjBRmO7ghBEyMV18evd016hq...
token_ttl             59m59s
```

This endpoint generates a non-renewable, non-revocable static OAuth2 access token
with a max lifetime of one hour, where `token_ttl` is given in seconds and the
`expires_at_seconds` is the expiry time for the token, given as a Unix timestamp.
The `token` value then can be used as a HTTP Authorization Bearer token in requests
to GCP APIs:

```shell-session
$ curl -H "Authorization: Bearer ya29.c.ElodBmNPwHUNY5gcBpnXcE4ywG4w1k..."
```

### Service account keys

To generate service account keys, read from `gcp/.../key`. OpenBao returns the service
account key data as a base64-encoded string in the `private_key_data` field. This can
be read by decoding it using `base64 --decode "ewogICJ0e..."` or another base64 tool of
your choice. The roleset or static account must have been created as type `service_account_key`:

```shell-session
$ bao read gcp/roleset/my-key-roleset/key

Key                 Value
---                 -----
lease_id            gcp/key/my-key-roleset/ce563a99-5e55-389b...
lease_duration      30m
lease_renewable     true
key_algorithm       KEY_ALG_RSA_2048
key_type            TYPE_GOOGLE_CREDENTIALS_FILE
private_key_data    ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsC...
```

This endpoint generates a new [GCP IAM service account key][iam-keys] associated
with the role's Service Account. When the lease expires (or is revoked
early), the Service Account key will be deleted.

**There is a default limit of 10 keys per Service Account.** For more
information on this limit and recommended mitigation, please see the [things to
note](#things-to-note) section below.

## Bindings

Roleset or static account bindings define a list of resources and the associated IAM roles on that
resource. Bindings are used as the `binding` argument when creating or
updating a roleset or static account and are specified in the following format using HCL:

```hcl
resource NAME {
  roles = [ROLE, [ROLE...]]
}
```

For example:

```hcl
resource "buckets/my-bucket" {
  roles = [
    "roles/storage.objectAdmin",
    "roles/storage.legacyBucketReader",
  ]
}

# At instance level, using self-link
resource "https://www.googleapis.com/compute/v1/projects/my-project/zone/my-zone/instances/my-instance" {
  roles = [
    "roles/compute.instanceAdmin.v1"
  ]
}

# At project level
resource "//cloudresourcemanager.googleapis.com/projects/my-project" {
  roles = [
    "roles/compute.instanceAdmin.v1",
    "roles/iam.serviceAccountUser",  # required if managing instances that run as service accounts
  ]
}

# At folder level
resource "//cloudresourcemanager.googleapis.com/folders/123456" {
  roles = [
    "roles/compute.viewer",
    "roles/deploymentmanager.viewer",
  ]
}

```

The top-level `resource` block defines the resource or resource path for which
IAM policy information will be bound. The resource path may be specified in a
few different formats:

- **Project-level self-link** - a URI with scheme and host, generally
  corresponding to the `self_link` attribute of a resource in GCP. This must
  include the resource nested in the parent project.

  ```text
  # compute alpha zone
  https://www.googleapis.com/compute/alpha/projects/my-project/zones/us-central1-c
  ```

- **Full resource name** - a schema-less URI consisting of a DNS-compatible API
  service name and resource path. See the [full resource name API
  documentation][resource-name-full] for more information.

  ```text
  # Compute snapshot
  //compute.googleapis.com/project/my-project/snapshots/my-compute-snapshot

  # Pubsub snapshot
  //pubsub.googleapis.com/project/my-project/snapshots/my-pubsub-snapshot

  # BigQuery dataset
  //bigquery.googleapis.com/projects/my-project/datasets/mydataset

  # Resource manager
  //cloudresourcemanager.googleapis.com/projects/my-project"
  ```

- **Relative resource name** - A path-noscheme URI path, usually as accepted by
  the API. Use this if the version or service are apparent from the resource
  type. Please see the [relative resource name API
  documentation][resource-name-relative] for more information.

  ```text
  # Storage bucket objects
  buckets/my-bucket
  buckets/my-bucket/objects/my-object

  # PubSub topics
  projects/my-project/topics/my-pubsub-topic
  ```

The nested `roles` attribute is an array of strings names of [GCP IAM
roles][iam-roles]. The roles may be specified in the following formats:

- **Global role name** - these are global roles built into Google Cloud. For the
  full list of available roles, please see the [list of predefined GCP
  roles][predefined-roles].

  ```text
  roles/viewer
  roles/bigquery.user
  roles/billing.admin
  ```

- **Organization-level custom role** - these are roles that are created at the
  organization level by organization owners.

  ```text
  organizations/my-organization/roles/my-custom-role
  ```

  For more information, please see the documentation on [GCP custom
  roles][custom-roles].

- **Project-level custom role** - these are roles that are created at a
  per-project level by project owners.

  ```text
  projects/my-project/roles/my-custom-role
  ```

  For more information, please see the documentation on [GCP custom
  roles][custom-roles].

## Authentication

The Google Cloud OpenBao secrets backend uses the official Google Cloud Golang
SDK. This means it supports the common ways of [providing credentials to Google
Cloud][cloud-creds]. In addition to specifying `credentials` directly via OpenBao
configuration, you can also get configuration from the following values **on the
OpenBao server**:

1. The environment variables `GOOGLE_APPLICATION_CREDENTIALS`. This is specified
   as the **path** to a Google Cloud credentials file, typically for a service
   account. If this environment variable is present, the resulting credentials are
   used. If the credentials are invalid, an error is returned.

1. Default instance credentials. When no environment variable is present, the
   default service account credentials are used. This is useful when running OpenBao
   on [Google Compute Engine][gce] or [Google Kubernetes Engine][gke]

For more information on service accounts, please see the [Google Cloud Service
Accounts documentation][service-accounts].

To use this secrets engine, the service account must have the following
minimum scope(s):

```text
https://www.googleapis.com/auth/cloud-platform
```

### Required permissions

The credentials given to OpenBao must have the following permissions when using rolesets at the
project level:

```text
# Service account + key admin
iam.serviceAccounts.create
iam.serviceAccounts.delete
iam.serviceAccounts.get
iam.serviceAccounts.list
iam.serviceAccounts.update
iam.serviceAccountKeys.create
iam.serviceAccountKeys.delete
iam.serviceAccountKeys.get
iam.serviceAccountKeys.list
```

When using static accounts or impersonated accounts, OpenBao must have the following permissions
at the service account level:

```text
# For `access_token` secrets and impersonated accounts
iam.serviceAccounts.getAccessToken

# For `service_account_keys` secrets
iam.serviceAccountKeys.create
iam.serviceAccountKeys.delete
iam.serviceAccountKeys.get
iam.serviceAccountKeys.list
```

When using rolesets or static accounts with bindings, OpenBao must have the following permissions:

```text
# IAM policy changes
<service>.<resource>.getIamPolicy
<service>.<resource>.setIamPolicy
```

where `<service>` and `<resource>` correspond to permissions which will be
granted, for example:

```text
# Projects
resourcemanager.projects.getIamPolicy
resourcemanager.projects.setIamPolicy

# All compute
compute.*.getIamPolicy
compute.*.setIamPolicy

# BigQuery datasets
bigquery.datasets.get
bigquery.datasets.update
```

You can either:

- Create a [custom role][custom-roles] using these permissions, and assign this
  role at a project-level

- Assign the set of roles required to get resource-specific
  `getIamPolicy/setIamPolicy` permissions. At a minimum you will need to assign
  `roles/iam.serviceAccountAdmin` and `roles/iam.serviceAccountKeyAdmin` so
  OpenBao can manage service accounts and keys.

- Notice that BigQuery requires different permissions than other resource. This is
  because BigQuery currently uses legacy ACL instead of traditional IAM permissions.
  This means to update access on the dataset, OpenBao must be able to update the dataset's
  metadata.

### Root credential rotation

If the mount is configured with credentials directly, the credential's key may be
rotated to a OpenBao-generated value that is not accessible by the operator. For more
details on this operation, please see the
[Root Credential Rotation](api.md#rotate-root-credentials) API docs.

## Things to note

### Rolesets vs. static accounts

Advantages of rolesets:

- Service accounts and IAM bindings are fully managed by OpenBao

Disadvantages of rolesets:

- Cannot easily decouple IAM bindings from the ones managed in OpenBao
- OpenBao requires permissions to manage IAM bindings and service accounts

Advantages of static accounts:

- Can manage IAM bindings independently from the ones managed in OpenBao
- OpenBao does not require permissions to IAM bindings and service accounts and only permissions
  related to the keys of the service account

Disadvantages of static accounts:

- Self management of service accounts is necessary.

### Access tokens vs. service account keys

Advantages of `access_tokens`:

- Can generate infinite number of tokens per roleset

Disadvantages of `access_tokens`:

- Cannot be used with some client libraries or tools
- Have a static life-time of 1 hr that cannot be modified, revoked, or extended.

Advantages of `service_account_keys`:

- Controllable life-time through OpenBao, allowing for longer access
- Can be used by all normal GCP tooling

Disadvantages of `service_account_keys`:

- Infinite lifetime in GCP (i.e. if they are not managed properly, leaked keys can live forever)
- Limited to 10 per roleset/service account.

When generating OAuth access tokens, OpenBao will still
generate a dedicated service account and key. This private key is stored in OpenBao
and is never accessible to other users, and the underlying key can
be rotated. See the [GCP API documentation][api] for more information on
rotation.

### Service accounts are tied to rolesets

Service Accounts are created when the roleset is created (or updated) rather
than each time a secret is generated. This may be different from how other
secrets engines behave, but it is for good reasons:

- IAM Service Account creation and permission propagation can take up to 60
  seconds to complete. By creating the Service Account in advance, we speed up
  the timeliness of future operations and reduce the flakiness of automated
  workflows.

- Each GCP project has a limit on the number of IAM Service Accounts. You can
  [request additional quota][quotas]. The quota increase is processed by humans,
  so it is best to request this additional quota in advance. This limit is
  currently 100, **including system-managed Service Accounts**. If Service
  Accounts were created per secret, this quota limit would reduce the number of
  secrets that can be generated.

### Service account keys quota limits

GCP IAM has a hard limit (currently 10) on the number of Service Account keys.
Attempts to generate more keys will result in an error. If you find yourself
running into this limit, consider the following:

- Have shorter TTLs or revoke access earlier. If you are not using past Service
  Account keys, consider rotating and freeing quota earlier.

- Create additional rolesets which share the same set of permissions. Additional
  rolesets can be created with the same set of permissions. This will create a
  new service account and increases the number of keys you can create.

- Where possible, use OAuth2 access tokens instead of Service Account keys.

### Resources in IAM bindings must exist at roleset or static account creation

Because the bindings for the Service Account are set during roleset/static account creation,
resources that do not exist will fail the `getIamPolicy` API call.

### Roleset creation may partially fail

Every Service Account creation, key creation, and IAM policy change is a GCP API
call per resource. If an API call to one of these resources fails, the roleset
creation fails and OpenBao will attempt to rollback.

These rollbacks are API calls, so they may also fail. The secrets engine uses a
WAL to ensure that unused bindings are cleaned up. In the case of quota limits,
you may need to clean these up manually.

### Do not modify openbao-owned IAM accounts

While OpenBao will initially create and assign permissions to IAM service
accounts, it is possible that an external user deletes or modifies this service
account. These changes are difficult to detect, and it is best to prevent this
type of modification through IAM permissions.

OpenBao roleset Service Accounts will have emails in the format:

```
openbao<roleset-prefix>-<creation-unix-timestamp>@...
```

Communicate with your teams (or use IAM permissions) to not modify these
resources.

## API

The GCP secrets engine has a full HTTP API. Please see the [GCP secrets engine API docs][api]
for more details.

[api]: api.md
[cloud-creds]: https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application
[custom-roles]: https://cloud.google.com/iam/docs/creating-custom-roles
[gce]: https://cloud.google.com/compute/
[gke]: https://cloud.google.com/kubernetes-engine/
[iam-keys]: https://cloud.google.com/iam/docs/service-accounts#service_account_keys
[iam-roles]: https://cloud.google.com/iam/docs/understanding-roles
[predefined-roles]: https://cloud.google.com/iam/docs/understanding-roles#predefined_roles
[resource-name-full]: https://cloud.google.com/apis/design/resource_names#full_resource_name
[resource-name-relative]: https://cloud.google.com/apis/design/resource_names#relative_resource_name
[quotas]: https://cloud.google.com/compute/quotas
[service-accounts]: https://cloud.google.com/compute/docs/access/service-accounts
