// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp/imageopto"
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// RequestLimits are the limits for the components of an HTTP request.
var RequestLimits Limits

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

	// ServerAddr contains the IP address of the server that received the
	// HTTP request.
	//
	// This field is ignored for outgoing requests.
	ServerAddr string

	// TLSInfo collects TLS metadata for incoming requests received over HTTPS.
	TLSInfo TLSInfo

	// tlsClientCertificateInfo is information about the tls client certificate, if available
	clientCertificate *TLSClientCertificateInfo

	// FastlyMeta collects Fastly-specific metadata for incoming requests
	fastlyMeta *FastlyMeta

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

	// SendPollIntervalFn allows more fine-grained control of the send poll interval.
	//
	// SendPollIntervalFn must be a function which takes an iteration number i and returns
	// the delay for the i'th polling interval.
	SendPollIntervalFn func(i int) time.Duration

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

	// ImageOptimizerOptions control the image optimizer request.
	ImageOptimizerOptions *imageopto.Options

	sent bool // a request may only be sent once

	abi        reqAbi
	downstream reqAbi
}

type reqAbi struct {
	req  *fastly.HTTPRequest
	body *fastly.HTTPBody
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

// _parseRequestURI can be set by SetParseRequestURI
var _parseRequestURI func(string) (*url.URL, error) = url.ParseRequestURI

func newClientRequest(abiReq *fastly.HTTPRequest, abiReqBody *fastly.HTTPBody) (*Request, error) {
	method, err := abiReq.GetMethod()
	if err != nil {
		return nil, fmt.Errorf("get method: %w", err)
	}

	uri, err := abiReq.GetURI()
	if err != nil {
		return nil, fmt.Errorf("get URI: %w", err)
	}

	u, err := _parseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parse URI: %w", err)
	}

	proto, major, minor, err := abiReq.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("get protocol version: %w", err)
	}

	header := NewHeader()
	keys := abiReq.GetHeaderNames()
	for keys.Next() {
		k := string(keys.Bytes())
		vals := abiReq.GetHeaderValues(k)
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

	remoteAddr, err := abiReq.DownstreamClientIPAddr()
	if err != nil {
		return nil, fmt.Errorf("get client IP: %w", err)
	}

	serverAddr, err := abiReq.DownstreamServerIPAddr()
	if err != nil {
		return nil, fmt.Errorf("get server IP: %w", err)
	}

	var tlsInfo TLSInfo
	switch u.Scheme {
	case "https":
		tlsInfo.Protocol, err = abiReq.DownstreamTLSProtocol()
		if err != nil {
			return nil, fmt.Errorf("get TLS protocol: %w", err)
		}

		tlsInfo.ClientHello, err = abiReq.DownstreamTLSClientHello()
		if err != nil {
			return nil, fmt.Errorf("get TLS client hello: %w", err)
		}

		tlsInfo.CipherOpenSSLName, err = abiReq.DownstreamTLSCipherOpenSSLName()
		if err != nil {
			return nil, fmt.Errorf("get TLS cipher name: %w", err)
		}

		tlsInfo.JA3MD5, err = abiReq.DownstreamTLSJA3MD5()
		if err != nil {
			return nil, fmt.Errorf("get TLS JA3 MD5: %w", err)
		}

		tlsInfo.JA4, err = abiReq.DownstreamTLSJA4()
		if err != nil {
			return nil, fmt.Errorf("get TLS JA4: %w", err)
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
		ServerAddr: serverAddr.String(),
		TLSInfo:    tlsInfo,
		downstream: reqAbi{req: abiReq, body: abiReqBody},
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

	// reset abiBody if we need to
	if req.abi.body != nil {
		// TODO(dgryski): sadly ignoring error here :(
		req.abi.body, _ = abiBodyFrom(req.Body)
	}
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
		SendPollIntervalFn:        req.SendPollIntervalFn,
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

// FastlyMeta returns a fleshed-out FastlyMeta object for the request.
func (req *Request) FastlyMeta() (*FastlyMeta, error) {
	if req.fastlyMeta != nil {
		return req.fastlyMeta, nil
	}

	var err error
	var fastlyMeta FastlyMeta

	fastlyMeta.SandboxID = os.Getenv("FASTLY_TRACE_ID")

	fastlyMeta.RequestID, err = req.downstream.req.DownstreamRequestID()
	if err != nil {
		return nil, fmt.Errorf("get request ID: %w", err)
	}

	fastlyMeta.H2, err = req.downstream.req.DownstreamH2Fingerprint()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("get H2 fingerprint: %w", err)
	}

	fastlyMeta.OH, err = req.downstream.req.DownstreamOHFingerprint()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("get OH fingerprint: %w", err)
	}

	fastlyMeta.DDOSDetected, err = req.downstream.req.DownstreamDDOSDetected()
	if err != nil {
		return nil, fmt.Errorf("get ddos detected: %w", err)
	}

	fastlyMeta.FastlyKeyIsValid, err = req.downstream.req.DownstreamFastlyKeyIsValid()
	if err != nil {
		return nil, fmt.Errorf("get fastly key is valid: %w", err)
	}

	req.fastlyMeta = &fastlyMeta

	return req.fastlyMeta, nil
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
		if err := req.setABIRequestOptions(); err != nil {
			return nil, err
		}
	}

	if req.ImageOptimizerOptions != nil {
		response, err := req.sendToImageOpto(ctx, backend)
		if err != nil {
			return nil, fmt.Errorf("send to image optimizer: %w", err)
		}

		return response, nil
	}

	if ok, err := req.shouldUseGuestCaching(); err != nil {
		// can't determine if we should use guest cache or host cache
		return nil, err
	} else if ok {
		response, err := req.sendWithGuestCache(ctx, backend)
		if err != nil {
			return nil, fmt.Errorf("send with guest cache: %w", err)
		}

		return response, nil
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
	case nil, *bytes.Buffer, *bytes.Reader, *strings.Reader, *fastly.HTTPBody:
		streaming = false
	}

	// use regular fastly host caching
	// handle normal request flow here

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

	resp, err := newResponseFromABIPending(ctx, req, backend, abiPending, errc)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func newResponseFromABIPending(ctx context.Context, req *Request, backend string, abiPending *fastly.PendingRequest, errc chan error) (*Response, error) {
	pollIntervalFn := req.SendPollIntervalFn
	if pollIntervalFn == nil {
		pollIntervalFn = func(n int) time.Duration { return req.SendPollInterval }
	}

	abiResp, abiRespBody, err := pendingToABIResponse(ctx, errc, abiPending, pollIntervalFn)
	if err != nil {
		return nil, fmt.Errorf("poll: %w", err)
	}

	resp, err := newResponse(req, backend, abiResp, abiRespBody)
	if err != nil {
		return nil, fmt.Errorf("construct response: %w", err)
	}

	return resp, nil
}

func pendingToABIResponse(ctx context.Context, errc chan error, abiPending *fastly.PendingRequest, pollIntervalFn func(int) time.Duration) (*fastly.HTTPResponse, *fastly.HTTPBody, error) {
	var iter int
	var pollInterval time.Duration

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
			pollInterval = safePollInterval(pollIntervalFn(iter))
			iter++
			time.Sleep(pollInterval)
		}
	}
}

