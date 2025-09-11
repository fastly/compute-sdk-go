.DEFAULT: test

.PHONY: test
test: test-go test-tinygo test-integration test-e2e

# Override these with environment variables or directly on the make command line.
GO_BUILD_FLAGS := -tags=fastlyinternaldebug,nofastlyhostcalls
GO_TEST_FLAGS  := -v
GO_PACKAGES    := ./...

.PHONY: test-go
test-go:
	@echo ">> Running Go tests..." >&2
	go test $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

# Using this target lets viceroy provide the wasm runtime, eliminating a dependency on wasmtime.
TINYGO_TARGET := ./targets/fastly-compute-wasip1.json

.PHONY: test-tinygo
test-tinygo: viceroy
	@echo ">> Running TinyGo tests..." >&2
	tinygo test -target=$(TINYGO_TARGET) $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

# Integration tests use viceroy and override the default values for these variables.
test-integration-%: GO_BUILD_FLAGS := -tags=fastlyinternaldebug
test-integration-%: GO_PACKAGES    := ./integration_tests/...


.PHONY: test-integration
test-integration: test-integration-go test-integration-tinygo

.PHONY: test-integration-go
test-integration-go: viceroy
	@echo ">> Running Go integration tests..." >&2
	GOARCH=wasm GOOS=wasip1 go test -exec "viceroy run -C fastly.toml" $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

.PHONY: test-integration-tinygo
test-integration-tinygo: viceroy
	@echo ">> Running TinyGo integration tests..." >&2
	tinygo test -target=$(TINYGO_TARGET) $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

# End to end tests use serve.sh and override the default values for these variables.
test-e2e-%: GO_BUILD_FLAGS := -tags=fastlyinternaldebug
test-e2e-%: GO_PACKAGES    := ./end_to_end_tests/...
test-e2e-%: export PATH := $(PWD)/end_to_end_tests:$(PATH) # allows go test to find serve.sh

.PHONY: test-e2e
test-e2e: test-e2e-go test-e2e-tinygo

.PHONY: test-e2e-go
test-e2e-go: viceroy
	@echo ">> Running Go end-to-end tests..." >&2
	GOARCH=wasm GOOS=wasip1 go test -exec "serve.sh viceroy run -C fastly.toml" $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

.PHONY: test-e2e-tinygo
test-e2e-tinygo: TINYGO_TARGET := ./targets/fastly-compute-wasip1-serve.json
test-e2e-tinygo: viceroy
	@echo ">> Running TinyGo end-to-end tests..." >&2
	tinygo test -target=$(TINYGO_TARGET) $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)

.PHONY: viceroy
viceroy:
	@which viceroy || ( \
	    echo "viceroy not found: please ensure it is installed and available in your PATH:" && \
		echo $$PATH && \
		echo && \
		echo "The fastly CLI installs Viceroy in the fastly subdirectory of the path returned by" && \
		echo "os.UserConfigDir():" && \
		echo "  > On Unix systems, it returns \$$XDG_CONFIG_HOME if non-empty, else \$$HOME/.config." && \
		echo "  > On Darwin, it returns \$$HOME/Library/Application Support." && \
		echo "From https://pkg.go.dev/os#UserConfigDir" && \
		exit 1 \
	)
