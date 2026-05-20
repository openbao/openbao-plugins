# Typesense Database Plugin

This is the official database plugin for integrating OpenBao with [Typesense](https://typesense.org/).

## Building

To build the plugin from source, run the following command from the root of the `openbao-plugins` repository:

```bash
make database-typesense
# Or, for a clean build:
make clean database-typesense
```

The compiled binary will be placed in the `bin/` directory, for example: `bin/openbao-plugin-database-typesense_linux_amd64_v1`.

## Testing

To run the integration tests for this plugin, you will need Docker running on your system, as the tests spin up a live Typesense container via `ory/dockertest/v3`.

```bash
cd database/typesense
go test -v ./...
```
