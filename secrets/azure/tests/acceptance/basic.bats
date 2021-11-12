#!/usr/bin/env bats

# based off of the "Vault Ecosystem - Testing Best Practices" Confluence page.

REPO_ROOT="$(git rev-parse --show-toplevel)"
PLUGIN_NAME="${REPO_ROOT##*/}"
VAULT_IMAGE="${VAULT_IMAGE:-hashicorp/vault:1.9.0-rc1}"
CONTAINER_NAME=''
VAULT_TOKEN='root'

TESTS_OUT_DIR="$(mktemp -d /tmp/${PLUGIN_NAME}.XXXXXXXXX)"
TESTS_OUT_FILE="${TESTS_OUT_DIR}/output.log"

PLUGIN_TYPE=''
case ${PLUGIN_NAME} in
  *-secrets-*)
    PLUGIN_TYPE='secret'
    ;;
  *-auth-*)
    PLUGIN_TYPE='auth'
    ;;
  *)
    echo "could not determine plugin type from ${PLUGIN_NAME}" >&2
    exit 1
    ;;
esac

if [[ -z "${AZURE_TENANT_ID}" ]]; then
    echo "AZURE_TENANT_ID env var not set" >&2
    exit 1
fi


if [[ -n "${WITH_DEV_PLUGIN}" ]]; then
    PLUGIN=${REPO_ROOT}/bin/${PLUGIN_NAME}
    PLUGIN_SHA256="$(sha256sum ${PLUGIN} | cut -d ' ' -f 1)"
fi

