#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


PLUGIN_DIR=$1
PLUGIN_NAME=$2
PLUGIN_PATH=$3

# Try to clean-up previous runs
bao auth disable "${PLUGIN_PATH}"
bao plugin deregister "${PLUGIN_NAME}"
killall "${PLUGIN_NAME}"

# Give a bit of time for the binary file to be released so we can copy over it
sleep 3

# Copy the binary so text file is not busy when rebuilding & the plugin is registered
cp ./bin/"$PLUGIN_NAME" "$PLUGIN_DIR"

# Sets up the binary with local changes
bao plugin register \
    -sha256="$(shasum -a 256 "$PLUGIN_DIR"/"$PLUGIN_NAME" | awk '{print $1}')" \
    auth "${PLUGIN_NAME}"

bao auth enable -path="${PLUGIN_PATH}" "${PLUGIN_NAME}"

# Write the azure auth configuration
bao write auth/"${PLUGIN_PATH}"/config \
    tenant_id="${TENANT_ID}" \
    client_id="${CLIENT_ID}" \
    client_secret="${CLIENT_SECRET}" \
    resource="https://management.azure.com/"

# Write a role with some bound constraints
bao write auth/"${PLUGIN_PATH}"/role/dev-role \
    bound_subscription_ids="${SUBSCRIPTION_ID}" \
    bound_resource_groups="${RESOURCE_GROUP_NAME}"

# Login using the access token and vm name
bao write auth/"${PLUGIN_PATH}"/login \
    role="dev-role" \
    jwt="${ACCESS_TOKEN_JWT}" \
    subscription_id="${SUBSCRIPTION_ID}" \
    resource_group_name="${RESOURCE_GROUP_NAME}" \
    vm_name="${VM_NAME}"

# Login using the access token and resource ID
# bao write auth/"${PLUGIN_PATH}"/login \
#     role="dev-role" \
#     jwt="${ACCESS_TOKEN_JWT}" \
#     subscription_id="${SUBSCRIPTION_ID}" \
#     resource_group_name="${RESOURCE_GROUP_NAME}" \
#     resource_id="${RESOURCE_ID}"
