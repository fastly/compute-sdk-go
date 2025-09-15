## Show this help message
.PHONY: help
help: help-awk

## Run all SDK tests [unit, integration, end-to-end] using Go and TinyGo
.PHONY: test
test: test-go test-tinygo

## Run all SDK tests using only Go
.PHONY: test-go
test-go: test-unit-go test-integration-go test-e2e-go

## Run all SDK tests using only TinyGo
.PHONY: test-tinygo
test-tinygo: test-unit-tinygo test-integration-tinygo test-e2e-tinygo

## Customize test runs by changing defaults for GO_TEST_FLAGS & GO_PACKAGES
.PHONY: defaults
defaults: defaults-go defaults-tinygo

## Build examples using Go and TinyGo
.PHONY: build-examples
build-examples: build-examples-go build-examples-tinygo

## Run only [unit] tests using Go and TinyGo
.PHONY: test-unit
test-unit: test-unit-go test-unit-tinygo

## Run only [integration] tests using Go and TinyGo
.PHONY: test-integration
test-integration: test-integration-go test-integration-tinygo

## Run only [end-to-end] tests using Go and TinyGo
.PHONY: test-e2e
test-e2e: test-e2e-go test-e2e-tinygo

.PHONY: defaults-go
defaults-go:
	@echo GO_TEST_FLAGS=$(GO_TEST_FLAGS)
	@echo GO_PACKAGES=$(GO_PACKAGES)

# := is immediate assignment, evaluated once
GO_TEST_FLAGS := -v
GO_PACKAGES   := ./...

.PHONY: defaults-tinygo
defaults-tinygo:
	@echo TINYGO_TEST_FLAGS=$(TINYGO_TEST_FLAGS)
	@echo TINYGO_PACKAGES=$(TINYGO_PACKAGES)

# TINYGO_ variables derive from the GO_ versions.
# = is recursive assignment, expanded at each use
TINYGO_TEST_FLAGS = $(GO_TEST_FLAGS)
TINYGO_PACKAGES   = $(GO_PACKAGES)
TINYGO_BUILD_TAGS = $(GO_BUILD_TAGS)

# With the defaults arranged, each type of test (unit, integration, end-to-end)
# needs specific modifications. Integration tests use `test -exec viceroy`,
# end-to-end tests use `test -exec serve.sh`.

# Change build tags for different test types:
test-unit-%:         GO_BUILD_TAGS := fastlyinternaldebug nofastlyhostcalls
test-integration-%:  GO_BUILD_TAGS := fastlyinternaldebug
test-e2e-%: 		 GO_BUILD_TAGS := fastlyinternaldebug
build-examples-%:    GO_BUILD_TAGS := fastlyinternaldebug

# Change package lists for different test types:
test-integration-%:  GO_PACKAGES   := ./integration_tests/...
test-e2e-%: 		 GO_PACKAGES   := ./end_to_end_tests/...

# Match tinygo targets for build tags and viceroy args:
test-%-go:       	   EXEC_ARGS := viceroy run -C fastly.toml
test-%-tinygo:   	   TINYGO_TARGET := ./targets/fastly-compute-wasip1.json
test-e2e-tinygo: 	   TINYGO_TARGET := ./targets/fastly-compute-wasip1-serve.json
build-examples-tinygo: TINYGO_TARGET ?= ./targets/fastly-compute-wasip1.json

# Allow `test -exec` and tinygo's emulator target to find `serve.sh`:
test-e2e-%: export PATH := $(PWD)/end_to_end_tests:$(PATH)

# GOFLAGS brings all the options together. They are used directly in the command
# lines in recipes below to avoid hiding them in the environment.
test-%-go:     GOFLAGS     = $(GO_TEST_FLAGS) -tags=$(subst $(space),$(comma),$(GO_BUILD_TAGS))
test-%-tinygo: TINYGOFLAGS = $(TINYGO_TEST_FLAGS) -target=$(TINYGO_TARGET) -tags=$(subst $(space),$(comma),$(TINYGO_BUILD_TAGS))

## Run only [unit] tests using Go
.PHONY: test-unit-go
test-unit-go:
	@echo >&2
	@echo ">> Running Go [unit] tests..." >&2
	go test $(GOFLAGS) $(GO_PACKAGES)

## Run only [unit] tests using TinyGo
.PHONY: test-unit-tinygo
test-unit-tinygo: viceroy
	@echo >&2
	@echo ">> Running TinyGo [unit] tests..." >&2
	tinygo test $(TINYGOFLAGS) $(TINYGO_PACKAGES)

## Run only [integration] tests using Go
.PHONY: test-integration-go
test-integration-go: viceroy
	@echo >&2
	@echo ">> Running Go [integration] tests..." >&2
	GOARCH=wasm GOOS=wasip1 go test -exec "$(EXEC_ARGS)" $(GOFLAGS) $(GO_PACKAGES)

## Run only [integration] tests using TinyGo
.PHONY: test-integration-tinygo
test-integration-tinygo: viceroy
	@echo >&2
	@echo ">> Running TinyGo [integration] tests..." >&2
	tinygo test $(TINYGOFLAGS) $(TINYGO_PACKAGES)

## Run only [end-to-end] tests using Go
.PHONY: test-e2e-go
test-e2e-go: viceroy
	@echo >&2
	@echo ">> Running Go [end-to-end] tests..." >&2
	GOARCH=wasm GOOS=wasip1 go test -exec "serve.sh $(EXEC_ARGS)" $(GOFLAGS) $(GO_PACKAGES)

## Run only [end-to-end] tests using TinyGo
.PHONY: test-e2e-tinygo
test-e2e-tinygo: viceroy
	@echo >&2
	@echo ">> Running TinyGo [end-to-end] tests..." >&2
	tinygo test $(TINYGOFLAGS) $(TINYGO_PACKAGES)

## Build examples using Go
# `go build` can accept multiple packages, proving that compilation succeeds,
# and throw away the result automatically.
.PHONY: build-examples-go $(GO_EXAMPLES)
build-examples-go:
	@echo >&2
	@echo ">> Building examples with Go..." >&2
	go build $(GOFLAGS) ./_examples/*/

## Build examples using TinyGo
# `tinygo build` doesn't accept multiple packages, so we use make wildcards.
EXAMPLE_DIRS := $(patsubst _examples/%/,%,$(filter %/, $(wildcard _examples/*/)))
TINYGO_EXAMPLES := $(addprefix build-examples-tinygo-,$(EXAMPLE_DIRS))
.PHONY: build-examples-tinygo $(TINYGO_EXAMPLES)
build-examples-tinygo:
	@echo >&2
	@echo ">> Building examples with TinyGo..." >&2
	@$(MAKE) -j4 $(TINYGO_EXAMPLES)
$(TINYGO_EXAMPLES): build-examples-tinygo-%:
	tinygo build $(TINYGOFLAGS) -o /dev/null ./_examples/$*/main.go

## Check for viceroy on path
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

empty :=
space := $(empty) $(empty)
comma := ,

.PHONY: help-awk
help-awk:
	@awk ' \
			/^[ \t]*##/ { \
					help=$$0; \
					sub(/^##[ \t]*/, "", help); \
			} \
			/^[a-zA-Z0-9_-]+:/ { \
					if (help) { \
							printf "\033[36m%-20s\033[0m %s\n", $$1, help; \
							help=""; \
					} \
			} \
	' $(MAKEFILE_LIST)
