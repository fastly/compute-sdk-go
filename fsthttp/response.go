// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// ResponseLimits are the limits for the components of an HTTP response.
var ResponseLimits Limits

// Response to an outgoing HTTP request made by this server.
type Response struct {
	// Request associated with the response.
	Request *Request

	// Backend that served the response.
	Backend string

	// StatusCode of the response.
	StatusCode int

	// Header received with the response.
	Header Header

	// Body of the response.
	Body io.ReadCloser

	cacheResponse cacheResponse

	abi struct {
		resp *fastly.HTTPResponse
	}
}

// Cookies parses and returns the cookies set in the Set-Cookie headers.
func (resp *Response) Cookies() []*Cookie {
	return readSetCookies(resp.Header)
}

// RemoteAddr returns the address of the server that provided the response.
func (resp *Response) RemoteAddr() (net.Addr, error) {
	var addr netaddr
	var err error

	addr.ip, err = resp.abi.resp.GetAddrDestIP()
	if err != nil {
		return nil, fmt.Errorf("get addr dest ip: %w", err)
	}

	addr.port, err = resp.abi.resp.GetAddrDestPort()
	if err != nil {
		return nil, fmt.Errorf("get addr dest port: %w", err)
	}

	return &addr, nil
}

type netaddr struct {
	ip   net.IP
	port uint16
}

var _ net.Addr = (*netaddr)(nil)

func (n *netaddr) Network() string {
	return "tcp"
}

func (n *netaddr) String() string {
	return net.JoinHostPort(n.ip.String(), strconv.Itoa(int(n.port)))
}

func newResponse(req *Request, backend string, abiResp *fastly.HTTPResponse, abiBody *fastly.HTTPBody) (*Response, error) {
	code, err := abiResp.GetStatusCode()
	if err != nil {
		return nil, fmt.Errorf("status code: %w", err)
	}

	header := NewHeader()
	keys := abiResp.GetHeaderNames()
	for keys.Next() {
		k := string(keys.Bytes())
		vals := abiResp.GetHeaderValues(k)
		for vals.Next() {
			v := string(vals.Bytes())
			header.Add(k, v)
		}
		if err := vals.Err(); err != nil {
			return nil, fmt.Errorf("read header key %q: %w", k, err)
		}
	}
	if err := keys.Err(); err != nil {
		return nil, fmt.Errorf("read header keys: %w", err)
	}

	r := &Response{
		Request:    req,
		Backend:    backend,
		StatusCode: code,
		Header:     header,
		Body:       abiBody,
	}

	r.abi.resp = abiResp
	return r, nil
}

const (
	fastlyDebug      = "fastly-debug"
	fastlyFF         = "fastly-ff"
	surrogateControl = "surrogate-control"
	surrogateKey     = "surrogate-key"
	xCache           = "x-cache"
	xCacheHits       = "x-cache-hits"

	val0    = "0"
	valHIT  = "HIT"
	valMISS = "MISS"
)

func (resp *Response) updateFastlyCacheHeaders(req *Request) {
	// TODO(dgryski): Set the abi headers too or just the map slice?

	if hits := resp.cacheResponse.hits; hits != 0 {
		resp.Header.Add(xCache, valHIT)
		resp.Header.Add(xCacheHits, strconv.Itoa(int(hits)))
	} else {
		resp.Header.Add(xCache, valMISS)
		resp.Header.Add(xCacheHits, val0)
	}

	shouldRemoveSurrogateHeaders := req.Header.Get(fastlyFF) == "" && req.Header.Get(fastlyDebug) == ""
	if shouldRemoveSurrogateHeaders {
		resp.Header.Del(surrogateKey)
		resp.Header.Del(surrogateControl)
	}
}

func (resp *Response) wasWrittenToCache() bool {
	return false ||
		(resp.cacheResponse.storageAction == fastly.HTTPCacheStorageActionInsert) ||
		(resp.cacheResponse.storageAction == fastly.HTTPCacheStorageActionUpdate)
}