func (req *Request) sendToImageOpto(ctx context.Context, backend string) (*Response, error) {
	// Send this request through the Image Optimizer infrastructure
	query, err := req.ImageOptimizerOptions.QueryString()
	if err != nil {
		return nil, err
	}

	abiResp, abiBody, err := req.abi.req.SendToImageOpto(req.abi.body, backend, query)
	if err != nil {
		return nil, err

	}

	resp, err := newResponse(req, backend, abiResp, abiBody)
	if err != nil {
		return nil, fmt.Errorf("construct response: %w", err)
	}

	return resp, nil
}

var guestCacheSWRPending sync.WaitGroup

func (req *Request) sendWithGuestCache(ctx context.Context, backend string) (*Response, error) {
	// use guest cache

	if ok, err := fastly.HTTPCacheIsRequestCacheable(req.abi.req); err != nil {
		return nil, fmt.Errorf("request not cacheable: %v", err)
	} else if !ok {
		// no error during lookup but request not cacheable;
		abiResp, abiBody, err := req.sendWithoutCaching(backend)
		if err != nil {
			return nil, err
		}

		resp, err := newResponse(req, backend, abiResp, abiBody)
		if err != nil {
			return nil, fmt.Errorf("construct response: %w", err)
		}

		resp.updateFastlyCacheHeaders(req)
		return resp, nil
	}

	var options fastly.HTTPCacheLookupOptions
	if key := req.CacheOptions.OverrideKey; key != "" {
		if len(key) != 32 {
			return nil, fmt.Errorf("bad length for OverrideKey: %v != 32", len(key))
		}
		options.OverrideKey(key)
		req.CacheOptions.OverrideKey = ""
	}

	// force the lookup to await in the host, retrieving any errors synchronously
	cacheHandle, err := fastly.HTTPCacheTransactionLookup(req.abi.req, &options)
	if err != nil {
		return nil, fmt.Errorf("cache transaction lookup: %w", err)
	}
	// in a function so we can change cacheHandle later and have it reflected here
	defer func() {
		if cacheHandle != nil {
			fastly.HTTPCacheTransactionClose(cacheHandle)
		}
	}()
	if err := httpCacheWait(cacheHandle); err != nil {
		return nil, err
	}

	// is there a "usable" cached response (i.e. fresh or within SWR period)
	resp, err := httpCacheGetFoundResponse(cacheHandle, req, backend, true)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		// got a response from the cache

		// if this is during SWR, we may be the "lucky winner" who is
		// tasked with performing a background revalidation
		if ok, _ := httpCacheMustInsertOrUpdate(cacheHandle); ok {
			pending, err := req.sendAsyncForCaching(ctx, cacheHandle, backend)
			if err != nil {
				return nil, err
			}

			// Wait for the pending respond, then call any after-end hooks
			guestCacheSWRPending.Add(1)
			go func(p *pendingBackendRequestForCaching, h *fastly.HTTPCacheHandle) {
				defer guestCacheSWRPending.Done()
				candidate, err := newCandidateFromPendingBackendCaching(p)
				if err != nil {
					// nowhere to log error
					return
				}
				candidate.applyInBackground()
				fastly.HTTPCacheTransactionClose(h)
			}(pending, cacheHandle)
			// let cache handle be closed in goroutine
			cacheHandle = nil
		}

		// Meanwhile, whether fresh or in SWR, we can immediately return
		// the cached response:
		resp.updateFastlyCacheHeaders(req)
		return resp, nil
	}

	// no cached response

	if ok, _ := httpCacheMustInsertOrUpdate(cacheHandle); ok {

		pending, err := req.sendAsyncForCaching(ctx, cacheHandle, backend)
		if err != nil {
			return nil, err
		}

		candidateResp, err := newCandidateFromPendingBackendCaching(pending)
		if err != nil {
			return nil, err
		}

		resp, err := candidateResp.applyAndStreamBack(req)
		if err != nil {
			return nil, err
		}
		resp.updateFastlyCacheHeaders(req)

		cacheHandle = nil

		return resp, nil
	}

	// Request collapsing has been disabled: pass the _original_ request through to the
	// origin without updating the cache.
	abiResp, abiBody, err := req.sendWithoutCaching(backend)
	if err != nil {
		return nil, err
	}

	resp, err = newResponse(req, backend, abiResp, abiBody)
	if err != nil {
		return nil, err
	}
	resp.updateFastlyCacheHeaders(req)
	return resp, nil
}

