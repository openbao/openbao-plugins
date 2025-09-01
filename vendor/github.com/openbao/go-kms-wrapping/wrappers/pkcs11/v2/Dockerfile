# syntax=docker/dockerfile:1
# check=skip=SecretsUsedInArgOrEnv

FROM docker.io/golang:latest

# Install required packages:
RUN apt update && \
	DEBIAN_FRONTEND="noninteractive" apt install -y --no-install-recommends softhsm2 opensc

# Set SoftHSM to output debug logs:
RUN sed -i 's/log.level = ERROR/log.level = DEBUG/' /etc/softhsm/softhsm2.conf

ENV BAO_HSM_LIB=/usr/lib/softhsm/libsofthsm2.so

ENV SO_PIN=1234
ENV BAO_HSM_PIN=4321
ENV BAO_HSM_TOKEN_LABEL=openbao/go-kms-wrapping

ENV AES_KEY_LABEL=bao-aes
ENV RSA_KEY_LABEL=bao-rsa

RUN <<EOF
	set -e

	# Initialize a slot, --slot 0 will allocate some random slot, not 0!
	softhsm2-util --init-token --slot 0 --label "$BAO_HSM_TOKEN_LABEL" --so-pin "$SO_PIN" --pin "$BAO_HSM_PIN"

	# Create our keys:
	pkcs11-tool --module "$BAO_HSM_LIB" --token-label "$BAO_HSM_TOKEN_LABEL" --pin "$BAO_HSM_PIN" \
		--keygen --key-type aes:32 --label "$AES_KEY_LABEL"
	pkcs11-tool --module "$BAO_HSM_LIB" --token-label "$BAO_HSM_TOKEN_LABEL" --pin "$BAO_HSM_PIN" \
		--keypairgen --key-type rsa:4096 --label "$RSA_KEY_LABEL"
EOF

WORKDIR /go-kms-wrapping
COPY ./go.mod ./go.sum ./

WORKDIR /go-kms-wrapping/wrappers/pkcs11
COPY ./wrappers/pkcs11/go.mod ./wrappers/pkcs11/go.sum ./

RUN go mod download

COPY . /go-kms-wrapping

ENV KMS_ACC_TESTS=true

# SoftHSM only supports SHA1.
ENV BAO_HSM_RSA_OAEP_HASH=SHA1

RUN <<EOF
	set -e

	# RSA
	export BAO_HSM_KEY_LABEL=$RSA_KEY_LABEL
	go clean -testcache && go test

	# RSA with hardware encryption
	go clean -testcache && BAO_HSM_DISABLE_SOFTWARE_ENCRYPTION=1 go test

	# AES
	export BAO_HSM_KEY_LABEL=$AES_KEY_LABEL
	go clean -testcache && go test

	# AES via slot number instead of token label
	slot="$(softhsm2-util --show-slots | grep '^Slot [0-9]*$' | awk '{ print $2 }' | sort -n | tail -n 1)"
	go clean -testcache && BAO_HSM_TOKEN_LABEL="" BAO_HSM_SLOT=$slot go test
EOF
