#!/bin/bash -eu

source /build-common.sh

COMPILE_IN_DIRECTORY="cmd/dockersockproxy"
BINARY_NAME="dockersockproxy"

standardBuildProcess
