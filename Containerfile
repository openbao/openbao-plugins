# This is a template which is intended to be used with the Makefile in this
# repo.

FROM scratch
ARG TARGETOS
ARG TARGETARCH
LABEL org.opencontainers.image.source=https://github.com/openbao/openbao-plugins

COPY bin/openbao-plugin-${PLUGIN}_${TARGETOS}_${TARGETARCH}* openbao-plugin-${PLUGIN}

ENTRYPOINT ["/openbao-plugin-${PLUGIN}"]
