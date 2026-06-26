# Google Cloud auth method

The `gcp` auth method allows Google Cloud Platform entities to authenticate to
OpenBao. OpenBao treats Google Cloud as a trusted third party and verifies
authenticating entities against the Google Cloud APIs. This backend allows for
authentication of:

- Google Cloud IAM service accounts
- Google Compute Engine (GCE) instances

This backend focuses on identities specific to Google _Cloud_ and does not
support authenticating arbitrary Google or Google Workspace users or generic OAuth
against Google.

## Authentication

### Via the CLI helper

OpenBao includes a CLI helper that obtains a signed JWT locally and sends the
request to OpenBao.

```shell-session
# Authentication to openbao outside of Google Cloud
$ bao login -method=gcp \
    role="my-role" \
    service_account="authenticating-account@my-project.iam.gserviceaccount.com" \
    jwt_exp="15m" \
    credentials=@path/to/signer/credentials.json
```

```shell-session
# Authentication to openbao inside of Google Cloud
$ bao login -method=gcp role="my-role"
```

For more usage information, run `bao auth help gcp`.

### Via the CLI

```shell-session
$ bao write -field=token auth/gcp/login \
    role="my-role" \
    jwt="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

See [Generating JWTs](#generating-jwts) for ways to obtain the JWT token.

### Via the API

```shell-session
$ curl \
    --request POST \
    --data '{"role":"my-role", "jwt":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}' \
    http://127.0.0.1:8200/v1/auth/gcp/login
