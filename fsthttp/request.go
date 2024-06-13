// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// RequestLimits are the limits for the components of an HTTP request.
var RequestLimits = Limits{
	maxHeaderNameLen:  fastly.DefaultMaxHeaderNameLen,
	maxHeaderValueLen: fastly.DefaultMaxHeaderValueLen,
	maxMethodLen:      fastly.DefaultMaxMethodLen,
	maxURLLen:         fastly.DefaultMaxURLLen,
}

// Request represents an HTTP request received by this server from a requesting
// client, or to be sent from this server during this execution. Some fields
// only have meaning in one context or the other.
type Request struct {
	// Method specifies the HTTP method: GET, POST, PUT, HEAD, etc.
	Method string

	// URL is the parsed and validated URL of the request.
	//
	// Outgoing requests are always sent to the preconfigured backend provided
	// to the send method. The URL is used for the request resource, Host
	// header, etc.
	URL *url.URL

	// Proto contains the HTTP protocol version used for incoming requests.
	//
	// These fields are ignored for outgoing requests.
	Proto      string // "HTTP/1.0"
	ProtoMajor int    // 1
	ProtoMinor int    // 0

	// Header contains the request header fields either received in the
	// incoming request, or to be sent with the outgoing request.
	Header Header

	// CacheOptions control caching behavior for outgoing requests.
	CacheOptions CacheOptions

	// Body is the request's body.
	//
	// For the incoming client request, the body will always be non-nil, but
	// reads may return immediately with EOF. For outgoing requests, the body is
	// optional. A body may only be read once.
	//
	// Prefer using the SetBody method over assigning to this value directly,
	// as it enables optimizations when sending outgoing requests.  See the
	// SetBody documentation for more information.
	Body io.ReadCloser

	// Host is the hostname parsed from the incoming request URL.
	Host string

	// RemoteAddr contains the IP address of the requesting client.
	//
	// This field is ignored for outgoing requests.
	RemoteAddr string

	// TLSInfo collects TLS metadata for incoming requests received over HTTPS.
	TLSInfo TLSInfo

	// SendPollInterval determines how often the Send method will check for
	// completed requests. While polling, the Go runtime is suspended, and all
	// user code stops execution. A shorter interval will make Send more
	// responsive, but provide less CPU time to user code. A longer poll
	// interval will make Send less responsive, but provide more CPU time to
	// user code.
	//
	// If SendPollInterval is zero, a default value of 1ms is used. The minimum
	// value is 1ms, and the maximum value is 1s.
	SendPollInterval time.Duration

	// DecompressResponseOptions control the auto decompress response behaviour.
	DecompressResponseOptions DecompressResponseOptions

	// ManualFramingMode controls how the framing headers
	// (Content-Length/Transfer-Encoding) are set for a request.
	//
	// If ManualFramingMode is true, the request uses the exact framing headers
	// set in the message.  If ManualFramingMode is false, or ManualFramingMode
	// is true and the framing is invalid, the framing headers are based on the
	// message body, and any framing headers already set in the message are
	// discarded.
	ManualFramingMode bool

	sent bool // a request may only be sent once

	abi struct {
		req  *fastly.HTTPRequest
		body *fastly.HTTPBody
	}
}

// NewRequest constructs an outgoing request with the given HTTP method, URI,
// and body. The URI is parsed via url.Parse.
func NewRequest(method string, uri string, body io.Reader) (*Request, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = nopCloser{body}
	}

	return &Request{
		Method: method,
		URL:    u,
		Header: NewHeader(),
		Body:   rc,
		Host:   u.Host,
	}, nil
}

