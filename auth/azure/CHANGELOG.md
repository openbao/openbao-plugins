## Unreleased

FEATURES:

* Allow any Azure resource that supports managed identities to authenticate with Vault [GH-71](https://github.com/hashicorp/vault-plugin-auth-azure/pull/71)
* Adds support for Virtual Machine Scale Set Flex Authentication [GH-63](https://github.com/hashicorp/vault-plugin-auth-azure/pull/63)

IMPROVEMENTS:

* Updates dependencies [[GH-65](https://github.com/hashicorp/vault-plugin-auth-azure/pull/65)]
  * `github.com/Azure/azure-sdk-for-go v67.0.0+incompatible`
  * `github.com/Azure/go-autorest/autorest v0.11.28`
  * `github.com/hashicorp/go-hclog v1.3.1`
  * `github.com/hashicorp/vault/api v1.8.2`
  * `github.com/hashicorp/vault/api v1.8.2`
  * `github.com/hashicorp/vault/sdk v0.6.1`
  * `golang.org/x/oauth2 v0.1.0`
* Upgrades to Go 1.19 [[GH-65](https://github.com/hashicorp/vault-plugin-auth-azure/pull/65)]
