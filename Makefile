.DEFAULT: test

test: test-go test-tinygo test-integration
.PHONY: test

# Override these with environment variables or directly on the make command line.
GO_FLAGS := -tags=fastlyinternaldebug,nofastlyhostcalls
GO_PACKAGES := ./...
GO_TEST_FLAGS := -v

test-go:
	go test $(GO_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-go

test-tinygo: TINYGO_TARGET := wasip1
test-tinygo:
	tinygo test -target=$(TINYGO_TARGET) $(GO_FLAGS) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-tinygo

# Integration uses viceroy and overrides the default values for a number of variables.
test-integration: GO_FLAGS := -tags=fastlyinternaldebug
test-integration: GO_PACKAGES := ./integration_tests/...
test-integration: TINYGO_TARGET := targets/fastly-compute-wasip1.json
test-integration: tools/viceroy
	GOARCH=wasm GOOS=wasip1 go test $(GO_FLAGS) -exec "viceroy run -C fastly.toml" $(GO_TEST_FLAGS) $(GO_PACKAGES)
	tinygo test $(GO_FLAGS) -target=$(TINYGO_TARGET) $(GO_TEST_FLAGS) $(GO_PACKAGES)
.PHONY: test-integration

tools:
	@mkdir -p tools

tools/viceroy: tools # Download latest version of Viceroy ./tools/viceroy; delete it if you'd like to upgrade
	@arch=$$(uname -m | sed 's/x86_64/amd64/'); \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		url=$$(curl -s https://api.github.com/repos/fastly/viceroy/releases/latest | jq --arg arch $$arch --arg os $$os -r '.assets[] | select((.name | contains($$arch)) and (.name | contains($$os))) | .browser_download_url'); \
		filename=$$(basename $$url); \
		curl -sSLO $$url && \
		tar -xzf $$filename --directory ./tools/ && \
		rm $$filename && \
		./tools/viceroy --version && \
		touch ./tools/viceroy

# Makes tools/viceroy available as an executable within thie Makefile's recipes.
PATH := $(PWD)/tools:$(PATH)

viceroy-update:
	@rm -f tools/viceroy
	@$(MAKE) tools/viceroy
.PHONY: viceroy-update