```

See [API docs][api-docs] for expected response.

## Configuration

Auth methods must be configured in advance before users or machines can
authenticate. These steps are usually completed by an operator or configuration
management tool.

1. Enable the Google Cloud auth method:

   ```shell-session
   $ bao auth enable gcp
   ```

1. Configure the auth method credentials if OpenBao is not running on Google Cloud:

   ```text
   $ openbao write auth/gcp/config \
       credentials=@/path/to/credentials.json
   ```

   If you are using instance credentials or want to specify credentials via
   an environment variable, you can skip this step. To learn more, see the
   [Google Cloud Credentials](#gcp-credentials) section below.

   -> **Note**: If you're using a [Private Google Access](https://cloud.google.com/vpc/docs/configure-private-google-access)
   environment, you will additionally need to configure your environment’s custom endpoints
   via the `custom_endpoint` configuration parameter.

1. Create a named role:

   For an `iam`-type role:

   ```shell-session
   $ bao write auth/gcp/role/my-iam-role \
       type="iam" \
       policies="dev,prod" \
       bound_service_accounts="my-service@my-project.iam.gserviceaccount.com"
   ```

   For a `gce`-type role:

   ```shell-session
   $ bao write auth/gcp/role/my-gce-role \
       type="gce" \
       policies="dev,prod" \
       bound_projects="my-project1,my-project2" \
       bound_zones="us-east1-b" \
       bound_labels="foo:bar,zip:zap" \
       bound_service_accounts="my-service@my-project.iam.gserviceaccount.com"
   ```

   Note that `bound_service_accounts` is only required for `iam`-type roles.

   For the complete list of configuration options for each type, please see the
   [API documentation][api-docs].

## GCP credentials

The Google Cloud OpenBao auth method uses the official Google Cloud Golang SDK.
This means it supports the common ways of [providing credentials to Google
Cloud][cloud-creds].

1. The environment variable `GOOGLE_APPLICATION_CREDENTIALS`. This is specified
   as the **path** to a Google Cloud credentials file, typically for a service
   account. If this environment variable is present, the resulting credentials are
   used. If the credentials are invalid, an error is returned.

1. Default instance credentials. When no environment variable is present, the
   default service account credentials are used.

For more information on service accounts, please see the [Google Cloud Service
Accounts documentation][service-accounts].

To use this auth method, the service account must have the following minimum
scope:

```text
https://www.googleapis.com/auth/cloud-platform
```

### Required GCP permissions

#### Enabled GCP APIs

The GCP project must have the following APIs enabled:

- [iam.googleapis.com](https://console.cloud.google.com/flows/enableapi?apiid=iam.googleapis.com)
  for `iam` and `gce` type roles.
- [compute.googleapis.com](https://console.cloud.google.com/flows/enableapi?apiid=compute.googleapis.com)
  for `gce` type roles.
- [cloudresourcemanager.googleapis.com](https://console.cloud.google.com/flows/enableapi?apiid=cloudresourcemanager.googleapis.com)
  for `iam` and `gce` type roles that set [`add_group_aliases`](/openbao/api-docs/auth/gcp#add_group_aliases) to true.

#### OpenBao server permissions

**For `iam`-type OpenBao roles**, the service account `credentials`
given to OpenBao can have the following role:

```text
roles/iam.serviceAccountKeyAdmin
```

**For `gce`-type OpenBao roles**, the service account `credentials`
given to OpenBao can have the following role:

```text
roles/compute.viewer
```

If you instead wish to create a custom role with only the exact GCP permissions
required, use the following list of permissions:

```text
iam.serviceAccounts.get
iam.serviceAccountKeys.get
compute.instances.get
compute.instanceGroups.list
```

These allow OpenBao to:

- verify the service account, either directly authenticating or associated with
  authenticating GCE instance, exists
- get the corresponding public keys for verifying JWTs signed by service account
  private keys.
- verify authenticating GCE instances exist
- compare bound fields for GCE roles (zone/region, labels, or membership
  in given instance groups)

If you are using Group Aliases as described below, you will also need to add the
`resourcemanager.projects.get` permission.

#### Permissions for authenticating against OpenBao

If you are authenticating to OpenBao from Google Cloud, you can skip the following step as
OpenBao will generate and present the identity token of the service account configured
on the instance or the pod.

Note that the previously mentioned permissions are given to the _OpenBao servers_.
The IAM service account or GCE instance that is **authenticating against OpenBao**
must have the following role:

```text
roles/iam.serviceAccountTokenCreator
```

> [!WARNING] 
> Make sure this role is only applied so your service account can
> impersonate itself. If this role is applied GCP project-wide, this will allow
> the service account to impersonate any service account in the GCP project where
> it resides.  See [Managing service account
> impersonation](https://cloud.google.com/iam/docs/impersonating-service-accounts)
> for more information.

## Group aliases

As of OpenBao 1.0, roles can specify an `add_group_aliases` boolean parameter
that adds [group aliases][identity-group-aliases] to the auth response. These
aliases can aid in building reusable policies since they are available as
interpolated values in OpenBao's policy engine. Once enabled, the auth response
will include the following aliases:

```json
[
  "project-$PROJECT_ID",
  "folder-$SUBFOLDER_ID",
  "folder-$FOLDER_ID",
  "organization-$ORG_ID"
]
```

If you are using a custom role for OpenBao server, you will need to add the
`resourcemanager.projects.get` permission to your custom role.

## Implementation details

This section describes the implementation details for how OpenBao communicates
with Google Cloud to authenticate and authorize JWT tokens. This information is
provided for those who are curious, but these details are not
required knowledge for using the auth method.

### IAM login

IAM login applies only to roles of type `iam`. The OpenBao authentication workflow
for IAM service accounts looks like this:

1. The client generates a signed JWT using the Service Account Credentials
   [`projects.serviceAccounts.signJwt`][signjwt-method] API method. For examples
   of how to do this, see the [Generating JWTs](#generating-jwts) section.

2. The client sends this signed JWT to OpenBao along with a role name.

3. OpenBao extracts the `kid` header value, which contains the ID of the
   key-pair used to generate the JWT, and the `sub` ID/email to find the service
   account key. If the service account does not exist or the key is not linked to
   the service account, OpenBao denies authentication.

4. OpenBao authorizes the confirmed service account against the given role. If
   that is successful, a OpenBao token with the proper policies is returned.

### GCE login

GCE login only applies to roles of type `gce` and **must be completed on an
infrastructure running on Google Cloud**. These steps will not work from your 
local laptop or another cloud provider.

1. The client obtains an [instance identity metadata token][instance-identity]
   on a GCE instance.

2. The client sends this JWT to OpenBao along with a role name.

3. OpenBao extracts the `kid` header value, which contains the ID of the
   key-pair used to generate the JWT, to find the OAuth2 public cert to verify
   this JWT.

4. OpenBao authorizes the confirmed instance against the given role, ensuring
   the instance matches the bound zones, regions, or instance groups. If that is
   successful, a OpenBao token with the proper policies is returned.

## Generating JWTs

This section details the various methods and examples for obtaining JWT
tokens.

### Service account credentials API

This describes how to use the GCP Service Account Credentials [API method][signjwt-method]
directly to generate the signed JWT with the claims that OpenBao expects. Note the CLI
does this process for you and is much easier, and that there is very little
reason to do this yourself.

#### curl example

OpenBao requires the following minimum claim set:

```json
{
  "sub": "$SERVICE_ACCOUNT_EMAIL_OR_ID",
  "aud": "openbao/$ROLE",
  "exp": "$EXPIRATION"
}
```

For the API method, providing the expiration claim `exp` is required. If it is omitted,
it will not be added automatically and OpenBao will deny authentication. Expiration must
be specified as a [NumericDate](https://tools.ietf.org/html/rfc7519#section-2) value
(seconds from Epoch). This value must be before the max JWT expiration allowed for a
role. This defaults to 15 minutes and cannot be more than 1 hour.

If a user generates a token that expires after 15 minutes, and the gcp role has `max_jwt_exp` set to the default, OpenBao will return the following error: `Expiration date must be set to no more that 15 mins in JWT_CLAIM, otherwise the login request returns error "role requires that service account JWTs expire within 900 seconds`. In this case, the user must create a new signed JWT with a shorter expiration, or set `max_jwt_exp` to a higher value in the gcp role.

