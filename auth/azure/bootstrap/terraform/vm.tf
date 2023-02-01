provider "azuread" {}
provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}
data "azurerm_subscription" "current" {}
data "azuread_application_published_app_ids" "well_known" {}

locals {
  app_rw_owned_by_id = azuread_service_principal.ms_graph.app_role_ids["Application.ReadWrite.All"]
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

# Use system assigned managed identity
resource "azurerm_role_assignment" "vault_azure_msi_assignment" {
  role_definition_name = "Reader"
  scope                = azurerm_linux_virtual_machine.vault_azure_vm.id
  principal_id         = azurerm_linux_virtual_machine.vault_azure_vm.identity[0].principal_id
}

resource "azurerm_user_assigned_identity" "vault_azure_uid" {
  name                = "vault_azure_tests"
  resource_group_name = azurerm_resource_group.vault_azure_rg.name
  location            = azurerm_resource_group.vault_azure_rg.location
}

resource "azurerm_role_assignment" "app_assignment_vm_read" {
  role_definition_name = "Reader"
  scope                = azurerm_linux_virtual_machine.vault_azure_vm.id
  principal_id         = azuread_service_principal.vault_azure_sp.object_id
}

resource "random_id" "random" {
  byte_length = 4
}

resource "azurerm_resource_group" "vault_azure_rg" {
  name     = "vault_azure_tests_${random_id.random.hex}"
  location = var.region
}

resource "azurerm_virtual_network" "vault_azure_vnet" {
  name                = "vault_azure_tests"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.vault_azure_rg.location
  resource_group_name = azurerm_resource_group.vault_azure_rg.name
}

resource "azurerm_subnet" "vault_azure_subnet" {
  name                 = "vault_azure_tests"
  resource_group_name  = azurerm_resource_group.vault_azure_rg.name
  virtual_network_name = azurerm_virtual_network.vault_azure_vnet.name
  address_prefixes     = ["10.0.2.0/24"]
}

resource "azurerm_network_interface" "vault_azure_nic" {
  name                = "vault_azure_tests"
  location            = azurerm_resource_group.vault_azure_rg.location
  resource_group_name = azurerm_resource_group.vault_azure_rg.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.vault_azure_subnet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.vault_azure_pub_ip.id
  }
}

resource "azurerm_public_ip" "vault_azure_pub_ip" {
  name                = "vault_azure_tests"
  resource_group_name = azurerm_resource_group.vault_azure_rg.name
  location            = azurerm_resource_group.vault_azure_rg.location
  allocation_method   = "Static"
}

# Restrict SSH access to the local public IP
data "http" "my_ip" {
  url = "https://ifconfig.me/ip"
}

resource "azurerm_network_security_group" "vault_azure_sg" {
  name                = "vault_azure_tests"
  location            = azurerm_resource_group.vault_azure_rg.location
  resource_group_name = azurerm_resource_group.vault_azure_rg.name

  security_rule {
    name                       = "ssh"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = chomp(data.http.my_ip.response_body)
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_interface_security_group_association" "vault_azure_vm_sga" {
  network_interface_id      = azurerm_network_interface.vault_azure_nic.id
  network_security_group_id = azurerm_network_security_group.vault_azure_sg.id
}

resource "azurerm_linux_virtual_machine" "vault_azure_vm" {
  name                = "vault-azure-tests-vm"
  resource_group_name = azurerm_resource_group.vault_azure_rg.name
  location            = azurerm_resource_group.vault_azure_rg.location
  size                = "Standard_F1"
  admin_username      = "adminuser"
  network_interface_ids = [
    azurerm_network_interface.vault_azure_nic.id,
  ]

  admin_ssh_key {
    username   = "adminuser"
    public_key = file(var.ssh_public_key_path)
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }

  identity {
    type = "SystemAssigned, UserAssigned"

    identity_ids = [
      azurerm_user_assigned_identity.vault_azure_uid.id,
    ]
  }
}

data "external" "access_token_jwt" {
  program = ["bash", "${path.root}/scripts/imds_access_token.sh"]
  query = {
    vm_ip_address = azurerm_public_ip.vault_azure_pub_ip.ip_address
  }

  depends_on = [azurerm_linux_virtual_machine.vault_azure_vm]
}

resource "local_file" "setup_environment_file" {
  filename = "local_environment_setup.sh"
  content  = <<EOF
export ACCESS_TOKEN_JWT=${data.external.access_token_jwt.result.access_token}
export VM_NAME=${azurerm_linux_virtual_machine.vault_azure_vm.name}
export VM_IP_ADDRESS=${azurerm_public_ip.vault_azure_pub_ip.ip_address}
export RESOURCE_ID=${azurerm_linux_virtual_machine.vault_azure_vm.id}
export RESOURCE_GROUP_NAME=${azurerm_resource_group.vault_azure_rg.name}
export SUBSCRIPTION_ID=${data.azurerm_client_config.current.subscription_id}
export TENANT_ID=${data.azurerm_client_config.current.tenant_id}
export CLIENT_ID=${azuread_application.vault_azure_app.application_id}
export CLIENT_SECRET=${azuread_service_principal_password.vault_azure_sp_pwd.value}
EOF
}

output "access_token_jwt" {
  value     = data.external.access_token_jwt.result.access_token
  sensitive = true
}

output "vm_name" {
  value = azurerm_linux_virtual_machine.vault_azure_vm.name
}

output "vm_ip_address" {
  value = azurerm_public_ip.vault_azure_pub_ip.ip_address
}

output "resource_group_name" {
  value = azurerm_resource_group.vault_azure_rg.name
}

output "resource_id" {
  value = azurerm_linux_virtual_machine.vault_azure_vm.id
}

output "subscription_id" {
  value = data.azurerm_client_config.current.subscription_id
}

output "tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "client_id" {
  value = azuread_application.vault_azure_app.application_id
}

output "client_secret" {
  value     = azuread_service_principal_password.vault_azure_sp_pwd.value
  sensitive = true
}
