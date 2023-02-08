## Unreleased

## v0.15.0

CHANGES:

* Changes user-agent header value to use correct Vault version information and include
  the plugin type and name in the comment section [[GH-123]](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/123)

FEATURES:

* Adds ability to persist an application for the lifetime of a role [[GH-98]](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/98)

IMPROVEMENTS:

* Updated dependencies [[GH-109](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/109)]
    * `github.com/Azure/azure-sdk-for-go v67.0.0+incompatible`
    * `github.com/Azure/go-autorest/autorest v0.11.28`
    * `github.com/Azure/go-autorest/autorest/azure/auth v0.5.11`
    * `github.com/hashicorp/go-hclog v1.3.1`
    * `github.com/hashicorp/go-uuid v1.0.3`
    * `github.com/hashicorp/vault/api v1.8.2`
    * `github.com/hashicorp/vault/sdk v0.6.1`
    * `github.com/mitchellh/mapstructure v1.5.0`
* Upgraded to go 1.19 [[GH-109](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/109)]

## v0.14.1

BUG FIXES:

* Adds WAL rollback mechanism to clean up Role Assignments during partial failure [[GH-110]](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/110)

## v0.14.0

IMPROVEMENTS:

* Add option to permanently delete AzureAD objects [[GH-104](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/104)]

CHANGES:

* Remove deprecated AAD graph code [[GH-101](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/101)]
* Remove partner ID from user agent string [[GH-95](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/95)]

## v0.11.4

CHANGES:

* Sets `use_microsoft_graph_api` to true by default [[GH-90](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/90)]

BUG FIXES:

* Fixes environment not being used when using MS Graph [[GH-87](https://github.com/hashicorp/vault-plugin-secrets-azure/pull/87)]