// FromCache returns whether the response was returned from the cache (true) or fresh from the backend (false).
func (resp *Response) FromCache() bool {
	// If we had to write it to the cache, then we must have fetched it from the backend, ergo it was not cached.
	// but check the storage action didn't prevent it from being written
	return !resp.wasWrittenToCache() &&
		(resp.cacheResponse.storageAction != fastly.HTTPCacheStorageActionDoNotStore) &&
		(resp.cacheResponse.storageAction != fastly.HTTPCacheStorageActionRecordUncacheable)
}

// TTL returns the Time to Live (TTL) in seconds in the cache for this response.
//
// The TTL determines the duration of "freshness" for the cached response
// after it is inserted into the cache.
func (resp *Response) TTL() (uint32, bool) {
	if resp.wasWrittenToCache() {
		return 0, false
	}

	return resp.cacheResponse.cacheWriteOptions.maxAge - resp.cacheResponse.cacheWriteOptions.age, true
}

// Age returns current age in seconds of the cached item, relative to the originating backend.
func (resp *Response) Age() (uint32, bool) {
	if resp.wasWrittenToCache() {
		return 0, false
	}

	return resp.cacheResponse.cacheWriteOptions.age, true
}

// StaleWhileRevalidate returns the time in seconds for which a cached item can safely be used despite being considered stale.
func (resp *Response) StaleWhileRevalidate() (uint32, bool) {
	if resp.wasWrittenToCache() {
		return 0, false
	}

	return resp.cacheResponse.cacheWriteOptions.stale, true
}

// Vary returns the set of request headers for which the response may vary.
func (resp *Response) Vary() string {
	return resp.cacheResponse.cacheWriteOptions.vary
}

// SurrogateKeys returns the surrogate keys for the cached response.
func (resp *Response) SurrogateKeys() string {
	return resp.cacheResponse.cacheWriteOptions.surrogate
}

// ResponseWriter is used to respond to client requests.
type ResponseWriter interface {
	// Header returns the headers that will be sent by WriteHeader.
	// Changing the returned headers after a call to WriteHeader has no effect.
	Header() Header

	// WriteHeader initiates the response to the client by sending an HTTP
	// response preamble with the provided status code, and all of the response
	// headers collected by Header. If WriteHeader is not called explicitly,
	// the first call to Write or Close will trigger an implicit call to
	// WriteHeader with a code of 200. After the first call to WriteHeader,
	// subsequent calls will print a warning but have no effect.
	//
	// 1xx status codes other than 103 (Early Hints) are not permitted and will
	// result in an ErrInvalidStatusCode error when calling Write, Close, or
	// Append, and clients will receive a 500 (Internal Server Error) response.
	WriteHeader(code int)

	// Write the data to the connection as part of an HTTP reply.
	//
	// If WriteHeader has not yet been called, Write calls WriteHeader(200)
	// before writing the data. Unlike the ResponseWriter in net/http, Write
	// will not automatically add Content-Type or Content-Length headers.
	Write(p []byte) (int, error)

	// Close the response to the client. The ResponseWriter is automatically
	// closed after the request handler finishes running, so it is not
	// necessary to call it explicitly, but doing so indicates that the
	// response can be fully written to the client immediately.
	//
	// If WriteHeader has not yet been called, Close calls WriteHeader(200)
	// before closing the response. Once closed, a ResponseWriter is
	// invalidated, and may no longer be accessed or used in any way.
	Close() error

	// SetManualFramingMode controls how the framing headers
	// (Content-Length/Transfer-Encoding) are set for a response.
	//
	// If set to true, the response uses the exact framing headers
	// set in the message.  If set to false, or set to true and
	// the framing is invalid, the framing headers are based on the
	// message body, and any framing headers already set in the message are
	// discarded.
	//
	// To have an effect on the response, this must be called before any call to Write() or WriteHeader().
	SetManualFramingMode(bool)

	// Append a body onto the end of this response. Will fail if passed anything other than a Response's Body field.
	// This operation is performed in amortized constant time, and so should always be preferred to directly copying a body with io.Copy.
	Append(other io.ReadCloser) error
}

