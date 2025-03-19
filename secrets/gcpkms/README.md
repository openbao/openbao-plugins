# OpenBao Secrets Engine for Google Cloud KMS



This is a standalone backend plugin for use with [OpenBao](https://www.github.com/openbao/openbao) that manages [Google Cloud
KMS][kms] keys and provides pass-through encryption/decryption of data through
KMS.

## Getting Started

This is an [OpenBao plugin](https://openbao.org/docs/plugins/)
and is meant to work with OpenBao. This guide assumes you have already installed Openbao
and have a basic understanding of how OpenBao works.

Otherwise, first read this guide on how to [get started with
OpenBao](https://openbao.org/docs/get-started/developer-qs/).

To learn specifically about how plugins work, see documentation on [OpenBao plugins](https://openbao.org/docs/plugins/).

### Usage

Please see [documentation for the plugin](./docs/index.md) in this repository.

## Developing

If you wish to work on this plugin, you'll first need [Go](https://www.golang.org)
installed on your machine (whichever version is required by OpenBao).

Make sure Go is properly installed, including setting up a [GOPATH](https://golang.org/doc/code.html#GOPATH).

To build the binary run:

    ```text
    $ make dev
    ```

    The plugin binary will be written to the `./bin` directory.

Run OpenBao plugins from that directory:

    ```text
    $ bao server -dev -dev-plugin-dir=./bin
    $ bao secrets enable -path=gcpkms -plugin=openbao-plugin-secrets-gcpkms plugin
    ```

### Tests

This plugin has both unit tests and acceptance tests. To run the acceptance
tests, you must:

- Have a service account in the project with the roles "Cloud KMS Admin" and "Cloud KMS Crypto Operator"
- Set `GOOGLE_APPLICATION_CREDENTIALS` to the service account key credentials for the above account
- Set `GOOGLE_CLOUD_PROJECT` to the name of the project
- Request an increase to the Cloud Key Management Service (KMS) API Write-Requests quota to 600 per minute

We recommend running tests in a dedicated Google Cloud project. On a fresh
project, you will need to enable the Cloud KMS API. This operation only needs to
be completed once per project.

```text
$ gcloud services enable cloudkms.googleapis.com --project $GOOGLE_CLOUD_PROJECT
```

After the API is enabled, it may take a few minutes to propagate. Please wait
and try again.

To run the tests:

```text
$ make test
```

**Warning:** the acceptance tests change real resources which may incur real
costs. Please run acceptance tests at your own risk.

### Cleanup

If a test panics or fails to cleanup, you can be left with orphaned KMS keys.
While their monthly cost is minimal, this may be undesirable. As such, there a
cleanup script is included. To execute this script, run:

```text
$ export GOOGLE_CLOUD_PROJECT=my-test-project
$ go run test/cleanup/main.go
```

**WARNING!** This will delete all keys in most key rings, so do not run this
against a production project!

[kms]: https://cloud.google.com/kms
