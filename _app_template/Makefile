GO ?= go1.17.6
GOROOT ?= $(shell (${GO} env GOROOT))
TINYGO ?= tinygo
FASTLY ?= fastly
GC ?= leaking
SCHEDULER ?= asyncify
NAME ?= tinygo-example
DESCRIPTION ?= "TinyGo Compute@Edge example"
AUTHOR ?= $(shell (${FASTLY} whoami | xargs echo))
SOURCE ?= $(shell (find . -name '*.go'))
WASM ?= bin/${NAME}.wasm
PACKAGE ?= pkg/${NAME}.tar.gz

$(WASM): $(SOURCE) Makefile bin
	env GOROOT=${GOROOT} ${TINYGO} build -target=wasi -wasm-abi=generic -gc=${GC} -scheduler=${SCHEDULER} ${ARGS} -o ${WASM} ./cmd/${NAME}

${PACKAGE}: pkg ${WASM} fastly.toml
	${FASTLY} compute pack --wasm-binary=${WASM}

fastly.toml:
	${FASTLY} compute init --force --language=other --name=${NAME} --description=${DESCRIPTION} --author=${AUTHOR}

bin:
	mkdir -p bin

pkg:
	mkdir -p pkg

.PHONY: serve
serve: ${WASM} fastly.toml
	${FASTLY} compute serve --file=${WASM} --skip-build

.PHONY: deploy
deploy: ${PACKAGE} fastly.toml
	${FASTLY} compute deploy --path=${PACKAGE}

.PHONY: clean
clean:
	rm -rf bin pkg
