#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# External data source script for obtaining an access token JWT from the Azure 
# Instance Metadata Service (IMDS) endpoint. The access token is used as the form
# of authentication for the Vault Azure Auth Method. For details, see
# - https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token#get-a-token-using-http
#- https://registry.terraform.io/providers/hashicorp/external/latest/docs/data-sources/data_source
# - https://developer.hashicorp.com/vault/api-docs/auth/azure#jwt
set -e

function error_exit() {
  echo "$1" 1>&2
  exit 1
}

function check_deps() {
  test -f "$(which jq)" || error_exit "Must install the jq command to continue"
}

function parse_input() {
  eval "$(jq -r '@sh "export VM_IP_ADDRESS=\(.vm_ip_address)"')"
  if [[ -z "${VM_IP_ADDRESS}" ]]; then error_exit "vm_ip_address not provided"; fi
}

function obtain_access_token() {
  ssh -o StrictHostKeyChecking=accept-new -l "adminuser" "${VM_IP_ADDRESS}" \
  "curl -s -G -H 'Metadata:true' -d 'api-version=2018-02-01' -d 'resource=https://management.azure.com/' http://169.254.169.254/metadata/identity/oauth2/token"
}

# Sleep for 10 seconds to allow Azure Instance Metadata Service
# to become reachable after the virtual machine boots.
sleep 10

check_deps
parse_input
obtain_access_token
