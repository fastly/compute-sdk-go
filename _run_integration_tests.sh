#!/bin/bash

set -e

echo "Running act to test integration tests with:"
echo "[Optional] Fastly Token: $1"
act -j sdktest-go-latest --secret fastly_token=$1