setup(){
    { # Braces used to redirect all setup logs.
    # 1. Configure Vault.

    export CONFIG_DIR="$(mktemp -d ${TESTS_OUT_DIR}/test-XXXXXXX)"
    export CONTAINER_NAME="${CONFIG_DIR##*/}"
    local PLUGIN_DIR="${CONFIG_DIR}/plugins"
    mkdir -vp ${CONFIG_DIR}/{terraform,plugins}
    echo "plugin_directory = \"/vault/config/plugins\"" > ${CONFIG_DIR}/vault.hcl

    cp -a ${REPO_ROOT}/tests/acceptance/terraform/*.tf ${CONFIG_DIR}/terraform/.
    cat > ${CONFIG_DIR}/terraform/terraform.tfvars <<HERE
tenant_id = "${AZURE_TENANT_ID}"
name_prefix = "${PLUGIN_NAME}"
HERE

    docker pull ${VAULT_IMAGE?}
    docker run \
      --name="${CONTAINER_NAME}" \
      --hostname=vault \
      -p 0:8200 \
      -v "${CONFIG_DIR}:/vault/config" \
      -e VAULT_DEV_ROOT_TOKEN_ID="${VAULT_TOKEN?}" \
      -e VAULT_ADDR="http://localhost:8200" \
      -e VAULT_DEV_LISTEN_ADDRESS="0.0.0.0:8200" \
      --privileged \
      --detach \
      ${VAULT_IMAGE?}

    local tries=0
    until [ $tries -ge 30 ]
    do
        ((++tries))
        HOST_PORT="$(docker inspect ${CONTAINER_NAME} | \
            jq -er '.[0].NetworkSettings.Ports."8200/tcp"[0].HostPort')"

        if nc -z localhost ${HOST_PORT} ; then
            export VAULT_ADDR="http://localhost:${HOST_PORT?}"
            vault login ${VAULT_TOKEN?} || continue
            break
        fi

        sleep .5
    done

    [ -z "${VAULT_ADDR}" ] && exit 1

    # enable auditing
    vault audit enable file file_path=stdout

    if [[ -n "${WITH_DEV_PLUGIN}" ]]; then
        cp -a ${PLUGIN} ${CONFIG_DIR}/plugins/.
        # replace the builtin plugin with a local build
        vault plugin register -sha256="${PLUGIN_SHA256}" ${PLUGIN_TYPE} ${PLUGIN_NAME}
        vault plugin reload -plugin=${PLUGIN_NAME}
    fi

    } >> $TESTS_OUT_FILE
}

teardown(){
    if [[ -n $SKIP_TEARDOWN ]]; then
        echo "Skipping teardown"
        return
    fi

    { # Braces used to redirect all teardown logs.

    # If the test failed, print some debug output
    if [[ "$BATS_ERROR_STATUS" -ne 0 ]]; then
        docker logs "${CONTAINER_NAME?}"
    fi

    # Teardown Vault configuration.
    docker rm --force "${CONTAINER_NAME}"

    printenv | sort

    pushd ${CONFIG_DIR}/terraform
    terraform apply -destroy -input=false -auto-approve
    popd

    rm -rf "${CONFIG_DIR}"

    } >> $TESTS_OUT_FILE
}

@test "Azure Secrets Engine - Legacy AAD" {
    pushd ${CONFIG_DIR}/terraform
    terraform init && terraform apply -input=false -auto-approve -var legacy_aad_resource_access=true
    local tf_output=$(terraform output -json | tee ${CONFIG_DIR}/tf-output.json)
    popd

    # TODO: remove this sleep, tests periodically fail if the credentials created during infrastructure
    # provisioning are not considered valid by Azure. Need to find a way to poll for the creds status.
    sleep 10

    local client_id="$(echo ${tf_output} | jq -er .application_id.value)"
    local client_secret="$(echo ${tf_output} | jq -er .application_password_value.value)"
    local subscription_id="$(echo ${tf_output} | jq -er .subscription_id.value)"
    local resource_group_name="$(echo ${tf_output} | jq -er .resource_group_name.value)"
    local tenant_id="$(echo ${tf_output} | jq -er .tenant_id.value)"

    vault secrets enable azure

    vault write azure/config \
        subscription_id=${subscription_id} \
        tenant_id="${tenant_id}" \
        client_id="${client_id}" \
        client_secret="${client_secret}"

    local ttl=10
    vault write azure/roles/my-role ttl="${ttl}" azure_roles=-<<EOF
[
    {
        "role_name": "Reader",
        "scope":  "/subscriptions/${subscription_id}/resourceGroups/${resource_group_name}"
    }
]
EOF
    local secret="$(vault read azure/creds/my-role -format=json)"
    local sp_id="$(echo ${secret} | jq -er .data.client_id)"
    local sp="$(az ad sp show --id "${sp_id}")"
    echo ${secret} | jq
    echo ${sp} | jq

    sleep ${ttl}
    local tries=0
    # wait for the service principal to expire and be removed by Vault - adds a 5 second buffer.
    until ! az ad sp show --id "${sp_id}" > /dev/null
    do
        if [[ "${tries}" -ge 10 ]]; then
            echo "vault failed to remove service principal ${sp_id}, ttl=${ttl}" >&2
            exit 1
        fi
        ((++tries))
        sleep .5
    done

} >> $TESTS_OUT_FILE

@test "Azure Secrets Engine - MS Graph" {
    pushd ${CONFIG_DIR}/terraform
    terraform init && terraform apply -input=false -auto-approve -var legacy_aad_resource_access=false
    local tf_output=$(terraform output -json | tee ${CONFIG_DIR}/tf-output.json)
    popd

    # TODO: remove this sleep, tests periodically fail if the credentials created during infrastructure
    # provisioning are not considered valid by Azure. Need to find a way to poll for the creds status.
    sleep 10

    local client_id="$(echo ${tf_output} | jq -er .application_id.value)"
    local client_secret="$(echo ${tf_output} | jq -er .application_password_value.value)"
    local subscription_id="$(echo ${tf_output} | jq -er .subscription_id.value)"
    local resource_group_name="$(echo ${tf_output} | jq -er .resource_group_name.value)"
    local tenant_id="$(echo ${tf_output} | jq -er .tenant_id.value)"

    vault secrets enable azure

    vault write azure/config \
        use_microsoft_graph_api=true \
        subscription_id="${subscription_id}" \
        tenant_id="${tenant_id}" \
        client_id="${client_id}" \
        client_secret="${client_secret}"

    local ttl=10
    vault write azure/roles/my-role ttl="${ttl}" azure_roles=-<<EOF
[
    {
        "role_name": "Reader",
        "scope":  "/subscriptions/${subscription_id}/resourceGroups/${resource_group_name}"
    }
]
EOF
    local secret="$(vault read azure/creds/my-role -format=json)"
    local sp_id="$(echo ${secret} | jq -er .data.client_id)"
    local sp="$(az ad sp show --id "${sp_id}")"
    echo ${secret} | jq
    echo ${sp} | jq

    sleep ${ttl}
    local tries=0
    # wait for the service principal to expire and be removed by Vault - adds a 5 second buffer.
    until ! az ad sp show --id "${sp_id}" > /dev/null
    do
        if [[ "${tries}" -ge 10 ]]; then
            echo "vault failed to remove service principal ${sp_id}, ttl=${ttl}" >&2
            exit 1
        fi
        ((++tries))
        sleep .5
    done

} >> $TESTS_OUT_FILE
