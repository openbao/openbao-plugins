# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

if which gdate &> /dev/null ; then
  function __date_cmd() {
    TZ=UTC gdate --rfc-3339=ns
  }
else
  function __date_cmd() {
    TZ=UTC date -Iseconds
  }
fi

function log() {
    local level="${2:-INFO}"
    echo "[$(__date_cmd) ${level} ${BATS_TEST_NAME:-unknown}] $1"
}

function logError() {
    log $1 'ERROR'
}

function logWarn() {
    log $1 'WARN'
}

function testAzureSecret() {
    local role_name="$1"
    local subscription_id="$2"
    local resource_group_name="$3"
    local vault_role_name="$4"
    local config_dir="$5"
    local engine_name="$6"
    local ttl=5

    log "Creating Azure secret, azure_role='${role_name}', vault_role='${vault_role_name}'"
    vault write "${engine_name}/roles/${vault_role_name}" ttl="${ttl}" azure_roles=-<<EOF
[
    {
        "role_name": "${role_name}",
        "scope":  "/subscriptions/${subscription_id}/resourceGroups/${resource_group_name}"
    }
]
EOF

    local secret_file="${config_dir}/secret-${vault_role_name}.json"
    vault read "${engine_name}/creds/${vault_role_name}" -format=json > ${secret_file}

    sp_id=$(cat "${secret_file}" | jq -er .data.client_id)
    log "Secret created successfully, azure_role='${role_name}', vault_role='${vault_role_name}', file=${secret_file}"

    if !assertSPExistence ; then
        return 1
    fi

    if !waitLeaseExpiration ${role_name} ${vault_role_name} ${sp_id} ${ttl}; then
        return 1
    fi

    log "Test completed successfully, azure_role='${role_name}', vault_role='${vault_role_name}'"
}

function waitLeaseExpiration() {
    local role_name="${1}"
    local vault_role_name="${2}"
    local sp_id="${3}"
    local ttl="${4}"
    log "Waiting for lease expiration, azure_role='${role_name}', vault_role='${vault_role_name}'"
    sleep ${ttl}
    local tries=0
    # wait for the service principal to expire and be removed by Vault - adds a 60 second buffer to the ttl.
    until ! az ad sp show --id "${sp_id}" &> /dev/null
    do
        if [[ "${tries}" -ge 60 ]]; then
            logError "Vault failed to remove service principal ${sp_id}, ttl=${ttl}"
            return 1
        fi
        ((++tries))
        sleep 1
    done
}

function assertSPExistence() {
    local sp_id="${1}"
    local found=''
    for n in {0..30}
    do
         if ! az ad sp show --id "${sp_id}" 1> /dev/null ; then
            logWarn "Failed to check service principal exists for ID ${sp_id}"
            sleep 1
            continue
         fi
         found=1
         break
    done

    if [ -z "${found}" ]; then
        logError "Expected SP ID '${sp_id}' not found in Azure"
        return 1
    fi
}

function execTerraform() {
    local config_dir="$1"
    pushd ${config_dir}/terraform >&2
    echo terraform ${@:2} >&2
    terraform ${@:2}
    popd >&2
}

function terraformInit() {
    local config_dir="$1"
    execTerraform ${config_dir} init
}

function terraformApply() {
    local config_dir="$1"
    execTerraform ${config_dir} apply -input=false -auto-approve ${@:2}
}

function terraformInitApply() {
    terraformInit $@
    terraformApply $@
}

function terraformOutput() {
    local config_dir="$1"
    execTerraform ${config_dir} output -json
}

function terraformDestroy() {
    local config_dir="$1"
    execTerraform ${config_dir} apply -input=false -auto-approve -destroy ${@:2}
}

function tfOutputLocalEnv() {
    local output_file="$1"
    tf_output=$(cat ${output_file}) || return $?
    client_id="$(echo ${tf_output} | jq -er .application_id.value)" || return $?
    client_secret="$(echo ${tf_output} | jq -er .application_password_value.value)" || return $?
    subscription_id="$(echo ${tf_output} | jq -er .subscription_id.value)" || return $?
    resource_group_name="$(echo ${tf_output} | jq -er .resource_group_name.value)" || return $?
    tenant_id="$(echo ${tf_output} | jq -er .tenant_id.value)" || return $?
    cat <<HERE
local client_id="${client_id}"
local client_secret="${client_secret}"
local subscription_id="${subscription_id}"
local resource_group_name="${resource_group_name}"
local tenant_id="${tenant_id}"
HERE
}
