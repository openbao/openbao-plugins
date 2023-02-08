## Unreleased

## v0.13.0

FEATURES:

* Add rotate root support to Azure Auth [GH-88](https://github.com/hashicorp/vault-plugin-auth-azure/pull/88)
* Allow any Azure resource that supports managed identities to authenticate with Vault [GH-71](https://github.com/hashicorp/vault-plugin-auth-azure/pull/71)
* Adds support for Virtual Machine Scale Set Flex Authentication [GH-63](https://github.com/hashicorp/vault-plugin-auth-azure/pull/63)

IMPROVEMENTS:
* Updates dependencies
  * `github.com/Azure/azure-sdk-for-go v67.2.0+incompatible`[[GH-88](https://github.com/hashicorp/vault-plugin-auth-azure/pull/88)]
  * `github.com/Azure/go-autorest/autorest v0.11.28` [[GH-65](https://github.com/hashicorp/vault-plugin-auth-azure/pull/65)]
  * `github.com/hashicorp/vault/api v1.8.3` [[GH-82](https://github.com/hashicorp/vault-plugin-auth-azure/pull/82)]
  * `github.com/hashicorp/vault/sdk v0.7.0` [[GH-82](https://github.com/hashicorp/vault-plugin-auth-azure/pull/82)]
  * `github.com/hashicorp/go-hclog v1.4.0`[[GH-78](https://github.com/hashicorp/vault-plugin-auth-azure/pull/78)]
  * `golang.org/x/oauth2 v0.4.0`[[GH-84](https://github.com/hashicorp/vault-plugin-auth-azure/pull/84)]

* Upgrades to Go 1.19 [[GH-65](https://github.com/hashicorp/vault-plugin-auth-azure/pull/65)]
