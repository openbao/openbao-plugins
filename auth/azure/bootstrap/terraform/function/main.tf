# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "random" {}
provider "azuread" {}
provider "azurerm" {
  features {
    resource_group {
      # the function is published via the Azure CLI so we will ensure any
      # new resources are also clean up on destroy
      prevent_deletion_if_contains_resources = false
    }
  }
}

resource "random_pet" "main" {
  length = 2
}

# use a random suffix for the resource group so we don't get resource
# collisions in Azure
resource "azurerm_resource_group" "main" {
  name     = "${var.project}-${var.env}-${random_pet.main.id}"
  location = var.region
}

resource "azuread_application" "main" {
  display_name = "${var.project}-${var.env}"
}

resource "azuread_service_principal" "main" {
  application_id = azuread_application.main.application_id
}

resource "azuread_service_principal_password" "main" {
  service_principal_id = azuread_service_principal.main.id
}

resource "azurerm_user_assigned_identity" "main" {
  name                = "${var.project}-${var.env}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
}

resource "azurerm_role_assignment" "main" {
  role_definition_name = "Reader"
  scope                = azurerm_linux_function_app.main.id
  principal_id         = azuread_service_principal.main.object_id
}

resource "azurerm_storage_account" "main" {
  name = "vltauthstorage"
  resource_group_name = azurerm_resource_group.main.name
  location = var.region
  account_tier = "Standard"
  account_replication_type = "LRS"
}

# Application Insights is a component of Azure Monitor which allows you to
# collect metrics and logs from your function app
resource "azurerm_application_insights" "main" {
  name                = "${var.project}-${var.env}"
  location            = var.region
  resource_group_name = azurerm_resource_group.main.name
  application_type    = "web"
}

# A Function App must always be associated with an App Service Plan which
# defines the compute resources available and how it scales
resource "azurerm_service_plan" "main" {
  name                = "${var.project}-${var.env}"
  resource_group_name = azurerm_resource_group.main.name
  location            = var.region
  os_type             = "Linux"
  sku_name            = "Y1"
}

resource "azurerm_linux_function_app" "main" {
  name                       = "${var.project}-${var.env}"
  resource_group_name        = azurerm_resource_group.main.name
  location                   = var.region
  service_plan_id            = azurerm_service_plan.main.id
  app_settings = {
    # this sets the `CLIENT_ID` env variable for our Azure function
    "CLIENT_ID" = "${azurerm_user_assigned_identity.main.client_id}",

    "FUNCTIONS_WORKER_RUNTIME" = "python",
    "APPINSIGHTS_INSTRUMENTATIONKEY" = azurerm_application_insights.main.instrumentation_key,
  }
  site_config {
    application_insights_key               = azurerm_application_insights.main.instrumentation_key
    application_insights_connection_string = azurerm_application_insights.main.connection_string
    application_stack {
      python_version = "3.9"
    }
  }

  identity {
    type = "UserAssigned"

    identity_ids = [
      azurerm_user_assigned_identity.main.id,
    ]
  }
  storage_account_name       = azurerm_storage_account.main.name
  storage_account_access_key = azurerm_storage_account.main.primary_access_key
}

data "azurerm_client_config" "current" {}

resource "local_file" "setup_environment_file" {
  filename = "local_environment_setup.sh"
  content  = <<EOF
export RESOURCE_ID=${azurerm_linux_function_app.main.id}
export RESOURCE_GROUP_NAME=${azurerm_resource_group.main.name}
export SUBSCRIPTION_ID=${data.azurerm_client_config.current.subscription_id}
export TENANT_ID=${data.azurerm_client_config.current.tenant_id}
export CLIENT_ID=${azuread_application.main.application_id}
export CLIENT_SECRET=${azuread_service_principal_password.main.value}
export AZURE_FUNC_HOSTNAME=${azurerm_linux_function_app.main.default_hostname}
EOF
}

output "function_app_name" {
  value = azurerm_linux_function_app.main.name
  description = "Deployed function app name"
}

output "function_app_default_hostname" {
  value = azurerm_linux_function_app.main.default_hostname
  description = "Deployed function app hostname"
}

output "resource_id" {
  value = azurerm_linux_function_app.main.name
}

output "resource_group_name" {
  value = azurerm_resource_group.main.name
}

output "subscription_id" {
  value = data.azurerm_client_config.current.subscription_id
}

output "tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "client_id" {
  value = azuread_application.main.application_id
}

output "client_secret" {
  value     = azuread_service_principal_password.main.value
  sensitive = true
}
