//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// withAdaptiveBuffer is a helper function that calls the provided function with
// an initial size, and repeats the call with the indicated buffer size when
// initSize is exceeded by the value.
func withAdaptiveBuffer(initSize int, f func(buf *prim.WriteBuffer) FastlyStatus) (*prim.WriteBuffer, error) {
	n := initSize
	for {
		buf := prim.NewWriteBuffer(n)
		status := f(buf)
		if status == FastlyStatusBufLen && buf.NValue() > 0 {
			n = int(buf.NValue())
			continue
		}
		if err := status.toError(); err != nil {
			return nil, err
		}
		return buf, nil
	}
}

// (module $fastly_http_req

// witx:
//
// (@interface func (export "redirect_to_websocket_proxy")
//
//	(param $backend_name string)
//	(result $err (expected (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_req redirect_to_websocket_proxy
//go:noescape
func fastlyHTTPReqRedirectToWebsocketProxy(
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
) FastlyStatus

func HandoffWebsocket(backend string) error {
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqRedirectToWebsocketProxy(
		backendBuffer.Data, backendBuffer.Len,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
// (@interface func (export "redirect_to_grip_proxy")
//
//	(param $backend_name string)
//	(result $err (expected (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_req redirect_to_grip_proxy
//go:noescape
func fastlyHTTPReqRedirectToGripProxy(
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
) FastlyStatus

func HandoffFanout(backend string) error {
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqRedirectToGripProxy(
		backendBuffer.Data, backendBuffer.Len,
	).toError(); err != nil {
		return err
	}

	return nil
}

// HTTPRequest represents an HTTP request.
// The zero value is invalid.
type HTTPRequest struct {
	h requestHandle
}

// witx:
//
//	(@interface func (export "body_downstream_get")
//	  (result $err $fastly_status)
//	  (result $req $request_handle)
//	  (result $body $body_handle)
//	)
//
//go:wasmimport fastly_http_req body_downstream_get
//go:noescape
func fastlyHTTPReqBodyDownstreamGet(
	req prim.Pointer[requestHandle],
	body prim.Pointer[bodyHandle],
) FastlyStatus

// BodyDownstreamGet returns the request and body of the singleton downstream
// request for the current execution.
func BodyDownstreamGet() (*HTTPRequest, *HTTPBody, error) {
	var (
		rh requestHandle = invalidRequestHandle
		bh bodyHandle    = invalidBodyHandle
	)

	if err := fastlyHTTPReqBodyDownstreamGet(
		prim.ToPointer(&rh),
		prim.ToPointer(&bh),
	).toError(); err != nil {
		return nil, nil, err
	}

	return &HTTPRequest{h: rh}, &HTTPBody{h: bh}, nil
}

// witx:
//
//	(@interface func (export "cache_override_set")
//	  (param $h $request_handle)
//	  (param $tag $cache_override_tag)
//	  (param $ttl u32)
//	  (param $stale_while_revalidate u32)
//	  (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req cache_override_set
//go:noescape
//lint:ignore U1000 deprecated in favor of V2
func fastlyHTTPReqCacheOverrideSet(
	h requestHandle,
	tag cacheOverrideTag,
	ttl prim.U32,
	staleWhileRevalidate prim.U32,
) FastlyStatus

// witx:
//
//	(@interface func (export "cache_override_v2_set")
//	  (param $h $request_handle)
//	  (param $tag $cache_override_tag)
//	  (param $ttl u32)
//	  (param $stale_while_revalidate u32)
//	  (param $sk (array u8))
//	  (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req cache_override_v2_set
//go:noescape
func fastlyHTTPReqCacheOverrideV2Set(
	h requestHandle,
	tag cacheOverrideTag,
	ttl prim.U32,
	staleWhileRevalidate prim.U32,
	skData prim.Pointer[prim.U8], skLen prim.Usize,
) FastlyStatus

// SetCacheOverride sets caching-related flags on the request.
func (r *HTTPRequest) SetCacheOverride(options CacheOverrideOptions) error {
	var tag cacheOverrideTag

	if options.Pass {
		tag |= cacheOverrideTagPass
	}

	if options.PCI {
		tag |= cacheOverrideTagPCI
	}

	if options.TTL > 0 {
		tag |= cacheOverrideTagTTL
	}

	if options.StaleWhileRevalidate > 0 {
		tag |= cacheOverrideTagStaleWhileRevalidate
	}

	options_SurrogateKeyBuffer := prim.NewReadBufferFromString(options.SurrogateKey).ArrayU8()

	return fastlyHTTPReqCacheOverrideV2Set(
		r.h,
		tag,
		prim.U32(options.TTL),
		prim.U32(options.StaleWhileRevalidate),
		options_SurrogateKeyBuffer.Data, options_SurrogateKeyBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "new")
//	  (result $err $fastly_status)
//	  (result $h $request_handle)
//	)
//
//go:wasmimport fastly_http_req new
//go:noescape
func fastlyHTTPReqNew(
	h prim.Pointer[requestHandle],
) FastlyStatus

// NewHTTPRequest returns a new, empty HTTP request.
func NewHTTPRequest() (*HTTPRequest, error) {
	var h requestHandle = invalidRequestHandle

	if err := fastlyHTTPReqNew(
		prim.ToPointer(&h),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPRequest{h: h}, nil
}

// witx:
//
//	(@interface func (export "header_names_get")
//	  (param $h $request_handle)
//	  (param $buf (@witx pointer char8))
//	  (param $buf_len (@witx usize))
//	  (param $cursor $multi_value_cursor)
//	  (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	  (param $nwritten_out (@witx pointer (@witx usize)))
//	  (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_names_get
//go:noescape
func fastlyHTTPReqHeaderNamesGet(
	h requestHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderNames returns an iterator that yields the names of each header of
// the request.
func (r *HTTPRequest) GetHeaderNames() *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {

		return fastlyHTTPReqHeaderNamesGet(
			r.h,
			prim.ToPointer(buf), bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
		)
	}

	return newValues(adapter, DefaultLargeBufLen)
}

// witx:
//
//	(@interface func (export "header_value_get")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (param $value (@witx pointer char8))
//	   (param $value_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_value_get
//go:noescape
func fastlyHTTPReqHeaderValueGet(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	value prim.Pointer[prim.Char8],
	valueMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// request, if any.
func (r *HTTPRequest) GetHeaderValue(name string) (string, error) {
	// Most header keys are short: e.g. "Host", "Content-Type", "User-Agent", etc.
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	value, err := withAdaptiveBuffer(DefaultSmallBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPReqHeaderValueGet(
			r.h,
			nameBuffer.Data, nameBuffer.Len,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}

// witx:
//
//	(@interface func (export "header_values_get")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (param $buf (@witx pointer char8))
//	   (param $buf_len (@witx usize))
//	   (param $cursor $multi_value_cursor)
//	   (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_values_get
//go:noescape
func fastlyHTTPReqHeaderValuesGet(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderValues returns an iterator that yields the values for the named
// header that are of the request.
func (r *HTTPRequest) GetHeaderValues(name string) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

		return fastlyHTTPReqHeaderValuesGet(
			r.h,
			nameBuffer.Data, nameBuffer.Len,
			prim.ToPointer(buf), bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
		)
	}

	return newValues(adapter, DefaultLargeBufLen)
}

// witx:
//
//	(@interface func (export "header_values_set")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (param $values (array char8)) ;; contains multiple values separated by \0
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_values_set
//go:noescape
func fastlyHTTPReqHeaderValuesSet(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valuesData prim.Pointer[prim.U8], valuesLen prim.Usize, // multiple values separated by \0
) FastlyStatus

// SetHeaderValues sets the provided header(s) on the request.
func (r *HTTPRequest) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		buf.WriteString(value)
		buf.WriteByte(0)
	}

	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	buf_Bytes_Buffer := prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8()

	return fastlyHTTPReqHeaderValuesSet(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
		buf_Bytes_Buffer.Data, buf_Bytes_Buffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_insert")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (param $value (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_insert
//go:noescape
func fastlyHTTPReqHeaderInsert(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valueData prim.Pointer[prim.U8], valueLen prim.Usize,
) FastlyStatus

// InsertHeader adds the provided header to the request.
func (r *HTTPRequest) InsertHeader(name, value string) error {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	valueBuffer := prim.NewReadBufferFromString(value).ArrayU8()

	return fastlyHTTPReqHeaderInsert(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
		valueBuffer.Data, valueBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_append")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (param $value (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_append
//go:noescape
func fastlyHTTPReqHeaderAppend(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valueData prim.Pointer[prim.U8], valueLen prim.Usize,
) FastlyStatus

// AppendHeader adds the provided header to the request.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPRequest) AppendHeader(name, value string) error {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	valueBuffer := prim.NewReadBufferFromString(value).ArrayU8()

	return fastlyHTTPReqHeaderAppend(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
		valueBuffer.Data, valueBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_remove")
//	   (param $h $request_handle)
//	   (param $name (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req header_remove
//go:noescape
func fastlyHTTPReqHeaderRemove(
	h requestHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
) FastlyStatus

// RemoveHeader removes the named header(s) from the request.
func (r *HTTPRequest) RemoveHeader(name string) error {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	return fastlyHTTPReqHeaderRemove(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "method_get")
//	   (param $h $request_handle)
//	   (param $buf (@witx pointer char8))
//	   (param $buf_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req method_get
//go:noescape
func fastlyHTTPReqMethodGet(
	h requestHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetMethod returns the HTTP method of the request.
func (r *HTTPRequest) GetMethod() (string, error) {
	// HTTP Methods are short: GET, POST, etc.
	value, err := withAdaptiveBuffer(DefaultSmallBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPReqMethodGet(
			r.h,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}

// witx:
//
//	(@interface func (export "method_set")
//	   (param $h $request_handle)
//	   (param $method string)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req method_set
//go:noescape
func fastlyHTTPReqMethodSet(
	h requestHandle,
	methodData prim.Pointer[prim.U8], methodLen prim.Usize,
) FastlyStatus

// SetMethod sets the HTTP method of the request.
func (r *HTTPRequest) SetMethod(method string) error {
	methodBuffer := prim.NewReadBufferFromString(method).Wstring()

	return fastlyHTTPReqMethodSet(
		r.h,
		methodBuffer.Data, methodBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "uri_get")
//	   (param $h $request_handle)
//	   (param $buf (@witx pointer char8))
//	   (param $buf_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req uri_get
//go:noescape
func fastlyHTTPReqURIGet(
	h requestHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetURI returns the fully qualified URI of the request.
func (r *HTTPRequest) GetURI() (string, error) {
	// Longest (unknown); Typically less than 1024, but some browsers accept much longer
	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPReqURIGet(
			r.h,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}

// witx:
//
//	(@interface func (export "uri_set")
//	   (param $h $request_handle)
//	   (param $uri string)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req uri_set
//go:noescape
func fastlyHTTPReqURISet(
	h requestHandle,
	uriData prim.Pointer[prim.U8], uriLen prim.Usize,
) FastlyStatus

// SetURI sets the request's fully qualified URI.
func (r *HTTPRequest) SetURI(uri string) error {
	uriBuffer := prim.NewReadBufferFromString(uri).Wstring()

	return fastlyHTTPReqURISet(
		r.h,
		uriBuffer.Data, uriBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "version_get")
//	   (param $h $request_handle)
//	   (result $err $fastly_status)
//	   (result $version $http_version)
//	)
//
//go:wasmimport fastly_http_req version_get
//go:noescape
func fastlyHTTPReqVersionGet(
	h requestHandle,
	version prim.Pointer[HTTPVersion],
) FastlyStatus

// GetVersion returns the HTTP version of the request.
func (r *HTTPRequest) GetVersion() (proto string, major, minor int, err error) {
	var v HTTPVersion

	if err := fastlyHTTPReqVersionGet(
		r.h,
		prim.ToPointer(&v),
	).toError(); err != nil {
		return "", 0, 0, err
	}

	return v.splat()
}

// witx:
//
//	(@interface func (export "version_set")
//	   (param $h $request_handle)
//	   (param $version $http_version)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req version_set
//go:noescape
func fastlyHTTPReqVersionSet(
	h requestHandle,
	version HTTPVersion,
) FastlyStatus

// SetVersion sets the HTTP version of the request.
func (r *HTTPRequest) SetVersion(v HTTPVersion) error {

	return fastlyHTTPReqVersionSet(
		r.h,
		v,
	).toError()
}

// witx:
//
//	;; The behavior of this method is identical to the original except for the `$error_detail`
//	;; out-parameter.
//	;;
//	;; If the returned `$fastly_status` is OK, `$error_detail` will not be read. Otherwise,
//	;; the status is returned identically to the original `send`, but `$error_detail` is populated.
//	;; Since `$send_error_detail` provides much more granular information about failures, it should
//	;; be used by SDKs as the primary source of error information in favor of `$fastly_status`.
//	;;
//	;; Make sure to initialize `$error_detail` with the full complement of mask values that the
//	;; guest supports. If the corresponding bits in the mask are not set, the host will not populate
//	;; fields in the `$error_detail` struct even if there are values available for those fields.
//	;; This allows forward compatibility when new fields are added.
//	(@interface func (export "send_v2")
//	    (param $h $request_handle)
//	    (param $b $body_handle)
//	    (param $backend string)
//	    (param $error_detail (@witx pointer $send_error_detail))
//	    (result $err (expected
//	            (tuple $response_handle $body_handle)
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req send_v2
//go:noescape
func fastlyHTTPReqSendV2(
	h requestHandle,
	b bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	errDetail prim.Pointer[sendErrorDetail],
	resp prim.Pointer[responseHandle],
	respBody prim.Pointer[bodyHandle],
) FastlyStatus

// Send the request, with the provided body, to the named backend. The body is
// buffered and sent all at once. Blocks until the request is complete, and
// returns the response and response body, or an error.
func (r *HTTPRequest) Send(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		respHandle = invalidResponseHandle
		bodyHandle = invalidBodyHandle
	)

	errDetail := newSendErrorDetail()
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendV2(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&respHandle),
		prim.ToPointer(&bodyHandle),
	).toSendError(errDetail); err != nil {
		return nil, nil, err
	}

	return &HTTPResponse{h: respHandle}, &HTTPBody{h: bodyHandle}, nil
}

// witx:
//
//	;; Like `send_v2`, but does NOT provide caching of any form, and does not set `X-Cache` or
//	;; similar.
//	;;
//	;; This hostcall is intended to ultimately replace `send_v2` as HTTP caching becomes managed
//	;; explicitly at the SDK level.
//	;;
//	;; Any cache override setting on the request is ignored.
//	;;
//	;; Making this a distinct hostcall, rather than a cache override variant, may make it easier
//	;; to tell when support for old styles of send can be safely dropped.
//	(@interface func (export "send_v3")
//	    (param $h $request_handle)
//	    (param $b $body_handle)
//	    (param $backend string)
//	    (param $error_detail (@witx pointer $send_error_detail))
//	    (result $err (expected
//	            (tuple $response_handle $body_handle)
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req send_v3
//go:noescape
func fastlyHTTPReqSendV3(
	h requestHandle,
	b bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	errDetail prim.Pointer[sendErrorDetail],
	resp prim.Pointer[responseHandle],
	respBody prim.Pointer[bodyHandle],
) FastlyStatus

// Send the request, with the provided body, to the named backend. The body is
// buffered and sent all at once. Blocks until the request is complete, and
// returns the response and response body, or an error. Does not set `X-Cache` or similar.
func (r *HTTPRequest) SendV3(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		respHandle = invalidResponseHandle
		bodyHandle = invalidBodyHandle
	)

	errDetail := newSendErrorDetail()
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendV3(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&respHandle),
		prim.ToPointer(&bodyHandle),
	).toSendError(errDetail); err != nil {
		return nil, nil, err
	}

	return &HTTPResponse{h: respHandle}, &HTTPBody{h: bodyHandle}, nil
}

// witx:
//
//	(@interface func (export "send_async")
//	   (param $h $request_handle)
//	   (param $b $body_handle)
//	   (param $backend string)
//	   (result $err $fastly_status)
//	   (result $pending_req $pending_request_handle)
//	)
//
//go:wasmimport fastly_http_req send_async
//go:noescape
func fastlyHTTPReqSendAsync(
	h requestHandle,
	b bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	pendingReq prim.Pointer[pendingRequestHandle],
) FastlyStatus

// PendingRequest is an outstanding or completed asynchronous HTTP request.
// The zero value is invalid.
type PendingRequest struct {
	h pendingRequestHandle
}

// SendAsync sends the request, with the provided body, to the named backend.
// The body is buffered and sent all at once. Returns immediately with a
// reference to the newly created request.
func (r *HTTPRequest) SendAsync(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	var pendingHandle = invalidPendingRequestHandle

	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendAsync(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&pendingHandle),
	).toError(); err != nil {
		return nil, err
	}

	return &PendingRequest{h: pendingHandle}, nil
}

// witx:
//
//	;; Like `send_async`, but does NOT provide caching of any form, and does not set `X-Cache` or
//	;; similar.
//	;;
//	;; Also encompasses `send_async_streaming` by including a streaming flag.
//	;;
//	;; This hostcall is intended to ultimately replace `send_async{_streaming}` as HTTP
//	;; caching becomes managed explicitly at the SDK level.
//	;;
//	;; Any cache override setting on the request is ignored.
//	;;
//	;; Making this a distinct hostcall, rather than a cache override variant, may make it easier
//	;; to tell when support for old styles of send can be safely dropped.
//	(@interface func (export "send_async_v2")
//	    (param $h $request_handle)
//	    (param $b $body_handle)
//	    (param $backend string)
//	    (param $streaming u32)
//	    (result $err (expected $pending_request_handle
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req send_async_v2
//go:noescape
func fastlyHTTPReqSendAsyncV2(
	h requestHandle,
	b bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	streaming prim.U32,
	pendingReq prim.Pointer[pendingRequestHandle],
) FastlyStatus

// SendAsyncV2 sends the request, with the provided body, to the named backend.
// The body is buffered and sent all at once. Returns immediately with a
// reference to the newly created request.  Does not set `X-Cache` or similar.
func (r *HTTPRequest) SendAsyncV2(requestBody *HTTPBody, backend string, streaming bool) (*PendingRequest, error) {
	var pendingHandle = invalidPendingRequestHandle

	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	var streamingU32 prim.U32
	if streaming {
		streamingU32 = 1
	}

	if err := fastlyHTTPReqSendAsyncV2(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		streamingU32,
		prim.ToPointer(&pendingHandle),
	).toError(); err != nil {
		return nil, err
	}

	return &PendingRequest{h: pendingHandle}, nil
}

// witx:
//
//	(@interface func (export "send_async_streaming")
//	   (param $h $request_handle)
//	   (param $b $body_handle)
//	   (param $backend string)
//	   (result $err $fastly_status)
//	   (result $pending_req $pending_request_handle)
//	)
//
//go:wasmimport fastly_http_req send_async_streaming
//go:noescape
func fastlyHTTPReqSendAsyncStreaming(
	h requestHandle,
	b bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	pendingReq prim.Pointer[pendingRequestHandle],
) FastlyStatus

// SendAsyncStreaming sends the request, with the provided body, to the named
// backend. Unlike Send or SendAsync, the request body is streamed, rather than
// buffered and sent all at once. Returns immediately with a reference to the
// newly created request.
func (r *HTTPRequest) SendAsyncStreaming(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	var pendingHandle = invalidPendingRequestHandle

	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendAsyncStreaming(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&pendingHandle),
	).toError(); err != nil {
		return nil, err
	}

	requestBody.closable = true

	return &PendingRequest{h: pendingHandle}, nil
}

// witx:
//
//	(@interface func (export "pending_req_poll_v2")
//	    (param $h $pending_request_handle)
//	    (param $error_detail (@witx pointer $send_error_detail))
//	    (result $err (expected
//	            (tuple $is_done
//	                $response_handle
//	                $body_handle)
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req pending_req_poll_v2
//go:noescape
func fastlyHTTPReqPendingReqPollV2(
	h pendingRequestHandle,
	errDetail prim.Pointer[sendErrorDetail],
	isDone prim.Pointer[prim.U32],
	resp prim.Pointer[responseHandle],
	respBody prim.Pointer[bodyHandle],
) FastlyStatus

// Poll checks to see if the pending request is complete, returning immediately.
// The returned response and response body are valid only if done is true and
// err is nil.
func (r *PendingRequest) Poll() (done bool, response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		respHandle = invalidResponseHandle
		bodyHandle = invalidBodyHandle
		isDone     prim.U32
		errDetail  = newSendErrorDetail()
	)

	if err := fastlyHTTPReqPendingReqPollV2(
		r.h,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&isDone),
		prim.ToPointer(&respHandle),
		prim.ToPointer(&bodyHandle),
	).toSendError(errDetail); err != nil {
		return false, nil, nil, err
	}

	return isDone > 0, &HTTPResponse{h: respHandle}, &HTTPBody{h: bodyHandle}, nil
}

// witx:
//
//	(@interface func (export "pending_req_wait_v2")
//	    (param $h $pending_request_handle)
//	    (param $error_detail (@witx pointer $send_error_detail))
//	    (result $err (expected
//	            (tuple $response_handle $body_handle)
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req pending_req_wait_v2
//go:noescape
func fastlyHTTPReqPendingReqWaitV2(
	h pendingRequestHandle,
	errDetail prim.Pointer[sendErrorDetail],
	resp prim.Pointer[responseHandle],
	respBody prim.Pointer[bodyHandle],
) FastlyStatus

// Wait blocks until the pending request is complete, returning the response and
// response body, or an error.
func (r *PendingRequest) Wait() (response *HTTPResponse, responseBody *HTTPBody, err error) {
	resp, err := NewHTTPResponse()
	if err != nil {
		return nil, nil, fmt.Errorf("response: %w", err)
	}

	respBody, err := NewHTTPBody()
	if err != nil {
		return nil, nil, fmt.Errorf("response body: %w", err)
	}

	errDetail := newSendErrorDetail()

	if err := fastlyHTTPReqPendingReqWaitV2(
		r.h,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&resp.h),
		prim.ToPointer(&respBody.h),
	).toSendError(errDetail); err != nil {
		return nil, nil, err
	}

	return resp, respBody, nil
}

// witx:
//
//	(@interface func (export "auto_decompress_response_set")
//	   (param $h $request_handle)
//	   (param $encodings $content_encodings)
//	   (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req auto_decompress_response_set
//go:noescape
func fastlyAutoDecompressResponseSet(
	h requestHandle,
	encodings contentEncodings,
) FastlyStatus

// SetAutoDecompressResponse set the content encodings to automatically
// decompress responses to this request.
func (r *HTTPRequest) SetAutoDecompressResponse(options AutoDecompressResponseOptions) error {
	var e contentEncodings

	if options.Gzip {
		e |= contentsEncodingsGzip
	}

	return fastlyAutoDecompressResponseSet(
		r.h,
		e,
	).toError()
}

// witx:
//
//	(@interface func (export "framing_headers_mode_set")
//	     (param $h $request_handle)
//	     (param $mode $framing_headers_mode)
//	     (result $err (expected (error $fastly_status)))
//	 )
//
//go:wasmimport fastly_http_req framing_headers_mode_set
//go:noescape
func fastlyHTTPReqSetFramingHeadersMode(
	h requestHandle,
	mode framingHeadersMode,
) FastlyStatus

// SetFramingHeadersMode ?
func (r *HTTPRequest) SetFramingHeadersMode(manual bool) error {
	var mode framingHeadersMode
	if manual {
		mode = framingHeadersModeManuallyFromHeaders
	}

	return fastlyHTTPReqSetFramingHeadersMode(
		r.h,
		mode,
	).toError()
}

// (module $fastly_http_downstream

type HTTPRequestPromise struct {
	h requestPromiseHandle
}

// witx:
//
// ;;; Indicate to the host that we will accept a new request from a client in the future.
// (@interface func (export "next_request")
//
//	(param $options_mask $next_request_options_mask)
//	(param $options (@witx pointer $next_request_options))
//	(result $err (expected $request_promise_handle (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_downstream next_request
//go:noescape
func fastlyHTTPDownstreamNextRequest(
	mask nextRequestOptionsMask,
	opts prim.Pointer[nextRequestOptions],
	promise prim.Pointer[requestPromiseHandle],
) FastlyStatus

func DownstreamNextRequest(opts *NextRequestOptions) (*HTTPRequestPromise, error) {
	var (
		rh requestPromiseHandle = invalidRequestPromiseHandle
	)

	if err := fastlyHTTPDownstreamNextRequest(
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&rh),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPRequestPromise{h: rh}, nil
}

// ;;; Block until an additional request from a client is ready,
// ;;; and return the request and its associated body.
// (@interface func (export "next_request_wait")
//
//	(param $handle $request_promise_handle)
//	(result $err (expected (tuple $request_handle $body_handle) (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_downstream next_request_wait
//go:noescape
func fastlyHTTPDowstreamNextRequestWait(
	p requestPromiseHandle,
	req prim.Pointer[requestHandle],
	body prim.Pointer[bodyHandle],
) FastlyStatus

func (p *HTTPRequestPromise) Wait() (*HTTPRequest, *HTTPBody, error) {
	var (
		rh requestHandle = invalidRequestHandle
		bh bodyHandle    = invalidBodyHandle
	)

	if err := fastlyHTTPDowstreamNextRequestWait(
		p.h,
		prim.ToPointer(&rh),
		prim.ToPointer(&bh),
	).toError(); err != nil {
		return nil, nil, err
	}

	return &HTTPRequest{h: rh}, &HTTPBody{h: bh}, nil
}

// witx:
//
//	;;; Abandon a promised future request. Indicate that we are no longer willing to receive
//	;;; an additional request from a client in the future.
//	(@interface func (export "next_request_abandon")
//	    (param $handle $request_promise_handle)
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream next_request_abandon
//go:noescape
func fastlyHTTPDowstreamNextRequestAbandon(
	p requestPromiseHandle,
) FastlyStatus

func (p *HTTPRequestPromise) Abandon() error {
	if err := fastlyHTTPDowstreamNextRequestAbandon(
		p.h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	(@interface func (export "downstream_original_header_names")
//	    (param $req $request_handle)
//	    (param $buf (@witx pointer (@witx char8)))
//	    (param $buf_len (@witx usize))
//	    (param $cursor $multi_value_cursor)
//	    (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	    (param $nwritten_out (@witx pointer (@witx usize)))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_original_header_names
//go:noescape
func fastlyHTTPDownstreamOriginalHeaderNames(
	req requestHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetOriginalHeaderNames returns an iterator that yields the names of each
// header of the singleton downstream request.
func (req *HTTPRequest) DownstreamOriginalHeaderNames() *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {

		return fastlyHTTPDownstreamOriginalHeaderNames(
			req.h,
			prim.ToPointer(buf), bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
		)
	}

	return newValues(adapter, DefaultLargeBufLen)
}

// witx:
//
// (@interface func (export "downstream_original_header_count")
//
//	(param $req $request_handle)
//	(result $err (expected $header_count (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_downstream downstream_original_header_count
//go:noescape
func fastlyHTTPDownstreamOriginalHeaderCount(
	req requestHandle,
	count prim.Pointer[prim.U32],
) FastlyStatus

// GetOriginalHeaderCount returns the number of headers of the singleton
// downstream request.
func (r *HTTPRequest) DownstreamOriginalHeaderCount() (int, error) {
	var count prim.U32

	if err := fastlyHTTPDownstreamOriginalHeaderCount(
		r.h,
		prim.ToPointer(&count),
	).toError(); err != nil {
		return 0, err
	}

	return int(count), nil
}

// witx:
//
//	(@interface func (export "downstream_client_ip_addr")
//	    (param $req $request_handle)
//	    ;; must be a 16-byte array
//	    (param $addr_octets_out (@witx pointer (@witx char8)))
//	    (result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_client_ip_addr
//go:noescape
func fastlyHTTPDownstreamClientIPAddr(
	req requestHandle,
	addrOctetsOut prim.Pointer[prim.Char8], // ipBufLen is 16 bytes
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamClientIPAddr returns the IP address of the downstream client that
// performed the singleton downstream request.
func (r *HTTPRequest) DownstreamClientIPAddr() (net.IP, error) {
	buf := prim.NewWriteBuffer(ipBufLen)

	if err := fastlyHTTPDownstreamClientIPAddr(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return nil, err
	}

	return net.IP(buf.AsBytes()), nil
}

// witx:
//
//	(@interface func (export "downstream_server_ip_addr")
//	    (param $req $request_handle)
//	    ;; must be a 16-byte array
//	    (param $addr_octets_out (@witx pointer (@witx char8)))
//	    (result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_server_ip_addr
//go:noescape
func fastlyHTTPDownstreamServerIPAddr(
	req requestHandle,
	addrOctetsOut prim.Pointer[prim.Char8], // ipBufLen is 16 bytes
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamServerIPAddr returns the IP address of the downstream server that
// received the HTTP request.
func (r *HTTPRequest) DownstreamServerIPAddr() (net.IP, error) {
	buf := prim.NewWriteBuffer(ipBufLen)

	if err := fastlyHTTPDownstreamServerIPAddr(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return nil, err
	}

	return net.IP(buf.AsBytes()), nil
}

//
//    (@interface func (export "downstream_client_h2_fingerprint")
//        (param $req $request_handle)
//        (param $h2fp_out (@witx pointer (@witx char8)))
//        (param $h2fp_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "downstream_client_request_id")
//        (param $req $request_handle)
//        (param $reqid_out (@witx pointer (@witx char8)))
//        (param $reqid_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "downstream_client_oh_fingerprint")
//        (param $req $request_handle)
//        (param $ohfp_out (@witx pointer (@witx char8)))
//        (param $ohfp_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "downstream_client_ddos_detected")
//        (param $req $request_handle)
//        (result $err (expected $ddos_detected (error $fastly_status)))
//    )

// witx:
//
// (@interface func (export "downstream_tls_cipher_openssl_name")
//
//	(param $req $request_handle)
//	(param $cipher_out (@witx pointer (@witx char8)))
//	(param $cipher_max_len (@witx usize))
//	(param $nwritten_out (@witx pointer (@witx usize)))
//	(result $err (expected (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_http_downstream downstream_tls_cipher_openssl_name
//go:noescape
func fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
	req requestHandle,
	cipherOut prim.Pointer[prim.Char8],
	cipherMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSCipherOpenSSLName returns the name of the OpenSSL TLS cipher
// used with the singleton downstream request, if any.
func (r *HTTPRequest) DownstreamTLSCipherOpenSSLName() (string, error) {
	// https://www.fastly.com/documentation/reference/vcl/variables/client-connection/tls-client-cipher/
	buf := prim.NewWriteBuffer(DefaultSmallBufLen) // Longest (49) = TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256_OLD

	if err := fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "downstream_tls_protocol")
//	    (param $req $request_handle)
//	    (param $protocol_out (@witx pointer (@witx char8)))
//	    (param $protocol_max_len (@witx usize))
//	    (param $nwritten_out (@witx pointer (@witx usize)))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_tls_protocol
//go:noescape
func fastlyHTTPReqDownstreamTLSProtocol(
	req requestHandle,
	protocolOut prim.Pointer[prim.Char8],
	protocolMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSProtocol returns the name of the TLS protocol used with the
// singleton downstream request, if any.
func (r *HTTPRequest) DownstreamTLSProtocol() (string, error) {
	// https://www.fastly.com/documentation/reference/vcl/variables/client-connection/tls-client-protocol/
	buf := prim.NewWriteBuffer(DefaultSmallBufLen) // Longest (~8) = TLSv1.2

	if err := fastlyHTTPReqDownstreamTLSProtocol(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "downstream_tls_client_hello")
//	    (param $req $request_handle)
//	    (param $chello_out (@witx pointer (@witx char8)))
//	    (param $chello_max_len (@witx usize))
//	    (param $nwritten_out (@witx pointer (@witx usize)))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_tls_client_hello
//go:noescape
func fastlyHTTPReqDownstreamTLSClientHello(
	req requestHandle,
	chelloOut prim.Pointer[prim.Char8],
	chelloMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSClientHello returns the ClientHello message sent by the client
// in the singleton downstream request, if any.
func (r *HTTPRequest) DownstreamTLSClientHello() ([]byte, error) {
	n := DefaultLargeBufLen // Longest (~132,000); typically < 2^14; RFC https://datatracker.ietf.org/doc/html/rfc8446#section-4.1.2
	for {
		buf := prim.NewWriteBuffer(n)
		status := fastlyHTTPReqDownstreamTLSClientHello(
			r.h,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
		if status == FastlyStatusBufLen && buf.NValue() > 0 {
			n = int(buf.NValue())
			continue
		}
		if err := status.toError(); err != nil {
			return nil, err
		}
		return buf.AsBytes(), nil
	}
}

//
//    (@interface func (export "downstream_tls_raw_client_certificate")
//        (param $req $request_handle)
//        (param $raw_client_cert_out (@witx pointer (@witx char8)))
//        (param $raw_client_cert_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "downstream_tls_client_cert_verify_result")
//        (param $req $request_handle)
//        (result $err (expected $client_cert_verify_result (error $fastly_status)))
//    )

// witx:
//
//	(@interface func (export "downstream_tls_ja3_md5")
//	    (param $req $request_handle)
//	    ;; must be a 16-byte array
//	    (param $cja3_md5_out (@witx pointer (@witx char8)))
//	    (result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_downstream downstream_tls_ja3_md5
//go:noescape
func fastlyHTTPReqDownstreamTLSJA3MD5(
	req requestHandle,
	cJA3MD5Out prim.Pointer[prim.Char8],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSJA3MD5 returns the MD5 []byte representing the JA3 signature of the singleton downstream request, if any.
func (r *HTTPRequest) DownstreamTLSJA3MD5() ([]byte, error) {
	var p [16]byte
	buf := prim.NewWriteBufferFromBytes(p[:])
	err := fastlyHTTPReqDownstreamTLSJA3MD5(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		prim.ToPointer(buf.NPointer()),
	).toError()
	if err != nil {
		return nil, err
	}
	return buf.AsBytes(), nil
}

//
//    (@interface func (export "downstream_tls_ja4")
//        (param $req $request_handle)
//        (param $ja4_out (@witx pointer (@witx char8)))
//        (param $ja4_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "downstream_compliance_region")
//        (param $req $request_handle)
//        (param $region_out (@witx pointer (@witx char8)))
//        (param $region_max_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//
//    (@interface func (export "fastly_key_is_valid")
//        (param $req $request_handle)
//        (result $err (expected $is_valid (error $fastly_status)))
//    )
//)

// (module $fastly_http_body

// HTTPBody represents the body of an HTTP request or response.
// The zero value is invalid.
type HTTPBody struct {
	h bodyHandle

	// Closing an HTTP body is only possible if the encapsulated body handle has
	// its "streaming bit" set. The streaming bit is set when the handle is
	// successfully passed to send_async_streaming or send_downstream with
	// streaming set to 1. The streaming bit is unqueryable, and we need to be
	// able to abstract over different concrete bodies. So we try to mirror that
	// hidden state in the body handle with this visible state in the struct,
	// and use it to check if it's safe to close the handle.
	closable bool
}

// witx:
//
//	(@interface func (export "append")
//	  (param $dest $body_handle)
//	  (param $src $body_handle)
//	  (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_body append
//go:noescape
func fastlyHTTPBodyAppend(
	dest bodyHandle,
	src bodyHandle,
) FastlyStatus

// Append the other body to this one.
func (b *HTTPBody) Append(other *HTTPBody) error {

	if err := fastlyHTTPBodyAppend(
		b.h,
		other.h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	(@interface func (export "new")
//	  (result $err $fastly_status)
//	  (result $h $body_handle)
//	)
//
//go:wasmimport fastly_http_body new
//go:noescape
func fastlyHTTPBodyNew(
	h prim.Pointer[bodyHandle],
) FastlyStatus

// NewHTTPBody returns a new, empty HTTP body.
func NewHTTPBody() (*HTTPBody, error) {
	var b = invalidBodyHandle

	if err := fastlyHTTPBodyNew(
		prim.ToPointer(&b),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPBody{h: b}, nil
}

// witx:
//
//	(@interface func (export "read")
//	  (param $h $body_handle)
//	  (param $buf (@witx pointer u8))
//	  (param $buf_len (@witx usize))
//	  (result $err $fastly_status)
//	  (result $nread (@witx usize))
//	)
//
//go:wasmimport fastly_http_body read
//go:noescape
func fastlyHTTPBodyRead(
	h bodyHandle,
	buf prim.Pointer[prim.U8],
	bufLen prim.Usize,
	nRead prim.Pointer[prim.Usize],
) FastlyStatus

// Read implements io.Reader, reading up to len(p) bytes from the body into p.
// Returns the number of bytes read, and any error encountered.
func (b *HTTPBody) Read(p []byte) (int, error) {
	buf := prim.NewWriteBufferFromBytes(p)

	if err := fastlyHTTPBodyRead(
		b.h,
		prim.ToPointer(buf.U8Pointer()),
		buf.Len(), // can't assume len(p) == cap(p)
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return 0, err
	}

	n := buf.NValue()
	if n == 0 {
		return 0, io.EOF
	}

	return int(n), nil
}

// witx:
//
//	(@interface func (export "write")
//	  (param $h $body_handle)
//	  (param $buf (array u8))
//	  (param $end $body_write_end)
//	  (result $err $fastly_status)
//	  (result $nwritten (@witx usize))
//	)
//
//go:wasmimport fastly_http_body write
//go:noescape
func fastlyHTTPBodyWrite(
	h bodyHandle,
	bufData prim.Pointer[prim.U8], bufLen prim.Usize,
	end bodyWriteEnd,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the body.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (b *HTTPBody) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		p_n_Buffer := prim.NewReadBufferFromBytes(p[n:]).ArrayU8()

		if err = fastlyHTTPBodyWrite(
			b.h,
			p_n_Buffer.Data, p_n_Buffer.Len,
			bodyWriteEndBack,
			prim.ToPointer(&nWritten),
		).toError(); err == nil {
			n += int(nWritten)
		}
	}
	return n, err
}

// witx:
//
//	;;; Frees the body on the host.
//	;;;
//	;;; For streaming bodies, this is a _successful_ stream termination, which will signal
//	;;; via framing that the body transfer is complete.
//	(@interface func (export "close")
//	  (param $h $body_handle)
//	  (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_body close
//go:noescape
func fastlyHTTPBodyClose(
	h bodyHandle,
) FastlyStatus

// Close the body. This indicates a successful end of the stream, as
// opposed to the Abandon method. Once closed, a body cannot be used again.
// Close is a no-op unless the body's "streaming bit" is set.
func (b *HTTPBody) Close() error {
	if !b.closable {
		return nil
	}

	return fastlyHTTPBodyClose(
		b.h,
	).toError()
}

// witx:
//
//	;;; Frees a streaming body on the host _unsuccessfully_, so that framing makes clear that
//	;;; the body is incomplete.
//	(@interface func (export "abandon")
//	  (param $h $body_handle)
//	  (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_body abandon
//go:noescape
func fastlyHTTPBodyAbandon(
	h bodyHandle,
) FastlyStatus

// Abandon the body. This indicates an unsuccessful end of the stream,
// as opposed to the Close method. Once closed, a body cannot be used again.
// Abandon is a no-op unless the body's "streaming bit" is set.
func (b *HTTPBody) Abandon() error {
	if !b.closable {
		return nil
	}

	return fastlyHTTPBodyAbandon(
		b.h,
	).toError()
}

// witx:
//
//	;;; Returns a u64 body length if the length of a body is known, or `FastlyStatus::None`
//	;;; otherwise.
//	;;;
//	;;; If the length is unknown, it is likely due to the body arising from an HTTP/1.1 message with
//	;;; chunked encoding, an HTTP/2 or later message with no `content-length`, or being a streaming
//	;;; body.
//	;;;
//	;;; Note that receiving a length from this function does not guarantee that the full number of
//	;;; bytes can actually be read from the body. For example, when proxying a response from a
//	;;; backend, this length may reflect the `content-length` promised in the response, but if the
//	;;; backend connection is closed prematurely, fewer bytes may be delivered before this body
//	;;; handle can no longer be read.
//	(@interface func (export "known_length")
//	    (param $h $body_handle)
//	    (result $err (expected $body_length (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_body known_length
//go:noescape
func fastlyHTTPBodyKnownLength(
	h bodyHandle,
	l prim.Pointer[prim.U64],
) FastlyStatus

// Length returns the size in bytes of the http body, if known.
//
// The length of the cached item may be unknown if the item is currently being streamed into
// the cache without a fixed length.
func (b *HTTPBody) Length() (uint64, error) {

	var l prim.U64

	if err := fastlyHTTPBodyKnownLength(
		b.h,
		prim.ToPointer(&l),
	).toError(); err != nil {
		return 0, err
	}

	return uint64(l), nil
}

// witx:
//
//	(module $fastly_http_resp
//	   (@interface func (export "new")
//	     (result $err $fastly_status)
//	     (result $h $response_handle)
//	   )
//
//go:wasmimport fastly_http_resp new
//go:noescape
func fastlyHTTPRespNew(
	h prim.Pointer[responseHandle],
) FastlyStatus

// HTTPResponse represents a response to an HTTP request.
// The zero value is invalid.
type HTTPResponse struct {
	h responseHandle
}

// NewHTTPREsponse returns a valid, empty HTTP response.
func NewHTTPResponse() (*HTTPResponse, error) {
	var respHandle = invalidResponseHandle

	if err := fastlyHTTPRespNew(
		prim.ToPointer(&respHandle),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPResponse{h: respHandle}, nil
}

// witx:
//
//	;; The following directly mirror header & version methods on req
//
//	(@interface func (export "header_names_get")
//	  (param $h $response_handle)
//	  (param $buf (@witx pointer char8))
//	  (param $buf_len (@witx usize))
//	  (param $cursor $multi_value_cursor)
//	  (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	  (param $nwritten_out (@witx pointer (@witx usize)))
//	  (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_names_get
//go:noescape
func fastlyHTTPRespHeaderNamesGet(
	h responseHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderNames returns an iterator that yields the names of each header of
// the response.
func (r *HTTPResponse) GetHeaderNames() *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {

		return fastlyHTTPRespHeaderNamesGet(
			r.h,
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
		)
	}

	return newValues(adapter, DefaultLargeBufLen)
}

// witx:
//
//	(@interface func (export "header_value_get")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (param $value (@witx pointer char8))
//	   (param $value_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_value_get
//go:noescape
func fastlyHTTPRespHeaderValueGet(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	value prim.Pointer[prim.Char8],
	valueMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// response, if any.
func (r *HTTPResponse) GetHeaderValue(name string) (string, error) {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	value, err := withAdaptiveBuffer(DefaultLargeBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPRespHeaderValueGet(
			r.h,
			nameBuffer.Data, nameBuffer.Len,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}

// witx:
//
//	(@interface func (export "header_values_get")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (param $buf (@witx pointer char8))
//	   (param $buf_len (@witx usize))
//	   (param $cursor $multi_value_cursor)
//	   (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_values_get
//go:noescape
func fastlyHTTPRespHeaderValuesGet(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GetHeaderValues returns an iterator that yields the values for the named
// header that are of the response.
func (r *HTTPResponse) GetHeaderValues(name string) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

		return fastlyHTTPRespHeaderValuesGet(
			r.h,
			nameBuffer.Data, nameBuffer.Len,
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
		)
	}

	return newValues(adapter, DefaultLargeBufLen)
}

// witx:
//
//	(@interface func (export "header_values_set")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (param $values (array char8)) ;; contains multiple values separated by \0
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_values_set
//go:noescape
func fastlyHTTPRespHeaderValuesSet(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valuesData prim.Pointer[prim.U8], valuesLen prim.Usize, // multiple values separated by \0
) FastlyStatus

// SetHeaderValues sets the provided header(s) on the response.
//
// TODO(pb): does this overwrite any existing name headers?
func (r *HTTPResponse) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		buf.WriteString(value)
		buf.WriteByte(0)
	}

	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	buf_Bytes_Buffer := prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8()

	return fastlyHTTPRespHeaderValuesSet(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
		buf_Bytes_Buffer.Data, buf_Bytes_Buffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_insert")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (param $value (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_insert
//go:noescape
func fastlyHTTPRespHeaderInsert(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valueData prim.Pointer[prim.U8], valueLen prim.Usize,
) FastlyStatus

// InsertHeader adds the provided header to the response.
func (r *HTTPResponse) InsertHeader(name, value string) error {
	var (
		nameBuf  = prim.NewReadBufferFromString(name)
		valueBuf = prim.NewReadBufferFromString(value)
	)
	nameBufArrayU8 := nameBuf.ArrayU8()
	valueBufArrayU8 := valueBuf.ArrayU8()

	return fastlyHTTPRespHeaderInsert(
		r.h,
		nameBufArrayU8.Data, nameBufArrayU8.Len,
		valueBufArrayU8.Data, valueBufArrayU8.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_append")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (param $value (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_append
//go:noescape
func fastlyHTTPRespHeaderAppend(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	valueData prim.Pointer[prim.U8], valueLen prim.Usize,
) FastlyStatus

// AppendHeader adds the provided header to the response.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPResponse) AppendHeader(name, value string) error {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()
	valueBuffer := prim.NewReadBufferFromString(value).ArrayU8()

	return fastlyHTTPRespHeaderAppend(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
		valueBuffer.Data, valueBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "header_remove")
//	   (param $h $response_handle)
//	   (param $name (array u8))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp header_remove
//go:noescape
func fastlyHTTPRespHeaderRemove(
	h responseHandle,
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
) FastlyStatus

// RemoveHeader removes the named header(s) from the response.
func (r *HTTPResponse) RemoveHeader(name string) error {
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	return fastlyHTTPRespHeaderRemove(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
	).toError()
}

// witx:
//
//	(@interface func (export "version_get")
//	   (param $h $response_handle)
//	   (result $err $fastly_status)
//	   (result $version $http_version)
//	)
//
//go:wasmimport fastly_http_resp version_get
//go:noescape
func fastlyHTTPRespVersionGet(
	h responseHandle,
	version prim.Pointer[HTTPVersion],
) FastlyStatus

// GetVersion returns the HTTP version of the request.
func (r *HTTPResponse) GetVersion() (proto string, major, minor int, err error) {
	var v HTTPVersion

	if err := fastlyHTTPRespVersionGet(
		r.h,
		prim.ToPointer(&v),
	).toError(); err != nil {
		return "", 0, 0, err
	}

	return v.splat()
}

// witx:
//
//	(@interface func (export "version_set")
//	   (param $h $response_handle)
//	   (param $version $http_version)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp version_set
//go:noescape
func fastlyHTTPRespVersionSet(
	h responseHandle,
	version HTTPVersion,
) FastlyStatus

// SetVersion sets the HTTP version of the response.
func (r *HTTPResponse) SetVersion(v HTTPVersion) error {

	return fastlyHTTPRespVersionSet(
		r.h,
		v,
	).toError()
}

// witx:
//
//	(@interface func (export "send_downstream")
//	   (param $h $response_handle)
//	   (param $b $body_handle)
//	   (param $streaming u32)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp send_downstream
//go:noescape
func fastlyHTTPRespSendDownstream(
	h responseHandle,
	b bodyHandle,
	streaming prim.U32,
) FastlyStatus

// SendDownstream sends the response, with the provided body, to the implicit
// downstream of the current execution. If stream is true, the response body is
// streamed to the downstream, rather than being buffered and sent all at once.
func (r *HTTPResponse) SendDownstream(responseBody *HTTPBody, stream bool) error {
	var streaming prim.U32
	if stream {
		streaming = 1
	}

	if err := fastlyHTTPRespSendDownstream(
		r.h,
		responseBody.h,
		streaming,
	).toError(); err != nil {
		return err
	}

	if stream {
		responseBody.closable = true
	}

	return nil
}

// witx:
//
//	(@interface func (export "status_get")
//	   (param $h $response_handle)
//	   (result $err $fastly_status)
//	   (result $status $http_status)
//	)
//
//go:wasmimport fastly_http_resp status_get
//go:noescape
func fastlyHTTPRespStatusGet(
	h responseHandle,
	status prim.Pointer[httpStatus],
) FastlyStatus

// GetStatusCode returns the status code of the response.
func (r *HTTPResponse) GetStatusCode() (int, error) {
	var status httpStatus

	if err := fastlyHTTPRespStatusGet(
		r.h,
		prim.ToPointer(&status),
	).toError(); err != nil {
		return 0, err
	}

	return int(status), nil
}

// witx:
//
//	(@interface func (export "status_set")
//	   (param $h $response_handle)
//	   (param $status $http_status)
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_resp status_set
//go:noescape
func fastlyHTTPRespStatusSet(
	h responseHandle,
	status httpStatus,
) FastlyStatus

// SetStatusCode sets the status code of the response.
func (r *HTTPResponse) SetStatusCode(code int) error {
	status := httpStatus(code)

	return fastlyHTTPRespStatusSet(
		r.h,
		status,
	).toError()
}

// witx:
//
//	(@interface func (export "framing_headers_mode_set")
//	     (param $h $response_handle)
//	     (param $mode $framing_headers_mode)
//	     (result $err (expected (error $fastly_status)))
//	 )
//
//go:wasmimport fastly_http_resp framing_headers_mode_set
//go:noescape
func fastlyHTTPRespSetFramingHeadersMode(
	h responseHandle,
	mode framingHeadersMode,
) FastlyStatus

// SetFramingHeadersMode ?
func (r *HTTPResponse) SetFramingHeadersMode(manual bool) error {
	var mode framingHeadersMode
	if manual {
		mode = framingHeadersModeManuallyFromHeaders
	}

	return fastlyHTTPRespSetFramingHeadersMode(
		r.h,
		mode,
	).toError()
}

// witx:
//
//	;;; Hostcall for getting the destination IP used for this request.
//	;;;
//	;;; The buffer for the IP address must be 16 bytes.
//	;;; syntax used in URLs as specified in RFC 3986 section 3.
//	(@interface func (export "get_addr_dest_ip")
//		(param $h $response_handle)
//		;; must be a 16-byte array
//		(param $addr_octets_out (@witx pointer (@witx char8)))
//		(result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_resp get_addr_dest_ip
//go:noescape
func fastlyHTTPRespGetAddrDestIP(
	h responseHandle,
	addr prim.Pointer[prim.Char8],
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

// GetAddrDestIP
func (r *HTTPResponse) GetAddrDestIP() (net.IP, error) {
	buf := prim.NewWriteBuffer(ipBufLen)

	if err := fastlyHTTPRespGetAddrDestIP(
		r.h,
		prim.ToPointer(buf.Char8Pointer()),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return nil, err
	}

	return net.IP(buf.AsBytes()), nil
}

// witx:
//
//	;;; Hostcall for getting the destination port used for this request.
//	(@interface func (export "get_addr_dest_port")
//		(param $h $response_handle)
//		(result $err (expected $port (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_resp get_addr_dest_port
//go:noescape
func fastlyHTTPRespGetAddrDestPort(
	h responseHandle,
	port prim.Pointer[prim.U16],
) FastlyStatus

// GetAddrDestPort
func (r *HTTPResponse) GetAddrDestPort() (uint16, error) {
	var port prim.U16

	if err := fastlyHTTPRespGetAddrDestPort(
		r.h, prim.ToPointer(&port),
	).toError(); err != nil {
		return 0, err
	}

	return uint16(port), nil
}

// witx:
//
//	(module $fastly_uap
//	 (@interface func (export "parse")
//	   (param $user_agent string)
//
//	   (param $family (@witx pointer char8))
//	   (param $family_len (@witx usize))
//	   (param $family_nwritten_out (@witx pointer (@witx usize)))
//
//	   (param $major (@witx pointer char8))
//	   (param $major_len (@witx usize))
//	   (param $major_nwritten_out (@witx pointer (@witx usize)))
//
//	   (param $minor (@witx pointer char8))
//	   (param $minor_len (@witx usize))
//	   (param $minor_nwritten_out (@witx pointer (@witx usize)))
//
//	   (param $patch (@witx pointer char8))
//	   (param $patch_len (@witx usize))
//	   (param $patch_nwritten_out (@witx pointer (@witx usize)))
//
//	   (result $err $fastly_status)
//	 )
//
//go:wasmimport fastly_uap parse
//go:noescape
func fastlyUAPParse(
	userAgentData prim.Pointer[prim.U8], userAgentLen prim.Usize,

	family prim.Pointer[prim.Char8],
	familyLen prim.Usize,
	familyNWrittenOut prim.Pointer[prim.Usize],

	major prim.Pointer[prim.Char8],
	majorLen prim.Usize,
	majorNWrittenOut prim.Pointer[prim.Usize],

	minor prim.Pointer[prim.Char8],
	minorLen prim.Usize,
	minorNWrittenOut prim.Pointer[prim.Usize],

	patch prim.Pointer[prim.Char8],
	patchLen prim.Usize,
	patchNWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// ParseUserAgent parses the user agent string into its component parts.
func ParseUserAgent(userAgent string) (family, major, minor, patch string, err error) {
	var (
		cap       = len(userAgent)
		familyBuf = prim.NewWriteBuffer(cap)
		majorBuf  = prim.NewWriteBuffer(cap)
		minorBuf  = prim.NewWriteBuffer(cap)
		patchBuf  = prim.NewWriteBuffer(cap)
	)

	userAgentBuffer := prim.NewReadBufferFromString(userAgent).Wstring()

	if err := fastlyUAPParse(
		userAgentBuffer.Data, userAgentBuffer.Len,

		prim.ToPointer(familyBuf.Char8Pointer()),
		familyBuf.Cap(),
		prim.ToPointer(familyBuf.NPointer()),

		prim.ToPointer(majorBuf.Char8Pointer()),
		majorBuf.Cap(),
		prim.ToPointer(majorBuf.NPointer()),

		prim.ToPointer(minorBuf.Char8Pointer()),
		minorBuf.Cap(),
		prim.ToPointer(minorBuf.NPointer()),

		prim.ToPointer(patchBuf.Char8Pointer()),
		patchBuf.Cap(),
		prim.ToPointer(patchBuf.NPointer()),
	).toError(); err != nil {
		return "", "", "", "", err
	}

	return familyBuf.ToString(), majorBuf.ToString(), minorBuf.ToString(), patchBuf.ToString(), nil
}