func newClientRequest() (*Request, error) {
	abiReq, abiReqBody, err := fastly.BodyDownstreamGet()
	if err != nil {
		return nil, fmt.Errorf("get client request and body: %w", err)
	}

	method, err := abiReq.GetMethod()
	if err != nil {
		return nil, fmt.Errorf("get method: %w", err)
	}

	uri, err := abiReq.GetURI()
	if err != nil {
		return nil, fmt.Errorf("get URI: %w", err)
	}

	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parse URI: %w", err)
	}

	proto, major, minor, err := abiReq.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("get protocol version: %w", err)
	}

	header := NewHeader()
	keys := abiReq.GetHeaderNames(RequestLimits.maxHeaderNameLen)
	for keys.Next() {
		k := string(keys.Bytes())
		vals := abiReq.GetHeaderValues(k, RequestLimits.maxHeaderValueLen)
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

	remoteAddr, err := fastly.DownstreamClientIPAddr()
	if err != nil {
		return nil, fmt.Errorf("get client IP: %w", err)
	}

	var tlsInfo TLSInfo
	switch u.Scheme {
	case "https":
		tlsInfo.Protocol, err = fastly.DownstreamTLSProtocol()
		if err != nil {
			return nil, fmt.Errorf("get TLS protocol: %w", err)
		}

		tlsInfo.ClientHello, err = fastly.DownstreamTLSClientHello()
		if err != nil {
			return nil, fmt.Errorf("get TLS client hello: %w", err)
		}

		tlsInfo.CipherOpenSSLName, err = fastly.DownstreamTLSCipherOpenSSLName()
		if err != nil {
			return nil, fmt.Errorf("get TLS cipher name: %w", err)
		}
	}

	// Setting the fsthttp.Request Host field to the url.URL Host field is
	// considered safe as the C@E hostcall to retrieve the URL, which is then
	// passed onto the guest, will always be an absolute one.
	return &Request{
		Method:     method,
		URL:        u,
		Proto:      proto,
		ProtoMajor: major,
		ProtoMinor: minor,
		Header:     header,
		Body:       abiReqBody,
		Host:       u.Host,
		RemoteAddr: remoteAddr.String(),
		TLSInfo:    tlsInfo,
	}, nil
}

// SetBody sets the [Request]'s body to the provided [io.Reader]. Prefer
// using this method over setting the Body field directly, as it enables
// optimizations in the runtime.
//
// If an unread body from an incoming client request is set on an
// outgoing upstream request, the body will be efficiently streamed from
// the incoming request.  It is also possible to set the unread body of
// a received response to the body of a request, with the same results.
//
// If the body is set from an in-memory reader such as [bytes.Buffer],
// [bytes.Reader], or [strings.Reader], the runtime will send the
// request with a Content-Length header instead of Transfer-Encoding:
// chunked.
func (req *Request) SetBody(body io.Reader) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = nopCloser{body}
	}

	req.Body = rc
}

// Clone returns a copy of the request. The returned copy will have a nil Body
// field, and its URL will have a nil User field.
func (req *Request) Clone() *Request {
	return &Request{
		Method:                    req.Method,
		URL:                       cloneURL(req.URL),
		Proto:                     req.Proto,
		ProtoMajor:                req.ProtoMajor,
		ProtoMinor:                req.ProtoMinor,
		Header:                    req.Header.Clone(),
		CacheOptions:              req.CacheOptions,
		Body:                      nil,
		Host:                      req.URL.Host,
		RemoteAddr:                req.RemoteAddr,
		TLSInfo:                   req.TLSInfo,
		SendPollInterval:          req.SendPollInterval,
		DecompressResponseOptions: req.DecompressResponseOptions,
		ManualFramingMode:         req.ManualFramingMode,
	}
}

// CloneWithBody returns a copy of the request, with the Body field set
// to the provided io.Reader.  Its URL will have a nil User field.
func (req *Request) CloneWithBody(body io.Reader) *Request {
	r := req.Clone()
	r.SetBody(body)
	return r
}

func cloneURL(u *url.URL) *url.URL {
	return &url.URL{
		Scheme:      u.Scheme,
		Opaque:      u.Opaque,
		User:        nil,
		Host:        u.Host,
		Path:        u.Path,
		RawPath:     u.RawPath,
		ForceQuery:  u.ForceQuery,
		RawQuery:    u.RawQuery,
		Fragment:    u.Fragment,
		RawFragment: u.RawFragment,
	}
}

// Cookies parses and returns the HTTP cookies sent with the request.
func (req *Request) Cookies() []*Cookie {
	return readCookies(req.Header, "")
}

// ErrNoCookie is returned by Request's Cookie method when a cookie is not found.
var ErrNoCookie = errors.New("fsthttp: named cookie not present")

// Cookie returns the named cookie provided in the request or
// ErrNoCookie if not found.
// If multiple cookies match the given name, only one cookie will
// be returned.
func (req *Request) Cookie(name string) (*Cookie, error) {
	for _, c := range readCookies(req.Header, name) {
		return c, nil
	}
	return nil, ErrNoCookie
}

// AddCookie adds a cookie to the request. Per RFC 6265 section 5.4,
// AddCookie does not attach more than one Cookie header field. That
// means all cookies, if any, are written into the same line,
// separated by semicolon.
// AddCookie only sanitizes c's name and value, and does not sanitize
// a Cookie header already present in the request.
func (req *Request) AddCookie(c *Cookie) {
	s := fmt.Sprintf("%s=%s", sanitizeCookieName(c.Name), sanitizeCookieValue(c.Value))
	if c := req.Header.Get("Cookie"); c != "" {
		req.Header.Set("Cookie", c+"; "+s)
	} else {
		req.Header.Set("Cookie", s)
	}
}

