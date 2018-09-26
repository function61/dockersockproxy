#!/bin/bash -eu

# static link so we can use empty container image
CGO_ENABLED=0 go build -o dockersockproxy -ldflags "-extldflags \"-static\""
