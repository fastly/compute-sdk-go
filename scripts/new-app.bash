#!/usr/bin/env bash

set -e
set -u
set -o pipefail

if [[ "$(go env GOOS)" != "darwin" ]]
then
	echo "only macOS is supported for now"
	exit 1
fi

if [ ! $(which fastly) ]
then
	echo "fastly CLI not found, install instructions: https://github.com/fastly/cli"
	exit 1
fi

if [ ! $(which tinygo) ]
then
	echo "tinygo not found, install via: brew install tinygo"
	exit 1
fi

TINYGO_VERSION="$(tinygo version | cut -d' ' -f3 | cut -d. -f2)"
if [[ (${TINYGO_VERSION} < 24) ]]
then
	echo "tinygo version 0.24.0+ required, upgrade via: brew upgrade tinygo"
	exit 1
fi

GO_BINARY="go1.18.3"
if [ ! $(which ${GO_BINARY}) ]
then
	echo "fetching ${GO_BINARY}"
	go install golang.org/dl/${GO_BINARY}@latest
	${GO_BINARY} download
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
TEMPLATE_DIR="${SCRIPT_DIR}/_app_template"
TEMP_DIR="$(mktemp -d)"
cp -r ${TEMPLATE_DIR} ${TEMP_DIR}
mv ${TEMP_DIR}/_app_template ${TEMP_DIR}/tinygo-example

cd ${TEMP_DIR}/tinygo-example
go mod init tinygo-example
go env -w GOPRIVATE=github.com/fastly
go get github.com/fastly/compute-sdk-go@latest

echo
echo "cd ${TEMP_DIR}/tinygo-example"
echo "make serve"
echo "make deploy"
echo
