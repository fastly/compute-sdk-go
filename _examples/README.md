# Go SDK Examples

This directory contains examples of using the Go SDK.
You can run them locally in Viceroy by running `fastly compute serve` from the individual example directory.

#### [`add-or-remove-cookies`](./add-or-remove-cookies)

This example shows how to read cookies from requests and how to add or remove cookies from responses.

#### [`corecache`](./corecache)

This example demonstrates the [Core Cache API](https://pkg.go.dev/github.com/fastly/compute-sdk-go/cache/core).

Resources are specified by the URL path.  A `GET` will retrieve a value from the cache, a `POST` will insert the value into the cache, and a `DELETE` will purge the value from the cache.

Note that this example isn't currently supported by Viceroy and has to be deployed to a Fastly service, which you can do with `fastly compute build` and `fastly compute deploy`.
If you haven't previously deployed it, the CLI will walk you through the steps of creating a new service for it.

#### [`geodata`](./geodata)

A simple example demonstrating what's available from the [geographic data API](https://pkg.go.dev/github.com/fastly/compute-sdk-go/geo) for IP addresses.

Note that Viceroy serves canned data for this API, but it can be customized in the `fastly.toml` file.
See the [documentation](https://developer.fastly.com/reference/compute/fastly-toml/#geolocation) for details.

#### [`hello-world`](./hello-world)

The canonical example.  It demonstrates the basic structure of a Fastly Compute program in Go.

#### [`http-adapter`](./http-adapter)

The Go SDK provides a set of APIs in its [`fsthttp`](https://pkg.go.dev/github.com/fastly/compute-sdk-go/fsthttp) package which should be familiar to any Go programmer who has used the standard library's [`net/http`](https://pkg.go.dev/net/http) package.
Although similar, the implementations are different and the types are not interchangeable.

[An adapter](https://pkg.go.dev/github.com/fastly/compute-sdk-go/fsthttp#Adapt) is provided to allow existing `net/http` handlers to be used with the Fastly Compute API.
This example demonstrates how to use it with [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux) from the standard library but it can be used with any other [`http.Handler`](https://pkg.go.dev/net/http#Handler), including ones from popular third-party libraries like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi).

#### [`kvstore`](./kvstore)

An example demonstrating reading from the [Fastly KV Store API](https://pkg.go.dev/github.com/fastly/compute-sdk-go/kvstore).

The KV store is populated locally by adding values to the `fastly.toml` file.
See the [documentation](https://developer.fastly.com/reference/compute/fastly-toml/#kv-and-secret-stores) for details.

#### [`limits`](./limits)

This example demonstrates how to adjust certain [request limits](https://pkg.go.dev/github.com/fastly/compute-sdk-go/fsthttp#Limits) before handling requests.  This example increases the maximum URL length from the default 8k to 16k.

#### [`logging-and-env`](./logging-and-env)

This example demonstrates how logging works, both by printing to stdout and stderr and by sending to dedicated logging endpoints using the [`rtlog`](https://pkg.go.dev/github.com/fastly/compute-sdk-go/rtlog) package.
It also prints out several environment variables that are set by default in the Compute environment.
Note, however, that most of these environment variables are unset when running locally.

#### [`middlewares`](./middlewares)

A demonstration of how to build middleware on top of `fsthttp.Handler` just as you would for `net/http.Handler`.

#### [`multiple-goroutines`](./multiple-goroutines)

This example shows how your handlers can run multiple concurrently-running goroutines.

#### [`parallel-requests`](./parallel-requests)

Building on the previous example, this one demonstrates how to make multiple origin requests in parallel.
Each request executes in its own goroutine.
The contents of the responses are discarded, but the overall request should finish in about as long as the longest individual origin request took.

#### [`print-request`](./print-request)

Returns a response containing information about the incoming request.

#### [`proxy-request`](./proxy-request)

This example demonstrates how to act as a simple proxy, forwarding the incoming request to an origin server and returning the response to the client.

#### [`proxy-request-framing`](./proxy-request-framing)

A variation on the previous example, this one demonstrates how you can manually override the framing modes of requests and responses.
This gives you control over `Content-Length` and `Transfer-Encoding` headers, which are normally handled automatically by the runtime.

#### [`secret-store`](./secret-store)

This example demonstrates the [Fastly Secret Store API](https://pkg.go.dev/github.com/fastly/compute-sdk-go/secretstore).
A secret store is opened, a secret is looked up, and the value is decrypted and returned to the client.

The secret store is populated locally by adding values to the `fastly.toml` file.
Note that values in the `fastly.toml` file are not encrypted.
See the [documentation](https://developer.fastly.com/reference/compute/fastly-toml/#kv-and-secret-stores) for details.

#### [`set-cookie`](./set-cookie)

A simple example that sets a cookie on the response.

#### [`set-google-analytics`](./set-google-analytics)

Due to [Intelligent Tracking Prevention 2.1](https://webkit.org/blog/8613/intelligent-tracking-prevention-2-1/) restrictions, cookies set from JavaScript are capped to 7 days of storage.
This example demonstrates how to use Compute to set a first-party server-side cookie for Google Analytics that can last longer.

#### [`simplecache`](./simplecache)

This example demonstrates the [Simple Cache API](https://pkg.go.dev/github.com/fastly/compute-sdk-go/cache/simple).

Resources are specified by the URL path.  A `GET` will retrieve a value from the cache, a `POST` will insert the value into the cache, and a `DELETE` will purge the value from the cache.

Note that this example isn't currently supported by Viceroy and has to be deployed to a Fastly service, which you can do with `fastly compute build` and `fastly compute deploy`.
If you haven't previously deployed it, the CLI will walk you through the steps of creating a new service for it.

#### [`stream-response`](./stream-response)

This example demonstrates a server that streams a response to the client by writing to the `http.ResponseWriter` in chunks, sleeping between each chunk.

#### [`with-timeout`](./with-timeout)

This example demonstrates how to use [`context.WithTimeout`](https://pkg.go.dev/context#WithTimeout) to set a timeout for origin requests.
It is expected that this example prints "context deadline exceeded" to the logs.
