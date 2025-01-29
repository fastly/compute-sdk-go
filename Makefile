.DEFAULT: test

test: test-go test-tinygo test-integration
.PHONY: test

# Makes tools/viceroy available as an executable within Makefile recipes.
PATH := $(PWD)/tools:$(PATH)

# Override these with environment variables or directly on the make command line.
GO_BUILD_FLAGS := -tags=fastlyinternaldebug,nofastlyhostcalls
GO_TEST_FLAGS  := -v
GO_PACKAGES    := ./...

test-go:
	go test $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-go

# Using this target lets viceroy provide the wasm runtime, eliminating a dependency on wasmtime.
TINYGO_TARGET := ./targets/fastly-compute-wasip1.json

test-tinygo:
	tinygo test -target=$(TINYGO_TARGET) $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-tinygo

# Integration tests use viceroy and override the default values for these variables.
test-integration-%: GO_BUILD_FLAGS := -tags=fastlyinternaldebug
test-integration-%: GO_PACKAGES    := ./integration_tests/...

test-integration: test-integration-go test-integration-tinygo
.PHONY: test-integration

test-integration-go: tools/viceroy
	GOARCH=wasm GOOS=wasip1 go test -exec "viceroy run -C fastly.toml" $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-integration-go

test-integration-tinygo: tools/viceroy
	tinygo test -target=$(TINYGO_TARGET) $(GO_BUILD_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-integration-tinygo

tools/viceroy: | tools # Download latest version of Viceroy ./tools/viceroy; delete it if you'd like to upgrade
	@arch=$$(uname -m | sed 's/x86_64/amd64/'); \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		url=$$(curl -s https://api.github.com/repos/fastly/viceroy/releases/latest | jq --arg arch $$arch --arg os $$os -r '.assets[] | select((.name | contains($$arch)) and (.name | contains($$os))) | .browser_download_url'); \
		filename=$$(basename $$url); \
		curl -sSLO $$url && \
		tar -xzf $$filename --directory ./tools/ && \
		rm $$filename && \
		./tools/viceroy --version && \
		touch ./tools/viceroy
ifneq ($(strip $(GITHUB_PATH)),)
	@echo "$(PWD)/tools" >> "$(GITHUB_PATH)"
endif

tools:
	@mkdir -p tools

viceroy-update:
	@rm -f tools/viceroy
	@$(MAKE) tools/viceroy
.PHONY: viceroy-update