func newRequestFromHandle(reqh *fastly.HTTPRequest, body io.ReadCloser, headers Header, options CacheOptions) (*Request, error) {
	method, err := reqh.GetMethod()
	if err != nil {
		return nil, err
	}
	url, err := reqh.GetURI()
	if err != nil {
		return nil, err
	}

	req, _ := NewRequest(method, url, body)
	req.CacheOptions = options
	req.Header = headers
	req.abi.req = reqh
	req.abi.body, _ = abiBodyFrom(body)

	return req, nil
}

type pendingBackendRequestForCaching struct {
	cacheHandle  *fastly.HTTPCacheHandle
	pending      *fastly.PendingRequest
	afterSend    func(*CandidateResponse) error
	cacheOptions CacheOptions
	req          *Request
}

func (req *Request) sendAsyncForCaching(ctx context.Context, cacheHandle *fastly.HTTPCacheHandle, backend string) (*pendingBackendRequestForCaching, error) {
	reqh, err := fastly.HTTPCacheGetSuggestedBackendRequest(cacheHandle)
	if err != nil {
		return nil, fmt.Errorf("get suggested backend request: %w", err)
	}

	suggReq, err := newRequestFromHandle(reqh, req.Body, req.Header, req.CacheOptions)
	if err != nil {
		return nil, err
	}

	if suggReq.CacheOptions.BeforeSend != nil {
		if err := suggReq.CacheOptions.BeforeSend(suggReq); err != nil {
			// TODO(dgryski): sentinel ErrReject ?
			return nil, err
		}
	}

	// If BeforeSend calls SetBody, abi.body will be updated for the new Body
	finalCacheOptions := suggReq.CacheOptions
	if err = suggReq.setABIRequestOptions(); err != nil {
		return nil, err
	}

	// copy body
	var errc chan error = make(chan error, 3)
	suggReq.copyBody(errc)

	abiPending, err := suggReq.sendAsyncWithoutCaching(ctx, backend)
	if err != nil {
		return nil, err
	}

	return &pendingBackendRequestForCaching{
		cacheHandle:  cacheHandle,
		req:          suggReq,
		pending:      abiPending,
		afterSend:    req.CacheOptions.AfterSend,
		cacheOptions: finalCacheOptions,
	}, nil
}

