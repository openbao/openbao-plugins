# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "azuread" {}
provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}
data "azurerm_subscription" "current" {}
data "azuread_application_published_app_ids" "well_known" {}
data "azuread_client_config" "current" {}

locals {
  app_rw_owned_by_id = azuread_service_principal.ms_graph.app_role_ids["Application.ReadWrite.All"]
  group_rw_all_id    = azuread_service_principal.ms_graph.app_role_ids["GroupMember.ReadWrite.All"]
}

resource "random_id" "random" {
  byte_length = 4
}

resource "azurerm_resource_group" "vault_azure_rg" {
  name     = "vault_azure_tests_${random_id.random.hex}"
  location = var.region
}

resource "azuread_application" "vault_azure_app" {
  display_name = "vault_azure_tests"

  # Details at https://learn.microsoft.com/en-us/graph/permissions-reference
  required_resource_access {
    resource_app_id = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph

    resource_access {
      id   = local.app_rw_owned_by_id
      type = "Role" # Application type
    }
    resource_access {
      id   = local.group_rw_all_id
      type = "Role" # Application type
    }
  }
}

resource "azuread_service_principal" "ms_graph" {
  application_id = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
  use_existing   = true
}

resource "azuread_service_principal" "vault_azure_sp" {
  application_id = azuread_application.vault_azure_app.application_id
}

resource "azuread_service_principal_password" "vault_azure_sp_pwd" {
  service_principal_id = azuread_service_principal.vault_azure_sp.id
}

resource "azuread_app_role_assignment" "app_admin_consent" {
  app_role_id         = local.app_rw_owned_by_id
  principal_object_id = azuread_service_principal.vault_azure_sp.object_id
  resource_object_id  = azuread_service_principal.ms_graph.object_id
}

resource "azuread_app_role_assignment" "group_admin_consent" {
  app_role_id         = local.group_rw_all_id
  principal_object_id = azuread_service_principal.vault_azure_sp.object_id
  resource_object_id  = azuread_service_principal.ms_graph.object_id
}

resource "azurerm_role_assignment" "vault_sp_read_assignment" {
  role_definition_name = "User Access Administrator"
  scope                = data.azurerm_subscription.current.id
  principal_id         = azuread_service_principal.vault_azure_sp.object_id
}

resource "azuread_group" "test_group" {
  display_name     = "azure-secrets-engine-test-group-${random_id.random.hex}"
  owners           = [data.azuread_client_config.current.object_id]
  security_enabled = true
}

resource "local_file" "setup_environment_file" {
  filename = "local_environment_setup.sh"
  content  = <<EOF
export AZURE_TEST_RESOURCE_GROUP=${azurerm_resource_group.vault_azure_rg.name}
export AZURE_SUBSCRIPTION_ID=${data.azurerm_client_config.current.subscription_id}
export AZURE_TENANT_ID=${data.azurerm_client_config.current.tenant_id}
export AZURE_GROUP_NAME=${azuread_group.test_group.display_name}
export AZURE_APPLICATION_OBJECT_ID=${azuread_application.vault_azure_app.object_id}
export AZURE_CLIENT_ID=${azuread_application.vault_azure_app.application_id}
export AZURE_CLIENT_SECRET=${azuread_service_principal_password.vault_azure_sp_pwd.value}
EOF
}

output "resource_group_name" {
  value = azurerm_resource_group.vault_azure_rg.name
}

output "subscription_id" {
  value = data.azurerm_client_config.current.subscription_id
}

output "tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "group_name" {
  value = azuread_group.test_group.display_name
}

# Application Object ID for an existing service principal that can be used
# instead of creating dynamic service principals
# https://developer.hashicorp.com/vault/api-docs/secret/azure#application_object_id
output "application_object_id" {
  value = azuread_application.vault_azure_app.object_id
}

output "client_id" {
  value = azuread_application.vault_azure_app.application_id
}

output "client_secret" {
  value     = azuread_service_principal_password.vault_azure_sp_pwd.value
  sensitive = true
}
