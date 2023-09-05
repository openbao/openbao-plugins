## Unreleased

## v0.15.1 (September 5th, 2023)

IMPROVEMENTS:

* Updated dependencies:
   * `cloud.google.com/go/kms` v1.6.0 -> v1.15.1
   * `github.com/gammazero/workerpool` v0.0.0-20190406235159-88d534f22b56 -> v1.1.3
   * `github.com/golang/protobuf` v1.5.2 -> v1.5.3
   * `github.com/hashicorp/go-hclog` v0.16.2 -> v1.5.0
   * `github.com/hashicorp/vault/api` v1.9.0 -> v1.9.2
   * `github.com/hashicorp/vault/sdk` v0.9.0 -> v0.9.2
   * `golang.org/x/oauth2` v0.4.0 -> v0.11.0
   * `google.golang.org/api` v0.103.0 -> v0.138.0
   * `google.golang.org/genproto` v0.0.0-20230110181048-76db0878b65f -> v0.0.0-20230822172742-b8732ec3820d
   * `google.golang.org/grpc` v1.47.0 -> v1.57.0

## v0.15.0

IMPROVEMENTS:

* enable plugin multiplexing [GH-26](https://github.com/hashicorp/vault-plugin-secrets-gcpkms/pull/26)
* update dependencies
  * `github.com/hashicorp/vault/api` v1.9.0
  * `github.com/hashicorp/vault/sdk` v0.8.1

## v0.14.0

CHANGES:

* Changes user-agent header value to use correct Vault version information and include
  the plugin type and name in the comment section. [[GH-21](https://github.com/hashicorp/vault-plugin-secrets-gcpkms/pull/21)]
* CreateOperation should only be implemented alongside ExistenceCheck [[GH-20](https://github.com/hashicorp/vault-plugin-secrets-gcpkms/pull/20)]

IMPROVEMENTS:

* Dependency updates
  * google.golang.org/api v0.5.0 => v0.83.0
  * github.com/hashicorp/vault/api v1.0.5-0.20200215224050-f6547fa8e820 => v1.8.3
  * github.com/hashicorp/vault/sdk v0.1.14-0.20200215224050-f6547fa8e820 => v0.7.0
  * golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 => v0.5.0
  * golang.org/x/net 0.0.0-20220722155237-a158d28d115b => v0.5.0