// Send the request to the named backend. Requests may only be sent to
// backends that have been preconfigured in your service, regardless of
// their URL. Once sent, a request cannot be sent again.
//
// By default, read-through caching is enabled for requests.  The HTTP
// response received from the backend will be cached and reused for
// subsequent requests if it meets cacheability requirements.  The
// behavior of this automatic caching can be tuned (or disabled) via the
// [Request]'s [CacheOptions] field.  This function provides the full
// benefits of Fastly's purging, request collapsing, and revalidation
// capabilities, and is recommended for most users who need to cache
// HTTP responses.
func (req *Request) Send(ctx context.Context, backend string) (*Response, error) {
	if req.sent {
		return nil, fmt.Errorf("request already sent")
	}

	if req.abi.req == nil && req.abi.body == nil {
		//  abi request not yet constructed
		if err := req.constructABIRequest(); err != nil {
			return nil, err
		}
	}

	// When the request's ManualFramingMode is false, SendAsyncStreaming
	// streams the request body to the backend using "Transfer-Encoding:
	// chunked".  SendAsync buffers the entire body and sends it with a
	// "Content-Length" header.
	//
	// For requests without a body, we want to avoid unnecessary chunked
	// encoding, and have observed servers that error when seeing it in
	// certain contexts.
	//
	// For requests where the body is an io.Reader implementer where the
	// size is known in advance, we want to send that along with a
	// Content-Length as well.  Those types are *bytes.Buffer,
	// *bytes.Reader, and *strings.Reader.
	//
	// For all other requests, we stream with chunked encoding.
	var (
		abiPending *fastly.PendingRequest
		err        error
		streaming  bool = true
		errc            = make(chan error, 3) // needs to be buffered to the max number of writes in copyBody()
	)

	switch underlyingReaderFrom(req.Body).(type) {
	case nil, *bytes.Buffer, *bytes.Reader, *strings.Reader:
		streaming = false
	}

	req.sent = true

	if streaming {
		go req.copyBody(errc)
		abiPending, err = req.abi.req.SendAsyncStreaming(req.abi.body, backend)
	} else {
		req.copyBody(errc)
		abiPending, err = req.abi.req.SendAsync(req.abi.body, backend)
	}
	if err != nil {
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusInval {
			return nil, ErrBackendNotFound
		}

		return nil, fmt.Errorf("begin send: %w", err)
	}

	pollInterval := safePollInterval(req.SendPollInterval)
	abiResp, abiRespBody, err := func() (*fastly.HTTPResponse, *fastly.HTTPBody, error) {
		for {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()

			case err := <-errc:
				if err != nil {
					return nil, nil, err
				}

			default:
				done, abiResp, abiRespBody, err := abiPending.Poll()
				if err != nil {
					return nil, nil, err
				}
				if done {
					return abiResp, abiRespBody, nil
				}
				time.Sleep(pollInterval)
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("poll: %w", err)
	}

	resp, err := newResponse(req, backend, abiResp, abiRespBody)
	if err != nil {
		return nil, fmt.Errorf("construct response: %w", err)
	}

	return resp, nil
}

func (req *Request) copyBody(errc chan<- error) {
	var (
		bodyExists   = req.Body != nil
		_, bodyIsABI = req.Body.(*fastly.HTTPBody)
		shouldCopy   = bodyExists && !bodyIsABI
	)

	if shouldCopy {
		_, copyErr := io.Copy(req.abi.body, req.Body)
		errc <- maybeWrap(copyErr, "copy body")
		errc <- maybeWrap(req.Body.Close(), "close user body")
		if copyErr == nil {
			errc <- maybeWrap(req.abi.body.Close(), "close request body")
		} else {
			errc <- maybeWrap(req.abi.body.Abandon(), "abandon request body")
		}
	} else {
		errc <- maybeWrap(req.abi.body.Close(), "close request body")
	}
}

func (req *Request) constructABIRequest() error {
	if req.abi.req != nil || req.abi.body != nil {
		return fmt.Errorf("request already constructed")
	}

	abiReq, err := fastly.NewHTTPRequest()
	if err != nil {
		return fmt.Errorf("construct request: %w", err)
	}

	if err := abiReq.SetMethod(req.Method); err != nil {
		return fmt.Errorf("set method: %w", err)
	}

	if err := abiReq.SetURI(req.URL.String()); err != nil {
		return fmt.Errorf("set URL: %w", err)
	}

	if err := abiReq.SetAutoDecompressResponse(fastly.AutoDecompressResponseOptions(req.DecompressResponseOptions)); err != nil {
		return fmt.Errorf("set auto decompress response: %w", err)
	}

	if err := abiReq.SetFramingHeadersMode(req.ManualFramingMode); err != nil {
		return fmt.Errorf("set framing headers mode: %w", err)
	}

	if err := abiReq.SetCacheOverride(fastly.CacheOverrideOptions(req.CacheOptions)); err != nil {
		return fmt.Errorf("set cache options: %w", err)
	}

	for _, key := range req.Header.Keys() {
		vals := req.Header.Values(key)
		if err := abiReq.SetHeaderValues(key, vals); err != nil {
			return fmt.Errorf("set headers: %w", err)
		}
	}

	abiReqBody, err := abiBodyFrom(req.Body)
	if err != nil {
		return fmt.Errorf("get body: %w", err)
	}

	req.abi.req = abiReq
	req.abi.body = abiReqBody

	return nil
}

// CacheOptions control caching behavior for outgoing requests.
type CacheOptions struct {
	// Pass controls whether or not the request should be cached at all. By
	// default pass is false, which means the request will only reach the
	// backend if a cached response is not available. If pass is set to true,
	// the request will always be sent directly to the backend.
	//
	// Pass is mutually exclusive with all other cache options. Setting any
	// other option will force pass to false.
	Pass bool

	// PCI controls the PCI/HIPAA compliant, non-volatile caching of the
	// request. PCI is false by default, which means the request may not be
	// PCI/HIPAA compliant. If PCI is set to true, caching will be made
	// compliant, and pass will be forced to false.
	//
	// https://docs.fastly.com/products/pci-compliant-caching-and-delivery
	PCI bool

	// TTL represents a Time-to-Live for cached responses to the request, in
	// seconds. If greater than zero, it overrides any behavior specified in the
	// response headers, and forces pass to false.
	TTL uint32

	// StaleWhileRevalidate represents a stale-while-revalidate time for the
	// request, in seconds. If greater than zero, it overrides any behavior
	// specified in the response headers, and forces pass to false.
	StaleWhileRevalidate uint32

	// SurrogateKey represents an explicit surrogate key for the request, which
	// will be added to any `Surrogate-Key` response headers received from the
	// backend. If nonempty, it forces pass to false.
	//
	// https://docs.fastly.com/en/guides/purging-api-cache-with-surrogate-keys
	SurrogateKey string
}

// TLSInfo collects TLS-related metadata for incoming requests. All fields are
// ignored for outgoing requests.
type TLSInfo struct {
	// Protocol contains the TLS protocol version used to secure the client TLS
	// connection, if any.
	Protocol string

	// ClientHello contains raw bytes sent by the client in the TLS ClientHello
	// message. See RFC 5246 for details.
	ClientHello []byte

	// CipherOpenSSLName contains the cipher suite used to secure the client TLS
	// connection. The value returned will be consistent with the OpenSSL name
	// for the cipher suite.
	CipherOpenSSLName string
}

// DecompressResponseOptions control the auto decompress response behaviour.
type DecompressResponseOptions struct {
	// Gzip controls whether a gzip-encoded response to the request will be
	// automatically decompressed.
	//
	// If the response to the request is gzip-encoded, it will be presented in
	// decompressed form, and the Content-Encoding and Content-Length headers
	// will be removed.
	Gzip bool
}

// nopCloser is functionally the same as io.NopCloser, except that we
// can get to the underlying io.Reader.
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func (n nopCloser) reader() io.Reader {
	return n.Reader
}

func underlyingReaderFrom(rc io.ReadCloser) io.Reader {
	if rc == nil {
		return nil
	}

	if nc, ok := rc.(nopCloser); ok {
		return nc.reader()
	}

	return rc.(io.Reader)
}

func abiBodyFrom(rc io.ReadCloser) (*fastly.HTTPBody, error) {
	b, ok := rc.(*fastly.HTTPBody)
	if ok {
		return b, nil
	}

	b, err := fastly.NewHTTPBody()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func safePollInterval(d time.Duration) time.Duration {
	const (
		min = 1 * time.Millisecond
		max = 1 * time.Second
	)
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}

func maybeWrap(err error, annotation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", annotation, err)
}
