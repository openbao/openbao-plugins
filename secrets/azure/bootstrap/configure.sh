#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


PLUGIN_DIR=$1
PLUGIN_NAME=$2
PLUGIN_PATH=$3

# Try to clean-up previous runs
vault secrets disable "${PLUGIN_PATH}"
vault plugin deregister "${PLUGIN_NAME}"
killall "${PLUGIN_NAME}"

# Give a bit of time for the binary file to be released so we can copy over it
sleep 3

# Copy the binary so text file is not busy when rebuilding & the plugin is registered
cp ./bin/"$PLUGIN_NAME" "$PLUGIN_DIR"

# Sets up the binary with local changes
vault plugin register \
    -sha256="$(shasum -a 256 "$PLUGIN_DIR"/"$PLUGIN_NAME" | awk '{print $1}')" \
    secret "${PLUGIN_NAME}"

vault secrets enable -path="${PLUGIN_PATH}" "${PLUGIN_NAME}"

# Write the azure secrets configuration
vault write "${PLUGIN_PATH}"/config \
    subscription_id="${AZURE_SUBSCRIPTION_ID}" \
    tenant_id="${AZURE_TENANT_ID}" \
    client_id="${AZURE_CLIENT_ID}" \
    client_secret="${AZURE_CLIENT_SECRET}"

# Write a role
vault write "${PLUGIN_PATH}"/roles/dev-role ttl="5m" azure_roles=-<<EOF
    [
        {
            "role_name": "Storage Blob Data Owner",
            "scope":  "/subscriptions/${AZURE_SUBSCRIPTION_ID}"
        }
    ]
EOF

# Read dynamic service principal
# vault read "${PLUGIN_PATH}"/creds/dev-role

# Rotate root credentials
# vault write -force "${PLUGIN_PATH}"/rotate-root