func (req *Request) sendAsyncWithoutCaching(_ context.Context, backend string) (*fastly.PendingRequest, error) {
	abiPending, err := req.abi.req.SendAsyncV2(req.abi.body, backend, false)
	if err != nil {
		return nil, fmt.Errorf("send async: %w", err)
	}

	return abiPending, nil
}

func (req *Request) sendWithoutCaching(backend string) (*fastly.HTTPResponse, *fastly.HTTPBody, error) {
	abiResp, abiBody, err := req.abi.req.SendV3(req.abi.body, backend)
	if err != nil {
		return nil, nil, fmt.Errorf("send v3: %w", err)
	}
	return abiResp, abiBody, nil
}

func (req *Request) mustUseHostCaching() (bool, error) {
	ok, err := fastly.HTTPCacheIsRequestCacheable(req.abi.req)
	return !ok, err
}

// ErrCachingNotSupported is returned by Send() if the requested caching type
// (host or guest) is not supported by the request configuration or runtime.
var ErrCachingNotSupported = errors.New("fsthttp: caching not supported")

func (req *Request) shouldUseGuestCaching() (bool, error) {
	// disabled via buildtags or unsupported hostcalls
	if !useGuestCaching {
		if req.CacheOptions.mustUseGuestCaching() {
			return false, ErrCachingNotSupported
		}

		return false, nil
	}

	if req.CacheOptions.Pass {
		// skip cache
		return false, nil
	}

	if req.Method == "PURGE" {
		// no caching for PURGE
		return false, nil
	}

	if req.ImageOptimizerOptions != nil {
		// request should go through imageopto
		return false, nil
	}

	mustUseHostCaching, err := req.mustUseHostCaching()
	// check for hostcall unsupported error
	if err != nil {
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusUnsupported {
			// disable for future calls
			useGuestCaching = false
			return false, nil
		}
		return false, err
	}

	if mustUseHostCaching {
		if req.CacheOptions.mustUseGuestCaching() {
			// oops, before/after hooks need guest caching.
			return false, ErrCachingNotSupported
		}

		// not cacheable; don't care if there's an error here
		return false, nil
	}

	return true, nil
}