One you have all this information, the JWT token can be signed using curl and
[oauth2l](https://github.com/google/oauth2l):

```shell-session
ROLE="my-role"
SERVICE_ACCOUNT="service-account@my-project.iam.gserviceaccount.com"
OAUTH_TOKEN="$(oauth2l header cloud-platform)"
EXPIRATION="<your_token_expiration>"
JWT_CLAIM="{\\\"aud\\\":\\\"openbao/${ROLE}\\\", \\\"sub\\\": \\\"${SERVICE_ACCOUNT}\\\", \\\"exp\\\": ${EXPIRATION}}"

$ curl \
  --header "${OAUTH_TOKEN}" \
  --header "Content-Type: application/json" \
  --request POST \
  --data "{\"payload\": \"${JWT_CLAIM}\"}" \
  "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/${SERVICE_ACCOUNT}:signJwt"
```

#### gcloud example

You can also do this through the (currently beta) gcloud command. Note that you will
be required to provide the expiration claim `exp` as a part of the JWT input to the
command.

```shell-session
$ gcloud beta iam service-accounts sign-jwt $INPUT_JWT_CLAIMS $OUTPUT_JWT_FILE \
    --iam-account=service-account@my-project.iam.gserviceaccount.com \
    --project=my-project
```

#### Golang example

Read more on the
[Google Open Source blog](https://opensource.googleblog.com/2017/08/hashicorp-openbao-and-google-cloud-iam.html).

### GCE

GCE tokens **can only be generated from a GCE instance**.

1.  OpenBao can automatically discover the identity token on a GCE/GKE instance. This simplifies
    authenticating to OpenBao like so:

    ```shell-session
    $ bao login \
      -method=gcp \
      role="my-gce-role"
    ```

1.  The JWT token can also be obtained from the `"service-accounts/default/identity"` endpoint for a
    instance's metadata server.

    #### Curl example

    ```shell-session
    ROLE="my-gce-role"

    $ curl \
      --header "Metadata-Flavor: Google" \
      --get \
      --data-urlencode "audience=http://openbao/${ROLE}" \
      --data-urlencode "format=full" \
      "http://metadata/computeMetadata/v1/instance/service-accounts/default/identity"
    ```

## API

The GCP Auth Plugin has a full HTTP API. Please see the
[API docs][api-docs] for more details.

[jwt]: https://tools.ietf.org/html/rfc7519
[signjwt-method]: https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/signJwt
[cloud-creds]: https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application
[service-accounts]: https://cloud.google.com/compute/docs/access/service-accounts
[api-docs]: api.md
[identity-group-aliases]: https://openbao.org/api-docs/secret/identity/group-alias/
[instance-identity]: https://cloud.google.com/compute/docs/instances/verifying-instance-identity
