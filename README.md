# OpenBao Plugins

This repository contains plugins for
[OpenBao](https://github.com/openbao/openbao), an open-source fork of HashiCorp
Vault. These plugins are maintained by the OpenBao project but are not included
in the core OpenBao binary.

To learn more about OpenBao plugins, please see the
[plugin system](https://openbao.org/docs/plugins/) documentation.

## Plugins

### Authentication Plugins

- **AWS** - Authenticate using AWS IAM credentials.
- **Azure** - Authenticate using Microsoft Azure credentials.
- **GCP** - Authenticate using Google Cloud Platform credentials.
- **GitHub** - Authenticate using GitHub credentials.

### Database Plugins

- **MongoDB** - Generate MongoDB database credentials.

### Secrets Plugins

- **AWS** - Generate AWS access credentials based on IAM policies.
- **Azure** - Generate Azure service principals with role and group assignments.
- **GCP** - Generate GCP service account keys and OAuth tokens based on IAM policies.
- **GCPKMS** - Encrypt data and manage keys via GCP KMS.
- **Nomad** - Generate Nomad ACL tokens.
- **Consul** - Generate Consul ACL tokens.

### KMS (Key Management Service) Plugins

- **AliCloud KMS** - Auto Unseal via AliCloud.
- **AWS KMS** - Auto Unseal via AWS.
- **Azure Key Vault** - Auto Unseal via Azure.
- **Google Cloud KMS** - Auto Unseal via Google Cloud.
- **OCI KMS** - Auto Unseal via Oracle Cloud.
- **OVHcloud KMS** - Auto Unseal via OVHcloud.
- **PKCS#11** - Auto Unseal via PKCS#11.
- **T Cloud Public KMS** - Auto Unseal via T Cloud Public.

## Download

Prebuilt binaries and their SHA-256 digests are available in the
[Releases](https://github.com/openbao/openbao-plugins/releases) section.
Prebuilt OCI artifacts are available on [ghcr.io](https://github.com/orgs/openbao/packages?repo_name=openbao-plugins).

## Installation

1. **Declare the plugin** to download and register it.

   ```hcl
   plugin_directory = "/path/to/download/dir"

   plugin_auto_download = true
   plugin_auto_register = true

   plugin "auth" "aws" {
      image       = "ghcr.io/openbao/openbao-plugin-auth-aws"
      version     = "v0.1.1"
      binary_name = "openbao-plugin-auth-aws"
      sha256sum   = "7a77057e62973c1aae6035f52110e3302605a47b622756d954915b7b55eca10c"
   }
   ```

   See the [OpenBao Plugin Configuration](https://openbao.org/docs/configuration/plugins)
   documentation for more details (such as manual plugin download).

2. **Enable the plugin.**
   E.g., for auth plugins:

   ```sh
   bao auth enable aws
   ```

## Development

To contribute or build plugins from source, follow these steps:

1. **Build the plugin**

   ```sh
   go build -o openbao-plugin-auth-aws ./auth/aws
   ```

2. Proceed as with a manually downloaded binary (place in `plugin_directory`,
   register, enable).

## Contributing

We welcome contributions! Please follow our [contribution
guidelines](https://github.com/openbao/openbao/blob/main/CONTRIBUTING.md) to
submit issues, improvements, or new plugins.

## License

This project is licensed under the [Mozilla Public License 2.0
(MPL-2.0)](LICENSE). Individual plugins may have different licenses, which will
be specified in their respective plugin directories.

