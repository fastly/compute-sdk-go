# compute-sdk-go

Go SDK for building [Fastly Compute](https://www.fastly.com/products/edge-compute) applications with [Go](https://go.dev) (1.21+) and [TinyGo](https://tinygo.org/) (0.28.1+).

## Quick Start

The Fastly Developer Hub has a great [Quick Start guide for Go](https://developer.fastly.com/learning/compute/go/).

Alternatively, you can take a look at the [Go Starter Kit](https://github.com/fastly/compute-starter-kit-go-default).

If you're using TinyGo, you'll also want to take a look at our [TinyGo Recommended Packages](#tinygo-recommended-packages) section, as this can help with the sharp edges of the SDK, like JSON support.

## Supported Toolchains

Compute builds on top of WebAssembly and the [WebAssembly System Interface](https://wasi.dev/).

TinyGo supports WASI as a target, and Go does as of its 1.21 release.

Each toolchain has its own tradeoffs.  Generally speaking, TinyGo produces smaller compiled artifacts and takes less RAM at runtime.  Build times are generally longer, sometimes considerably.  TinyGo does not support all of the Go standard library, and in particular support for the `reflect` package is incomplete.  This means that some third-party packages may not work with TinyGo.

Runtime performance is mixed, with TinyGo faster on some applications and Go faster on others.  If you have a performance-critical application, we recommend benchmarking both toolchains to see which works best for you.

To switch between TinyGo and Go, set the `build` command in the `[scripts]` section of your `fastly.toml` as follows:

    [scripts]
    build = "tinygo build -target=wasi -o bin/main.wasm ."

or

    [scripts]
    build = "GOARCH=wasm GOOS=wasip1 go build -o bin/main.wasm ."

You might need to adjust the actual build command depending on your project.

## Installation

If you're using Go, download [the latest Go release](https://go.dev/dl/). For TinyGo, follow the [TinyGo Quick install guide](https://tinygo.org/getting-started/install/).

Then, you can install `compute-sdk-go` in your project by running:

`go get github.com/fastly/compute-sdk-go`

## Examples

Examples can be found in the [`examples`](./_examples) directory.

The Fastly Developer Hub has a collection of [common use cases in VCL ported to Go](https://developer.fastly.com/learning/compute/migrate/) which also acts as a great set of introductory examples of using Go on Compute.

## API Reference

The API reference documentation can be found on [pkg.go.dev/github.com/fastly/compute-sdk-go](https://pkg.go.dev/github.com/fastly/compute-sdk-go).

## Testing

Tests that rely on a Compute runtime use [Viceroy](https://github.com/fastly/Viceroy), our local development tool.

The `Makefile` installs viceroy in ./tools/ and uses this version to run tests.

Write your tests as ordinary Go tests.  Viceroy provides the Compute APIs locally, although be aware that not all platform functionality is available.  You can look at the `integration_tests` directory for examples.

To run your tests:

    make test

This target runs tests in both Go and TinyGo, and `integration_tests` in both Go and TinyGo in Viceroy.  See additional targets in `Makefile` for running subsets of these tests.

The `targets/fastly-compute-wasi{,p1}.json` files provide TinyGo targets to run Viceroy.

## Logging

Logging can be done using a Fastly Compute Log Endpoint ([example](./_examples/logging-and-env/main.go)), or by using normal stdout like:

```
fmt.Printf("request received: %s\n", r.URL.String())
```

## Readthrough HTTP Cache Support

Customizing cache behaviour with the readthrough cache is an opt-in feature; enable it by adding `-tags=fsthttp_guest_cache` to the build line of your `fastly.toml`.

```
[scripts]
build = "tinygo build -target=wasip1 -tags=fsthttp_guest_cache -o bin/main.wasm ."
```

## TinyGo Recommended Packages

TinyGo is still a new project, which has yet to get a version `1.0.0`. Therefore, the project is incomplete, but in its current state can still handle a lot of tasks on Compute. However, [some languages features of Go are still missing](https://tinygo.org/docs/reference/lang-support/).

To help with your adoption of `compute-sdk-go`, here are some recommended packages to help with some of the current missing language features:

### JSON Parsing

TinyGo's  `reflect` support (which is needed by `encoding/json` among other things) is still new. While most use cases should work, for performance or other compatibility reasons you might need to consider a third-party JSON package if the standard library doesn't meet your needs.

* [valyala/fastjson](https://github.com/valyala/fastjson)
* [mailru/easyjson](https://github.com/mailru/easyjson)
* [buger/jsonparser](https://github.com/buger/jsonparser)

## Changelog

The changelog can be found [here](./CHANGELOG.md).

## Security

If you find any security issues, see the [Fastly Security Reporting Page](https://www.fastly.com/security/report-security-issue) or send an email to: `security@fastly.com`

Note that communications related to security issues in Fastly-maintained OSS as described here are distinct from [Fastly security advisories](https://www.fastly.com/security-advisories).

## License

[Apache-2.0 WITH LLVM-exception](./LICENSE)
