## Unreleased

### Added

- secretstore: add Plaintext toplevel convenience function
- acl: add ACL hostcalls

## 1.3.3 (2024-09-12)

### Added

- kvstore: add ErrTooManyRequests
- fsthttp: add ServerAddr to Request
- fsthttp: add pluggable URL parser
- fsthttp: add TCP and HTTP keepalives configuration for backends
- fsthttp: add RemoteAddr to Response
- fsthttp: add pooling connection configuration for backends
- compute: add GetVCPUMilliseconds
- fsthttp: add client certificate configuration for backends
- fsthttp: add grpc flag for backends

### Changed

- configstore: switch to new configstore hostcalls

## 1.3.2 (2024-06-25)

### Added

- configstore: add Store.Has() method
- configstore: add Store.GetBytes() method
- configstore: reduce memory usage with shared buffer

### Changed

- fsthttp: make buffer sizes adaptable

## 1.3.1 (2024-04-25)

### Added

- `kvstore`: add Store.Delete method

### Changed

- Update Viceroy requirement to 0.9.6

## 1.3.0 (2024-02-21)

### Added

- Add support for edge rate limiting (`erl`)

## 1.2.1 (2024-01-19)

### Added

- Better error handling for geo data

### Changed

- Copy, don't stream, in-memory io.Readers like bytes.Buffer, bytes.Reader and strings.Reader
- Fix a bug where a panic under Go (but not TinyGo) would result in handlers returning 200 OK instead of 500 Internal Server Error by not deferring Close() on the response writer internally.

## 1.2.0 (2023-11-17)

### Added

- Add support for device detection (`device`)

### Changed

- Switch geolocation internals to use `encoding/json` from a custom built parser

## 1.1.0 (2023-10-31)

### Added

- Improve error handling and documentation in `kvstore` package
- Use new hostcalls for better error messages when sending requests to a backend
- Add better unexpected error handling (`cache/core`, `configstore`, `secretstore`)

## 1.0.0 (2023-09-13)

- Unchanged from 0.2.0

### Added

- Tag version 1.0.0

## 0.2.0 (2023-08-11)

### Added

- Add support for Go 1.21 WASI

### Changed

- Remove support for TinyGo 0.28.0 and earlier

## 0.1.7 (2023-08-04)

### Added

- Add Append method to ResponseWriter
- Add SecretFromBytes to secretstore
- Add support for HandoffWebsocket, HandoffFanout hostcalls (`exp/handoff`)
- Add support for backend query API (`backend`)
- Add support for testing via Viceroy with `go test`

### Changed

- Improve returned errors

## 0.1.6 (2023-07-13)

### Added

- Add Simple Cache API

## 0.1.5 (2023-06-23)

### Changed

- Fix KV Store hostcalls

### Added

- Add support for RegisterDynamicBackend

## 0.1.4 (2023-05-30)

### Changed

- Send `Content-Length: 0` instead of  `Transfer-Encoding: chunked` for requests without a body

### Added

- Add Core Cache API
- Add Purge API
- Add package level documentation for Secret Store and KV Store APIs

## 0.1.3 (2023-05-15)

### Changed

- Rename objectstore -> kvstore
- Deprecate fstctx

### Added

- Add fsthttp.RequestLimits

## 0.1.2 (2023-01-30)

### Changed

- Renamed edgedict -> configstore.
- Made HTTP Request/Response field size limit configurable

### Added

- Add support for Object Store API
- Add support for Secret Store API
- Add adaptor for net/http.RoundTripper (for net/http.Client support)
- Add adaptor for net/http.Handler
- Add fsthttp.Error() and fsthttp.NotFound() helpers

--
## 0.1.1 (2022-06-14)

### Changed

- Use Go 1.17

--
## 0.1.0 (2022-06-11)

### Added

- Initial Release
