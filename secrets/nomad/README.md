# OpenBao Plugin: Nomad Secrets Backend

This is a standalone backend plugin for use with
[OpenBao](https://www.github.com/openbao/openbao).  This plugin generates Nomad
ACL tokens dynamically based on pre-existing Nomad ACL policies.

## Quick Links
- [Docs](docs/readme.md)
- [API docs](docs/api/readme.md)
- Main Project Github: https://www.github.com/openbao/openbao

## Getting Started

This is a OpenBao plugin and is meant to work with OpenBao. This guide assumes
you have already installed OpenBao and have a basic understanding of how OpenBao
works.

To learn specifically about how plugins work, see documentation on [OpenBao
plugins](https://github.com/openbao/openbao/blob/development/website/content/docs/plugins/index.mdx).

## Usage

Please see [documentation for the plugin](docs/readme.md).

## Developing

If you wish to work on this plugin, you'll first need
[Go](https://www.golang.org) installed on your machine.

To compile a development version of this plugin, run `go build`.  This will put
the plugin binary in the repository root folder.

```sh
$ go build
```

Once you've done that, you can test your new plugin version in OpenBao. To do
this, put the plugin binary into a location of your choice. This directory will
be specified as the
[`plugin_directory`](https://github.com/openbao/openbao/blob/development/website/content/docs/configuration/index.mdx)
in the OpenBao config used to start the server.

```
...
plugin_directory = "path/to/plugin/directory"
...
```

Start a OpenBao server with this config file:
```sh
$ bao server -config=path/to/config.json ...
...
```

Once the server is started, register the plugin in the OpenBao server's plugin
catalog:

```sh
$ bao plugin register \
        -sha256=<expected SHA256 Hex value of the plugin binary> \
        secret \
        openbao-plugin-secrets-nomad
```

Note you should generate a new sha256 checksum if you have made changes to the
plugin. Example using `sha256sum`:

```sh
$ sha256sum openbao-plugin-secrets-nomad 
1642208f51c221e1d1acecbd16dcfb6ea43909a150f35fff3fc233490b722d5e  openbao-plugin-secrets-nomad
```

Enable the plugin backend using the secrets enable plugin command:

```sh
$ bao secrets enable -path=nomad openbao-plugin-secrets-nomad
...

Successfully enabled 'openbao-plugin-secrets-nomad' at 'nomad'!
```

#### Tests

If you are developing this plugin and want to verify it is still
functioning (and you haven't broken anything else), we recommend
running the tests.

To run the tests, invoke `go test`:

```sh
$ go test ./nomad/...
```

You can also sfilter tests like so:

```sh
$ go test ./nomad/... --run=TestBackend_config_Bootstrap
```
