#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e

export GRPC_GO_LOG_VERBOSITY_LEVEL=2
export GRPC_GO_LOG_SEVERITY_LEVEL=info

pkill vault || true

make dev
mkdir -p bin/
cp "$GOPATH/bin/vault-plugin-secrets-gcpkms" bin/

vault server \
  -log-level=warn \
  -dev \
  -dev-plugin-dir="$(pwd)/bin" &
VAULT_PID=$!
sleep 2

vault secrets enable -path=gcpkms -plugin-name=vault-plugin-secrets-gcpkms plugin
