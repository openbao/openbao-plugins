# Azure auth method

The `azure` auth method allows authentication against OpenBao using
Azure Active Directory credentials. It treats Azure as a Trusted Third Party
and expects a [JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)
signed by Azure Active Directory for the configured tenant.

This method supports authentication for system-assigned and user-assigned
managed identities. See [Managed identities for Azure resources](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview)
for more information about these resources.

This documentation assumes the Azure method is mounted at the `/auth/azure`
path in OpenBao. Since it is possible to enable auth methods at any location,
please update your API calls accordingly.

## Prerequisites:

The Azure auth method requires client credentials to access Azure APIs. The following
are required to configure the auth method:

- A configured [Azure AD application](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-integrating-applications)
  which is used as the resource for generating MSI access tokens.
- Client credentials (shared secret) with read access to particular Azure Resource Manager
  resources. See [Azure AD Service to Service Client Credentials](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-protocols-oauth-service-to-service).

If OpenBao is hosted on Azure, Openbao can use MSI to access Azure instead of a shared secret.
A managed identity must be [enabled](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/)
on the resource that acquires the access token.

The following Azure [role assignments](https://learn.microsoft.com/en-us/azure/role-based-access-control/overview#role-assignments)
must be granted to the Azure AD application in order for the auth method to access Azure
APIs during authentication.

### Role assignments

~> **Note:** The role assignments are only required when the
[`vm_name`](/openbao/api-docs/auth/azure#vm_name), [`vmss_name`](/openbao/api-docs/auth/azure#vmss_name),
or [`resource_id`](/openbao/api-docs/auth/azure#resource_id) parameters are used on login.

| Azure Environment                                                                    | Login Parameter | Azure API Permission                                                                                         |
|--------------------------------------------------------------------------------------|-----------------|--------------------------------------------------------------------------------------------------------------|
| Virtual Machine                                                                      | `vm_name`       | `Microsoft.Compute/virtualMachines/*/read`                                                                   |
| Virtual Machine Scale Set ([Uniform Orchestration][vmss-uniform])                    | `vmss_name`     | `Microsoft.Compute/virtualMachineScaleSets/*/read`                                                           |
| Virtual Machine Scale Set ([Flexible Orchestration][vmss-flex])                      | `vmss_name`     | `Microsoft.Compute/virtualMachineScaleSets/*/read` `Microsoft.ManagedIdentity/userAssignedIdentities/*/read` |
| Services that ([support managed identities][managed-identities]) for Azure resources | `resource_id`   | `read` on the resource used to obtain the JWT                                                                |

[vmss-uniform]: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-orchestration-modes#scale-sets-with-uniform-orchestration
[vmss-flex]: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-orchestration-modes#scale-sets-with-flexible-orchestration
[managed-identities]: https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/managed-identities-status

### API permissions

The following [API permissions](https://learn.microsoft.com/en-us/azure/active-directory/develop/permissions-consent-overview#types-of-permissions)
must be assigned to the service principal provided to OpenBao for managing the root rotation in Azure:

| Permission Name               | Type        |
| ----------------------------- | ----------- |
| Application.ReadWrite.All     | Application |

## Authentication

### Via the CLI

The default path is `/auth/azure`. If this auth method was enabled at a different
path, specify `auth/my-path/login` instead.

```shell-session
$ bao write auth/azure/login \
    role="dev-role" \
    jwt="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
    subscription_id="12345-..." \
    resource_group_name="test-group" \
    vm_name="test-vm"
```

The `role` and `jwt` parameters are required. When using
`bound_service_principal_ids` and `bound_group_ids` in the token roles, all the
information is required in the JWT (except for `vm_name`, `vmss_name`, `resource_id`). When
using other `bound_*` parameters, calls to Azure APIs will be made and
`subscription_id`, `resource_group_name`, and `vm_name`/`vmss_name` are all required
and can be obtained through instance metadata.

For example:

```shell-session
$ bao write auth/azure/login role="dev-role" \
     jwt="$(curl -s 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fmanagement.azure.com%2F' -H Metadata:true | jq -r '.access_token')" \
     subscription_id=$(curl -s -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" | jq -r '.compute | .subscriptionId')  \
     resource_group_name=$(curl -s -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" | jq -r '.compute | .resourceGroupName') \
     vm_name=$(curl -s -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" | jq -r '.compute | .name')
```

### Via the API

The default endpoint is `auth/azure/login`. If this auth method was enabled
at a different path, use that value instead of `azure`.

```shell-session
$ curl \
    --request POST \
    --data '{"role": "dev-role", "jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}' \
    https://127.0.0.1:8200/v1/auth/azure/login
```

The response will contain the token at `auth.client_token`:

```json
{
  "auth": {
    "client_token": "f33f8c72-924e-11f8-cb43-ac59d697597c",
    "accessor": "0e9e354a-520f-df04-6867-ee81cae3d42d",
    "policies": ["default", "dev", "prod"],
    "lease_duration": 2764800,
    "renewable": true
  }
}
```

## Configuration

Auth methods must be configured in advance before machines can authenticate.
These steps are usually completed by an operator or configuration management
tool.

### Via the CLI

1. Enable Azure authentication in OpenBao:

   ```shell-session
   $ bao auth enable azure
   ```

1. Configure the Azure auth method:

   ```shell-session
   $ bao write auth/azure/config \
       tenant_id=7cd1f227-ca67-4fc6-a1a4-9888ea7f388c \
       resource=https://management.azure.com/ \
       client_id=dd794de4-4c6c-40b3-a930-d84cd32e9699 \
       client_secret=IT3B2XfZvWnfB98s1cie8EMe7zWg483Xy8zY004=
   ```

   For the complete list of configuration options, please see the API
   documentation.

1. Create a role:

   ```shell-session
   $ bao write auth/azure/role/dev-role \
       policies="prod,dev" \
       bound_subscription_ids=6a1d5988-5917-4221-b224-904cd7e24a25 \
       bound_resource_groups=openbao
   ```

   Roles are associated with an authentication type/entity and a set of OpenBao
   policies. Roles are configured with constraints specific to the
   authentication type, as well as overall constraints and configuration for
   the generated auth tokens.

   For the complete list of role options, please see the [API documentation](api.md).

### Via the API

1. Enable Azure authentication in OpenBao:

   ```shell-session
   $ curl \
       --header "X-OpenBao-Token: ..." \
       --request POST \
       --data '{"type": "azure"}' \
       https://127.0.0.1:8200/v1/sys/auth/azure
   ```

1. Configure the Azure auth method:

   ```shell-session
   $ curl \
       --header "X-OpenBao-Token: ..." \
       --request POST \
       --data '{"tenant_id": "...", "resource": "..."}' \
       https://127.0.0.1:8200/v1/auth/azure/config
   ```

1. Create a role:

   ```shell-session
   $ curl \
       --header "X-OpenBao-Token: ..." \
       --request POST \
       --data '{"policies": ["dev", "prod"], ...}' \
       https://127.0.0.1:8200/v1/auth/azure/role/dev-role
   ```

## Azure managed identities

There are two types of [managed identities](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview#managed-identity-types)
in Azure: System-assigned and User-assigned. System-assigned identities are unique to
every virtual machine in Azure. If the resources using Azure auth are recreated
frequently, using system-assigned identities could result in many OpenBao entities being
created. For environments with high ephemeral workloads, user-assigned identities are
recommended.


### Limitations

The TTL of the access token returned by Azure AD for a managed identity is
24hrs and is not configurable. See ([limitations of using managed identities][id-limitations])
for more info.

[id-limitations]: https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/managed-identity-best-practice-recommendations#limitation-of-using-managed-identities-for-authorization

## Azure debug logs

The Azure auth plugin supports debug logging which includes additional information
about requests and responses from the Azure API.

To enable the Azure debug logs, set the following environment variable on the OpenBao
server:

```shell
AZURE_GO_SDK_LOG_LEVEL=DEBUG
```

## API

The Azure Auth Plugin has a full HTTP API. Please see the [API documentation](api.md) for more details.
