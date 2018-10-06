#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="dockersockproxy"
COMPILE_IN_DIRECTORY="cmd/dockersockproxy"

standardBuildProcess