func (req *Request) copyBody(errc chan<- error) {
	var (
		bodyExists   = req.Body != nil
		_, bodyIsABI = req.Body.(*fastly.HTTPBody)
		shouldCopy   = bodyExists && !bodyIsABI
	)

	// if bodyIsABI then we should have set abi.body when SetBody calls abiBodyFrom()

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
	if req.abi.req == nil {

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

		req.abi.req = abiReq
	}

	if req.abi.body == nil {

		abiReqBody, err := abiBodyFrom(req.Body)
		if err != nil {
			return fmt.Errorf("get body: %w", err)
		}
		req.abi.body = abiReqBody
	}

	return nil
}

func (req *Request) setABIRequestOptions() error {
	abiReq := req.abi.req

	if err := abiReq.SetAutoDecompressResponse(fastly.AutoDecompressResponseOptions(req.DecompressResponseOptions)); err != nil {
		return fmt.Errorf("set auto decompress response: %w", err)
	}

	if err := abiReq.SetFramingHeadersMode(req.ManualFramingMode); err != nil {
		return fmt.Errorf("set framing headers mode: %w", err)
	}

	cacheOpts := fastly.CacheOverrideOptions{
		Pass:                 req.CacheOptions.Pass,
		PCI:                  req.CacheOptions.PCI,
		TTL:                  req.CacheOptions.TTL,
		StaleWhileRevalidate: req.CacheOptions.StaleWhileRevalidate,
		SurrogateKey:         req.CacheOptions.SurrogateKey,
	}

	if err := abiReq.SetCacheOverride(cacheOpts); err != nil {
		return fmt.Errorf("set cache options: %w", err)
	}
	for _, key := range req.Header.Keys() {
		vals := req.Header.Values(key)
		if err := abiReq.SetHeaderValues(key, vals); err != nil {
			return fmt.Errorf("set headers: %w", err)
		}
	}

	if key := req.CacheOptions.OverrideKey; key != "" {
		if len(key) != 32 {
			return fmt.Errorf("bad length for OverrideKey: %v != 32", len(key))
		}

		// the header must be 64-byte upper-case hex encoded
		key = strings.ToUpper(hex.EncodeToString([]byte(key)))

		if err := abiReq.SetHeaderValues("fastly-xqd-cache-key", []string{key}); err != nil {
			return fmt.Errorf("set headers cache-key: %w", err)
		}
	}

	return nil
}

// CacheOptions control caching behavior for outgoing requests.
type CacheOptions struct {
	// Pass controls whether or not to force a request to bypass the cache.
	// By default this is false, which means the request will only reach the
	// backend if the request is not normally cacheable (such as a POST), or
	// if a cached response is not available. If pass is set to true, the
	// request will always be sent directly to the backend.
	//
	// Setting Pass to false does not guarantee that a response will be
	// cached for this request, it only allows a caching attempt to be made.
	// For example, a `GET` request may appear cacheable, but if the backend
	// response contains `cache-control: no-store`, the response will not be
	// cached.
	//
	// Pass is mutually exclusive with all other cache options. Setting any
	// other option will change Pass to false.
	Pass bool

	// PCI controls the PCI/HIPAA compliant, non-volatile caching of the
	// request. PCI is false by default, which means the request may not be
	// PCI/HIPAA compliant. If PCI is set to true, caching will be made
	// compliant, and the request will not be forced to bypass the cache.
	//
	// https://docs.fastly.com/products/pci-compliant-caching-and-delivery
	PCI bool

	// TTL represents a Time-to-Live for cached responses to the request, in
	// seconds. If greater than zero, it overrides any behavior specified in the
	// response headers, and the request will not be forced to bypass the cache.
	TTL uint32

	// StaleWhileRevalidate represents a stale-while-revalidate time for the
	// request, in seconds. If greater than zero, it overrides any behavior
	// specified in the response headers, and the request will not be forced to
	// bypass the cache.
	StaleWhileRevalidate uint32

	// SurrogateKey represents an explicit surrogate key for the request, which
	// will be added to any `Surrogate-Key` response headers received from the
	// backend. If nonempty, the request will not be forced to bypass the cache.
	//
	// https://docs.fastly.com/en/guides/purging-api-cache-with-surrogate-keys
	SurrogateKey string

	// Cache key to use in lieu of the automatically-generated cache key based on the request's
	// properties.
	OverrideKey string

	// Sets a callback to be invoked if a request is going all the way to a
	// backend, allowing the request to be modified beforehand.
	//
	// This callback is useful when, for example, a backend requires an
	// additional header to be inserted, but that header is expensive to
	// produce. The callback will only be invoked if the original request
	// cannot be responded to from the cache, so the header is only computed
	// when it is truly needed.
	//
	// NOTE: To enable BeforeSend the build tag fsthttp_guest_cache must be set.
	// Without it, the function will always return an error.
	BeforeSend func(*Request) error

	// Sets a callback to be invoked after a response is returned from a
	// backend, but before it is stored into the cache.
	//
	// This callback allows for cache properties like TTL to be customized
	// beyond what the backend response headers specify. It also allows for the
	// response itself to be modified prior to storing into the cache.
	//
	// NOTE: To enable AfterSend the build tag fsthttp_guest_cache must be set.
	// Without it, the function will always return an error.
	AfterSend func(*CandidateResponse) error
}

