# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Configure Terraform
terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.1.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.1.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.29.1"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "2.30.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "0.7.2"
    }
  }
}

# Configure the Azure Active Directory Provider
provider "azuread" {
  tenant_id = var.tenant_id
}

# Configure the Azure Provider
provider "azurerm" {
  features {}
}

variable "tenant_id" {
  description = "Tenant ID that should be used."
}

variable "name_prefix" {
  description = "Prefix all resources with name."

}

resource "random_id" "name" {
  byte_length = 8
  prefix      = var.name_prefix
}

locals {
  name = random_id.name.dec
}

data "azurerm_subscription" "primary" {}

data "azurerm_client_config" "vault_azure_secrets" {}

data "azuread_client_config" "current" {}

resource "azuread_application" "vault_azure_secrets" {
  display_name     = local.name
  identifier_uris  = ["api://${local.name}"]
  owners           = [data.azuread_client_config.current.object_id]
  sign_in_audience = "AzureADMyOrg"

  feature_tags {
    enterprise = false
    gallery    = true
  }


  required_resource_access {
    # Microsoft Graph
    resource_app_id = "00000003-0000-0000-c000-000000000000" # Microsoft Graph

    resource_access {
      id   = "df021288-bdef-4463-88db-98f22de89214" # User.Read.All
      type = "Role"
    }

    resource_access {
      id   = "b4e74841-8e56-480b-be8b-910348b18b4c" # User.ReadWrite
      type = "Scope"
    }

    resource_access {
      id   = "1bfefb4e-e0b5-418b-a88f-73c46d2cc8e9" # Application.ReadWrite.All
      type = "Role"
    }

    resource_access {
      id   = "19dbc75e-c2e2-444c-a770-ec69d8559fc7" # Directory.ReadWrite.All
      type = "Role"
    }

    resource_access {
      id   = "62a82d76-70ea-41e2-9197-370581804d09" # Group.ReadWrite.All
      type = "Role"
    }
  }
}

resource "time_rotating" "vault_azure_secrets" {
  rotation_days = 7
}

resource "azuread_application_password" "vault_azure_secrets" {
  application_object_id = azuread_application.vault_azure_secrets.object_id
  rotate_when_changed = {
    rotation = time_rotating.vault_azure_secrets.id
  }
}

resource "azuread_service_principal" "vault_azure_secrets" {
  description                  = local.name
  application_id               = azuread_application.vault_azure_secrets.application_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_group" "vault_azure_secrets" {
  display_name     = local.name
  owners           = [data.azuread_client_config.current.object_id]
  security_enabled = true

  members = [
    azuread_service_principal.vault_azure_secrets.object_id
  ]
}

resource "azurerm_resource_group" "vault_azure_secrets" {
  name     = local.name
  location = "East US"
}

data "azurerm_role_definition" "builtin" {
  name = "Owner"
}

resource "azurerm_role_assignment" "vault_azure_secrets" {
  scope              = data.azurerm_subscription.primary.id
  principal_id       = azuread_service_principal.vault_azure_secrets.object_id
  role_definition_id = "/subscriptions/${data.azurerm_subscription.primary.subscription_id}${data.azurerm_role_definition.builtin.id}"
}

resource "null_resource" "admin_consent" {
  depends_on = [
    azuread_application.vault_azure_secrets,
    azurerm_role_assignment.vault_azure_secrets,
  ]
  provisioner "local-exec" {
    command = <<HERE
for i in {0..60} ; do
  az ad app permission admin-consent --id "${azuread_application.vault_azure_secrets.application_id}" &> /dev/null && exit 0
  sleep .5
done
exit 1
HERE
  }
}

resource "null_resource" "wait_grants" {
  depends_on = [
    null_resource.admin_consent
  ]
  provisioner "local-exec" {
    command = <<HERE
for i in {0..60} ; do
  len=$(az ad app permission list-grants --id "${azuread_application.vault_azure_secrets.application_id}" | jq '. | length')
  [ "$len" -gt 0 ] && exit 0
  sleep .5
done
exit 1
HERE
  }
}

output "application_id" {
  value = azuread_application.vault_azure_secrets.application_id
}

output "application_password_value" {
  sensitive = true
  value     = azuread_application_password.vault_azure_secrets.value
}

output "application_password_id" {
  sensitive = true
  value     = azuread_application_password.vault_azure_secrets.id
}

output "role_assignment_principal_type" {
  value = azurerm_role_assignment.vault_azure_secrets.principal_type
}

output "role_assignment_principal_id" {
  value = azurerm_role_assignment.vault_azure_secrets.principal_id
}

output "service_principal_id" {
  value = azuread_service_principal.vault_azure_secrets.id
}

output "resource_group_name" {
  value = azurerm_resource_group.vault_azure_secrets.name
}

output "tenant_id" {
  sensitive = true
  value     = var.tenant_id
}

output "subscription_id" {
  sensitive = true
  value     = data.azurerm_subscription.primary.subscription_id
}

output "name" {
  value = local.name
}
