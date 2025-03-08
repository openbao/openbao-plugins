#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


TOOL=openbao-plugin-auth-azure
#
# This script builds the application from source for a platform.
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd "$DIR"

# Set build tags
BUILD_TAGS="${BUILD_TAGS}:-${TOOL}"

# Get the git commit
GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"

# Delete the old dir
echo "==> Removing old directory..."
rm -f bin/*
mkdir -p bin/


# Build!
echo "==> Building..."
go build \
    -ldflags "${LD_FLAGS}" \
    -o "bin/${TOOL}" \
    -tags="${BUILD_TAGS}" \
    ./cmd

# Done!
echo
echo "==> Results:"
ls -hl bin/