type responseWriter struct {
	header            Header
	abiResp           *fastly.HTTPResponse
	abiBody           *fastly.HTTPBody
	wroteHeaders      bool
	closed            bool
	ManualFramingMode bool
	sendErr           error
}

func newResponseWriter() (*responseWriter, error) {
	abiResp, err := fastly.NewHTTPResponse()
	if err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	abiRespBody, err := fastly.NewHTTPBody()
	if err != nil {
		return nil, fmt.Errorf("create response body: %w", err)
	}

	return &responseWriter{
		header:  NewHeader(),
		abiResp: abiResp,
		abiBody: abiRespBody,
	}, nil
}

func (resp *responseWriter) Header() Header {
	return resp.header
}

var excludeHeadersNoBody = map[string]bool{CanonicalHeaderKey("Content-Length"): true, CanonicalHeaderKey("Transfer-Encoding"): true}

func (resp *responseWriter) WriteHeader(code int) {
	if resp.wroteHeaders {
		println("fsthttp: multiple calls to WriteHeader")
		return
	}

	resp.abiResp.SetFramingHeadersMode(resp.ManualFramingMode)
	resp.abiResp.SetStatusCode(code)

	var skip map[string]bool
	if code == StatusEarlyHints {
		skip = excludeHeadersNoBody
	}

	for _, key := range resp.header.Keys() {
		// don't send body headers if we're sending early hints
		if skip[key] {
			continue
		}
		resp.abiResp.SetHeaderValues(key, resp.header.Values(key))
	}

	// WriteHeader is infallible, so if we're unable to create the downstream response capture the
	// error and return it on Write, Append, and Close calls.  serve() will panic on this error and
	// ensure that a 500 error is sent to the client.
	// EarlyHints are buffered and sent all-at-once so we can stream later.  Other status codes immediately start a streaming response.
	stream := code != StatusEarlyHints
	resp.sendErr = resp.abiResp.SendDownstream(resp.abiBody, stream)
	if resp.sendErr != nil {
		// FastlyStatusInval is returned if an invalid status code (1xx except 103 Early Hints) is used.
		if status, ok := fastly.IsFastlyError(resp.sendErr); ok && status == fastly.FastlyStatusInval {
			resp.sendErr = ErrInvalidStatusCode
		}
	}

	if code == StatusEarlyHints {
		// For early hints, don't mark the headers as "sent" so we can send them again next time.
		return
	}

	resp.wroteHeaders = true
}

var (
	// ErrClosed is returned when attempting to write to a ResponseWriter whose network connection has been closed.
	ErrClosed = errors.New("connection has been closed")

	// ErrInvalidStatusCode is returned when attempting to write a resposne with an invalid HTTP status code.
	ErrInvalidStatusCode = errors.New("invalid HTTP status code")
)

func (resp *responseWriter) Write(p []byte) (int, error) {
	if resp.sendErr != nil {
		return 0, resp.sendErr
	}
	if !resp.wroteHeaders {
		resp.WriteHeader(200)
	}
	n, err := resp.abiBody.Write(p)
	if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusBadf {
		err = ErrClosed
	}
	return n, err
}

func (resp *responseWriter) Close() error {
	if resp.sendErr != nil {
		return resp.sendErr
	}
	if !resp.wroteHeaders {
		resp.WriteHeader(200)
	}
	if resp.closed {
		return nil
	}
	resp.closed = true
	return resp.abiBody.Close()
}

func (resp *responseWriter) SetManualFramingMode(mode bool) {
	resp.ManualFramingMode = mode
}

func (resp *responseWriter) Append(other io.ReadCloser) error {
	if resp.sendErr != nil {
		return resp.sendErr
	}
	if !resp.wroteHeaders {
		resp.WriteHeader(200)
	}
	otherAbiBody, ok := other.(*fastly.HTTPBody)
	if !ok {
		return fmt.Errorf("non-Response Body passed to ResponseWriter.Append")
	}
	resp.abiBody.Append(otherAbiBody)
	return nil
}