func (c *CacheOptions) mustUseGuestCaching() bool {
	return c.BeforeSend != nil || c.AfterSend != nil
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

	// JA3MD5 contains the bytes of the JA3 signature of the client TLS request.
	// See https://www.fastly.com/blog/the-state-of-tls-fingerprinting-whats-working-what-isnt-and-whats-next
	JA3MD5 []byte

	// JA4 contains the bytes of the JA4 signature of the client TLS request.
	// See https://github.com/FoxIO-LLC/ja4/blob/main/technical_details/JA4.md
	JA4 []byte
}

func (req *Request) TLSClientCertificateInfo() (*TLSClientCertificateInfo, error) {
	if req.clientCertificate != nil {
		return req.clientCertificate, nil
	}

	var err error
	var cert TLSClientCertificateInfo

	cert.RawClientCertificate, err = req.downstream.req.DownstreamTLSRawClientCertificate()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("get TLS raw client certificate: %w", err)
	}

	if cert.RawClientCertificate != nil {
		cert.ClientCertIsVerified, err = req.downstream.req.DownstreamTLSClientCertVerifyResult()
		if err != nil {
			return nil, fmt.Errorf("get TLS client certificate verify: %w", err)
		}
	}

	req.clientCertificate = &cert
	return req.clientCertificate, nil
}

type TLSClientCertificateInfo struct {
	// RawClientCertificate contains the bytes of the raw client certificate, if one was provided.
	RawClientCertificate []byte

	// ClientCertIsVerified is true if the provided client certificate is valid.
	ClientCertIsVerified bool
}

// FastlyMeta holds various Fastly-specific metadata for a request.
type FastlyMeta struct {
	// SandboxID is the unique identifier for the sandbox handling the request.
	SandboxID string

	// RequestID is the unique identifier for the request.
	RequestID string

	// H2 is the HTTP/2 fingerprint of a client request if available
	H2 []byte

	// OH is a fingerprint of the client request's original headers
	OH []byte

	// DDOSDetected is true if the request was determined to be part of a DDOS attack.
	DDOSDetected bool

	// FastlyKeyIsValid is true if the request contains a valid Fastly API token.
	// This is for services to restrict authenticating PURGE requests for the readthrough cache.
	FastlyKeyIsValid bool
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

// HandoffWebsocket passes the WebSocket directly to a backend.
//
// This can only be used on services that have the WebSockets feature
// enabled and on requests that are valid WebSocket requests.  The sending
// completes in the background.
//
// Once this method has been called, no other response can be sent to this
// request, and the application can exit without affecting the send.
func (r *Request) HandoffWebsocket(backend string) error {
	return r.downstream.req.HandoffWebsocket(backend)
}

// HandoffFanout passes the request through the Fanout GRIP proxy and on to
// a backend.
//
// This can only be used on services that have the Fanout feature enabled.
//
// The sending completes in the background. Once this method has been
// called, no other response can be sent to this request, and the
// application can exit without affecting the send.
func (r *Request) HandoffFanout(backend string) error {
	return r.downstream.req.HandoffWebsocket(backend)
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
