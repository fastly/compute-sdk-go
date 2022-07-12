# compute-tinygo

Experimental Go SDK for building [Compute@Edge](https://www.fastly.com/products/edge-compute/serverless) applications with [TinyGo](https://tinygo.org/).

## Quick Start

The Fastly Developer Hub has a great [Quick Start guide for Go](https://developer.fastly.com/learning/compute/go/).

Alternatively, you can take a look at the [Go Starter Kit](https://github.com/fastly/compute-starter-kit-go-default).

You'll also want to take a look at our [Recommended Packages](#recommended-packages) section, as this can help with the sharp edges of the SDK, like JSON support.

## Installation

First, install TinyGo by following the [TinyGo Quick install guide](https://tinygo.org/getting-started/install/).

Then, you can install `compute-sdk-go` in your project by running:

`go get github.com/fastly/compute-sdk-go`

## Examples

Examples can be found in the [`examples`](./_examples) directory.

The Fastly Developer Hub has a collection of [common use cases in VCL ported to TinyGo](https://developer.fastly.com/learning/compute/migrate/). Which also acts as a great set of introductory examples of using TinyGo on Compute@Edge.

## API Reference

The API reference documentation can be found on [pkg.go.dev/github.com/fastly/compute-sdk-go](https://pkg.go.dev/github.com/fastly/compute-sdk-go).

## Logging

Logging can be done using a Fastly Compute@Edge Log Endpoint ([example](./_examples/logging-and-env/main.go)), or by using normal stdout like:

```
fmt.Printf("request received: %s\n", r.URL.String())
```

## Recommended Packages

TinyGo is still a new project, which has yet to get a version `1.0.0`. Therefore, the project is incomplete, but in its current state can still handle a lot of tasks on Compute@Edge. For example, [some languages features of Go are still missing](https://tinygo.org/docs/reference/lang-support/), such as Reflection support, which is used for things like parsing JSON using the Go standard library. To help with your adoption of `compute-tinygo`, here are some recommended packages to help with some of the current missing language features:

### JSON Parsing

* [valyala/fastjson](https://github.com/valyala/fastjson)
* [mailru/easyjson](https://github.com/mailru/easyjson)
* [buger/jsonparser](https://github.com/buger/jsonparser)

Additional context on JSON support in TinyGo can be found [here](https://github.com/tinygo-org/tinygo/issues/447)

## Changelog

The changelog can be found [here](./CHANGELOG.md).

## Security

If you find any security issues, see the [Fastly Security Reporting Page](https://www.fastly.com/security/report-security-issue) or send an email to: `security@fastly.com`

Note that communications related to security issues in Fastly-maintained OSS as described here are distinct from [Fastly security advisories](https://www.fastly.com/security-advisories).

## License

[Apache-2.0 WITH LLVM-exception](./LICENSE)
