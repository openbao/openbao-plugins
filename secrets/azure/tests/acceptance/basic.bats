#!/usr/bin/env bats

load common.sh

# based off of the "Vault Ecosystem - Testing Best Practices" Confluence page.

REPO_ROOT="$(git rev-parse --show-toplevel)"
PLUGIN_NAME="${REPO_ROOT##*/}"
VAULT_IMAGE="${VAULT_IMAGE:-hashicorp/vault:1.12.1}"
CONTAINER_NAME=''
VAULT_TOKEN='root'

TESTS_OUT_DIR="$(mktemp -d /tmp/${PLUGIN_NAME}.XXXXXXXXX)"
TESTS_OUT_FILE="${TESTS_OUT_FILE:-${TESTS_OUT_DIR}/output.log}"

PLUGIN_TYPE=''
case ${PLUGIN_NAME} in
  *-secrets-*)
    PLUGIN_TYPE='secret'
    # short name e.g. `azure`
    ENGINE_NAME="${PLUGIN_NAME##*-secrets-}"
    ;;
  *-auth-*)
    PLUGIN_TYPE='auth'
    # short name e.g. `azure`
    ENGINE_NAME="${PLUGIN_NAME##*-auth-}"
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
    PLUGIN=${REPO_ROOT}/pkg/linux_amd64/${PLUGIN_NAME}
    PLUGIN_SHA256="$(sha256sum ${PLUGIN} | cut -d ' ' -f 1)" || exit 1
fi

setup(){
    { # Braces used to redirect all setup logs.
    # 1. Configure Vault.

    log "SetUp"

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
      -e VAULT_LICENSE="${VAULT_LICENSE}" \
      --privileged \
      --detach \
      ${VAULT_IMAGE?}

    local tries=0
    until [ $tries -ge 30 ]
    do
        ((++tries))
        HOST_PORT="$(docker inspect ${CONTAINER_NAME} | \
            jq -er '.[0].NetworkSettings.Ports."8200/tcp"[0].HostPort')"

        if nc -z localhost ${HOST_PORT} &> /dev/null ; then
            export VAULT_ADDR="http://localhost:${HOST_PORT?}"
            vault login ${VAULT_TOKEN?} &> /dev/null || continue
            break
        fi

        sleep .5
    done

    [ -z "${VAULT_ADDR}" ] && exit 1

    # enable auditing
    vault audit enable file file_path=stdout

    if [[ -n "${WITH_DEV_PLUGIN}" ]]; then
        log "Registering vault plugin"
        cp -a ${PLUGIN} ${CONFIG_DIR}/plugins/.
        # replace the builtin plugin with a local build
        vault plugin register -sha256="${PLUGIN_SHA256}" -command=${PLUGIN_NAME} ${PLUGIN_TYPE} ${ENGINE_NAME}
        vault plugin reload -plugin=${ENGINE_NAME}
    fi

    log "SetUp successful"
    } >> $TESTS_OUT_FILE
}

teardown(){
    log "TearDown"

    if [[ -n $SKIP_TEARDOWN ]]; then
        logWarn "Skipping teardown"
        logWarn "Clean up required, please run '(cd ${CONFIG_DIR}/terraform && terraform apply -destroy)'"
        logWarn "See ${TESTS_OUT_FILE} for more details"
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

    terraformDestroy ${CONFIG_DIR}

    rm -rf "${CONFIG_DIR}"

    echo "See ${TESTS_OUT_FILE} for more details" >&2
    log "TearDown successful"

    } >> $TESTS_OUT_FILE
}

@test "Azure Secrets Engine - MS Graph" {
    local tf_output_file=${CONFIG_DIR}/tf-output.json
    terraformInitApply ${CONFIG_DIR}
    terraformOutput ${CONFIG_DIR} > ${tf_output_file}

    tfOutputLocalEnv ${tf_output_file} > ${CONFIG_DIR}/local.env
    . ${CONFIG_DIR}/local.env
    local >&2

    vault secrets enable ${ENGINE_NAME}
    vault write "${ENGINE_NAME}/config" \
        subscription_id="${subscription_id}" \
        tenant_id="${tenant_id}" \
        client_id="${client_id}" \
        client_secret="${client_secret}"

    # Azure API access provisioning seems to be delayed for whatever reason, so sleep a bit.
    sleep 30

    local roles=('Reader' 'Storage Blob Data Owner')
    for ((i=0; i < ${#roles[@]}; i++)); do
        testAzureSecret "${roles[$i]}" ${subscription_id} ${resource_group_name} "role-${i}" ${CONFIG_DIR} ${ENGINE_NAME}
    done
} >> $TESTS_OUT_FILE
