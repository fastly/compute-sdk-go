//go:build tinygo.wasm && wasi && !nofastlyhostcalls
// +build tinygo.wasm,wasi,!nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//	(module $fastly_abi
//	 (@interface func (export "init")
//	   (param $abi_version u64)
//	   (result $err $fastly_status))
//	)
//
//go:wasm-module fastly_abi
//export init
//go:noescape
func fastlyABIInit(abiVersion prim.U64) FastlyStatus

// Initialize the Fastly ABI at the given version.
func Initialize(version uint64) error {
	return fastlyABIInit(prim.U64(version)).toError()
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
//go:wasm-module fastly_uap
//export parse
//go:noescape
func fastlyUAPParse(
	userAgent prim.Wstring,

	family *prim.Char8,
	familyLen prim.Usize,
	familyNWrittenOut *prim.Usize,

	major *prim.Char8,
	majorLen prim.Usize,
	majorNWrittenOut *prim.Usize,

	minor *prim.Char8,
	minorLen prim.Usize,
	minorNWrittenOut *prim.Usize,

	patch *prim.Char8,
	patchLen prim.Usize,
	patchNWrittenOut *prim.Usize,
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

	if err := fastlyUAPParse(
		prim.NewReadBufferFromString(userAgent).Wstring(),

		familyBuf.Char8Pointer(),
		familyBuf.Cap(),
		familyBuf.NPointer(),

		majorBuf.Char8Pointer(),
		majorBuf.Cap(),
		majorBuf.NPointer(),

		minorBuf.Char8Pointer(),
		minorBuf.Cap(),
		minorBuf.NPointer(),

		patchBuf.Char8Pointer(),
		patchBuf.Cap(),
		patchBuf.NPointer(),
	).toError(); err != nil {
		return "", "", "", "", err
	}

	return familyBuf.ToString(), majorBuf.ToString(), minorBuf.ToString(), patchBuf.ToString(), nil
}

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
//go:wasm-module fastly_http_body
//export append
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
//go:wasm-module fastly_http_body
//export new
//go:noescape
func fastlyHTTPBodyNew(
	h *bodyHandle,
) FastlyStatus

// NewHTTPBody returns a new, empty HTTP body.
func NewHTTPBody() (*HTTPBody, error) {
	var b HTTPBody

	if err := fastlyHTTPBodyNew(
		&b.h,
	).toError(); err != nil {
		return nil, err
	}

	return &b, nil
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
//go:wasm-module fastly_http_body
//export read
//go:noescape
func fastlyHTTPBodyRead(
	h bodyHandle,
	buf *prim.U8,
	bufLen prim.Usize,
	nRead *prim.Usize,
) FastlyStatus

// Read implements io.Reader, reading up to len(p) bytes from the body into p.
// Returns the number of bytes read, and any error encountered.
func (b *HTTPBody) Read(p []byte) (int, error) {
	buf := prim.NewWriteBufferFromBytes(p)
	if err := fastlyHTTPBodyRead(
		b.h,
		buf.U8Pointer(),
		buf.Len(), // can't assume len(p) == cap(p)
		buf.NPointer(),
	).toError(); err != nil {
		return 0, err
	}

	n := int(buf.NValue())
	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
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
//go:wasm-module fastly_http_body
//export write
//go:noescape
func fastlyHTTPBodyWrite(
	h bodyHandle,
	buf prim.ArrayU8,
	end bodyWriteEnd,
	nWritten *prim.Usize,
) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the body.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (b *HTTPBody) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		if err = fastlyHTTPBodyWrite(
			b.h,
			prim.NewReadBufferFromBytes(p[n:]).ArrayU8(),
			bodyWriteEndBack,
			&nWritten,
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
//go:wasm-module fastly_http_body
//export close
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
//go:wasm-module fastly_http_body
//export abandon
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

// (module $fastly_log

// LogEndpoint represents a specific Fastly log endpoint.
type LogEndpoint struct {
	h endpointHandle
}

// witx:
//
//	(@interface func (export "endpoint_get")
//	  (param $name (array u8))
//	  (result $err $fastly_status)
//	  (result $endpoint_handle_out $endpoint_handle))
//
//go:wasm-module fastly_log
//export endpoint_get
//go:noescape
func fastlyLogEndpointGet(
	name prim.ArrayU8,
	endpointHandleOut *endpointHandle,
) FastlyStatus

// GetLogEndpoint opens the log endpoint identified by name.
func GetLogEndpoint(name string) (*LogEndpoint, error) {
	var e LogEndpoint

	if err := fastlyLogEndpointGet(
		prim.NewReadBufferFromString(name).ArrayU8(),
		&e.h,
	).toError(); err != nil {
		return nil, err
	}

	return &e, nil
}

// witx:
//
//	(@interface func (export "write")
//	  (param $h $endpoint_handle)
//	  (param $msg (array u8))
//	  (result $err $fastly_status)
//	  (result $nwritten_out (@witx usize)))
//
// )
//
//go:wasm-module fastly_log
//export write
//go:noescape
func fastlyLogWrite(
	h endpointHandle,
	msg prim.ArrayU8,
	nWrittenOut *prim.Usize,
) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the endpoint.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (e *LogEndpoint) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		if err = fastlyLogWrite(
			e.h,
			prim.NewReadBufferFromBytes(p[n:]).ArrayU8(),
			&nWritten,
		).toError(); err == nil {
			n += int(nWritten)
		}
	}
	return n, err
}

// (module $fastly_http_req

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
//go:wasm-module fastly_http_req
//export body_downstream_get
//go:noescape
func fastlyHTTPReqBodyDownstreamGet(
	req *requestHandle,
	body *bodyHandle,
) FastlyStatus

// BodyDownstreamGet returns the request and body of the singleton downstream
// request for the current execution.
func BodyDownstreamGet() (*HTTPRequest, *HTTPBody, error) {
	var (
		rh requestHandle = requestHandle(math.MaxUint32 - 1)
		bh bodyHandle    = bodyHandle(math.MaxUint32 - 1)
	)
	if err := fastlyHTTPReqBodyDownstreamGet(
		&rh,
		&bh,
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
//go:wasm-module fastly_http_req
//export cache_override_set
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
//go:wasm-module fastly_http_req
//export cache_override_v2_set
//go:noescape
func fastlyHTTPReqCacheOverrideV2Set(
	h requestHandle,
	tag cacheOverrideTag,
	ttl prim.U32,
	staleWhileRevalidate prim.U32,
	sk prim.ArrayU8,
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

	return fastlyHTTPReqCacheOverrideV2Set(
		r.h,
		tag,
		prim.U32(options.TTL),
		prim.U32(options.StaleWhileRevalidate),
		prim.NewReadBufferFromString(options.SurrogateKey).ArrayU8(),
	).toError()
}

// witx:
//
//	(@interface func (export "downstream_client_ip_addr")
//	   ;; must be a 16-byte array
//	   (param $addr_octets_out (@witx pointer char8))
//	   (result $err $fastly_status)
//	   (result $nwritten_out (@witx usize))
//	)
//
//go:wasm-module fastly_http_req
//export downstream_client_ip_addr
//go:noescape
func fastlyHTTPReqDownstreamClientIPAddr(
	addrOctetsOut *prim.Char8, // must be 16-byte array
	nwrittenOut *prim.Usize,
) FastlyStatus

// DownstreamClientIPAddr returns the IP address of the downstream client that
// performed the singleton downstream request.
func DownstreamClientIPAddr() (net.IP, error) {
	buf := prim.NewWriteBuffer(16) // must be a 16-byte array
	if err := fastlyHTTPReqDownstreamClientIPAddr(
		buf.Char8Pointer(),
		buf.NPointer(),
	).toError(); err != nil {
		return nil, err
	}

	return net.IP(buf.AsBytes()), nil
}

// witx:
//
//	(@interface func (export "downstream_tls_cipher_openssl_name")
//	   (param $cipher_out (@witx pointer char8))
//	   (param $cipher_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export downstream_tls_cipher_openssl_name
//go:noescape
func fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
	cipherOut *prim.Char8,
	cipherMaxLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// DownstreamTLSCipherOpenSSLName returns the name of the OpenSSL TLS cipher
// used with the singleton downstream request, if any.
func DownstreamTLSCipherOpenSSLName() (string, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)
	if err := fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "downstream_tls_protocol")
//	   (param $protocol_out (@witx pointer char8))
//	   (param $protocol_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export downstream_tls_protocol
//go:noescape
func fastlyHTTPReqDownstreamTLSProtocol(
	protocolOut *prim.Char8,
	protocolMaxLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// DownstreamTLSProtocol returns the name of the TLS protocol used with the
// singleton downstream request, if any.
func DownstreamTLSProtocol() (string, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)
	if err := fastlyHTTPReqDownstreamTLSProtocol(
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "downstream_tls_client_hello")
//	   (param $chello_out (@witx pointer char8))
//	   (param $chello_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export downstream_tls_client_hello
//go:noescape
func fastlyHTTPReqDownstreamTLSClientHello(
	chelloOut *prim.Char8,
	chelloMaxLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// DownstreamTLSClientHello returns the ClientHello message sent by the client
// in the singleton downstream request, if any.
func DownstreamTLSClientHello() ([]byte, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)
	if err := fastlyHTTPReqDownstreamTLSClientHello(
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return nil, err
	}

	return buf.AsBytes(), nil
}

// witx:
//
//	(@interface func (export "new")
//	  (result $err $fastly_status)
//	  (result $h $request_handle)
//	)
//
//go:wasm-module fastly_http_req
//export new
//go:noescape
func fastlyHTTPReqNew(
	h *requestHandle,
) FastlyStatus

// NewHTTPRequest returns a new, empty HTTP request.
func NewHTTPRequest() (*HTTPRequest, error) {
	var r HTTPRequest

	if err := fastlyHTTPReqNew(
		&r.h,
	).toError(); err != nil {
		return nil, err
	}

	return &r, nil
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
//go:wasm-module fastly_http_req
//export header_names_get
//go:noescape
func fastlyHTTPReqHeaderNamesGet(
	h requestHandle,
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderNames returns an iterator that yields the names of each header of
// the request.
func (r *HTTPRequest) GetHeaderNames(maxHeaderNameLen int) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		return fastlyHTTPReqHeaderNamesGet(
			r.h,
			buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut,
		)
	}

	return newValues(adapter, maxHeaderNameLen)
}

// witx:
//
//	(@interface func (export "original_header_names_get")
//	  (param $buf (@witx pointer char8))
//	  (param $buf_len (@witx usize))
//	  (param $cursor $multi_value_cursor)
//	  (param $ending_cursor_out (@witx pointer $multi_value_cursor_result))
//	  (param $nwritten_out (@witx pointer (@witx usize)))
//	  (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export original_header_names_get
//go:noescape
func fastlyHTTPReqOriginalHeaderNamesGet(
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetOriginalHeaderNames returns an iterator that yields the names of each
// header of the singleton downstream request.
func GetOriginalHeaderNames(maxHeaderNameLen int) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		return fastlyHTTPReqOriginalHeaderNamesGet(
			buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut,
		)
	}

	return newValues(adapter, maxHeaderNameLen)
}

// witx:
//
//	(@interface func (export "original_header_count")
//	  (result $err $fastly_status)
//	  (result $count u32)
//	)
//
//go:wasm-module fastly_http_req
//export original_header_count
//go:noescape
func fastlyHTTPReqOriginalHeaderCount(
	count *prim.U32,
) FastlyStatus

// GetOriginalHeaderCount returns the number of headers of the singleton
// downstream request.
func GetOriginalHeaderCount() (int, error) {
	var count prim.U32
	if err := fastlyHTTPReqOriginalHeaderCount(
		&count,
	).toError(); err != nil {
		return 0, err
	}

	return int(count), nil
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
//go:wasm-module fastly_http_req
//export header_value_get
//go:noescape
func fastlyHTTPReqHeaderValueGet(
	h requestHandle,
	name prim.ArrayU8,
	value *prim.Char8,
	valueMaxLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// request, if any.
func (r *HTTPRequest) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	if err := fastlyHTTPReqHeaderValueGet(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
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
//go:wasm-module fastly_http_req
//export header_values_get
//go:noescape
func fastlyHTTPReqHeaderValuesGet(
	h requestHandle,
	name prim.ArrayU8,
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderValues returns an iterator that yields the values for the named
// header that are of the request.
func (r *HTTPRequest) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		return fastlyHTTPReqHeaderValuesGet(
			r.h,
			prim.NewReadBufferFromString(name).ArrayU8(),
			buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut,
		)
	}

	return newValues(adapter, maxHeaderValueLen)
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
//go:wasm-module fastly_http_req
//export header_values_set
//go:noescape
func fastlyHTTPReqHeaderValuesSet(
	h requestHandle,
	name prim.ArrayU8,
	values prim.ArrayChar8, // multiple values separated by \0
) FastlyStatus

// SetHeaderValues sets the provided header(s) on the request.
func (r *HTTPRequest) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		fmt.Fprint(&buf, value)
		buf.WriteByte(0)
	}

	return fastlyHTTPReqHeaderValuesSet(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8(),
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
//go:wasm-module fastly_http_req
//export header_insert
//go:noescape
func fastlyHTTPReqHeaderInsert(
	h requestHandle,
	name prim.ArrayU8,
	value prim.ArrayU8,
) FastlyStatus

// InsertHeader adds the provided header to the request.
func (r *HTTPRequest) InsertHeader(name, value string) error {
	return fastlyHTTPReqHeaderInsert(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		prim.NewReadBufferFromString(value).ArrayU8(),
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
//go:wasm-module fastly_http_req
//export header_append
//go:noescape
func fastlyHTTPReqHeaderAppend(
	h requestHandle,
	name prim.ArrayU8,
	value prim.ArrayU8,
) FastlyStatus

// AppendHeader adds the provided header to the request.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPRequest) AppendHeader(name, value string) error {
	return fastlyHTTPReqHeaderAppend(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		prim.NewReadBufferFromString(value).ArrayU8(),
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
//go:wasm-module fastly_http_req
//export header_remove
//go:noescape
func fastlyHTTPReqHeaderRemove(
	h requestHandle,
	name prim.ArrayU8,
) FastlyStatus

// RemoveHeader removes the named header(s) from the request.
func (r *HTTPRequest) RemoveHeader(name string) error {
	return fastlyHTTPReqHeaderRemove(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
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
//go:wasm-module fastly_http_req
//export method_get
//go:noescape
func fastlyHTTPReqMethodGet(
	h requestHandle,
	buf *prim.Char8,
	bufLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetMethod returns the HTTP method of the request.
func (r *HTTPRequest) GetMethod(maxMethodLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxMethodLen)
	if err := fastlyHTTPReqMethodGet(
		r.h,
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "method_set")
//	   (param $h $request_handle)
//	   (param $method string)
//	   (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export method_set
//go:noescape
func fastlyHTTPReqMethodSet(
	h requestHandle,
	method prim.Wstring,
) FastlyStatus

// SetMethod sets the HTTP method of the request.
func (r *HTTPRequest) SetMethod(method string) error {
	return fastlyHTTPReqMethodSet(
		r.h,
		prim.NewReadBufferFromString(method).Wstring(),
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
//go:wasm-module fastly_http_req
//export uri_get
//go:noescape
func fastlyHTTPReqURIGet(
	h requestHandle,
	buf *prim.Char8,
	bufLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetURI returns the fully qualified URI of the request.
func (r *HTTPRequest) GetURI(maxURLLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxURLLen)
	if err := fastlyHTTPReqURIGet(
		r.h,
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "uri_set")
//	   (param $h $request_handle)
//	   (param $uri string)
//	   (result $err $fastly_status)
//	)
//
//go:wasm-module fastly_http_req
//export uri_set
//go:noescape
func fastlyHTTPReqURISet(
	h requestHandle,
	uri prim.Wstring,
) FastlyStatus

// SetURI sets the request's fully qualified URI.
func (r *HTTPRequest) SetURI(uri string) error {
	return fastlyHTTPReqURISet(
		r.h,
		prim.NewReadBufferFromString(uri).Wstring(),
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
//go:wasm-module fastly_http_req
//export version_get
//go:noescape
func fastlyHTTPReqVersionGet(
	h requestHandle,
	version *HTTPVersion,
) FastlyStatus

// GetVersion returns the HTTP version of the request.
func (r *HTTPRequest) GetVersion() (proto string, major, minor int, err error) {
	var v HTTPVersion
	if err := fastlyHTTPReqVersionGet(
		r.h,
		&v,
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
//go:wasm-module fastly_http_req
//export version_set
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
//	(@interface func (export "send")
//	   (param $h $request_handle)
//	   (param $b $body_handle)
//	   (param $backend string)
//	   (result $err $fastly_status)
//	   (result $resp $response_handle)
//	   (result $resp_body $body_handle)
//	)
//
//go:wasm-module fastly_http_req
//export send
//go:noescape
func fastlyHTTPReqSend(
	h requestHandle,
	b bodyHandle,
	backend prim.Wstring,
	resp *responseHandle,
	respBody *bodyHandle,
) FastlyStatus

// Send the request, with the provided body, to the named backend. The body is
// buffered and sent all at once. Blocks until the request is complete, and
// returns the response and response body, or an error.
func (r *HTTPRequest) Send(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		resp     HTTPResponse
		respBody HTTPBody
	)

	if err := fastlyHTTPReqSend(
		r.h,
		requestBody.h,
		prim.NewReadBufferFromString(backend).Wstring(),
		&resp.h,
		&respBody.h,
	).toError(); err != nil {
		return nil, nil, err
	}

	return &resp, &respBody, nil
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
//go:wasm-module fastly_http_req
//export send_async
//go:noescape
func fastlyHTTPReqSendAsync(
	h requestHandle,
	b bodyHandle,
	backend prim.Wstring,
	pendingReq *pendingRequestHandle,
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
	var pendingReq PendingRequest

	if err := fastlyHTTPReqSendAsync(
		r.h,
		requestBody.h,
		prim.NewReadBufferFromString(backend).Wstring(),
		&pendingReq.h,
	).toError(); err != nil {
		return nil, err
	}

	return &pendingReq, nil
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
//go:wasm-module fastly_http_req
//export send_async_streaming
//go:noescape
func fastlyHTTPReqSendAsyncStreaming(
	h requestHandle,
	b bodyHandle,
	backend prim.Wstring,
	pendingReq *pendingRequestHandle,
) FastlyStatus

// SendAsyncStreaming sends the request, with the provided body, to the named
// backend. Unlike Send or SendAsync, the request body is streamed, rather than
// buffered and sent all at once. Returns immediately with a reference to the
// newly created request.
func (r *HTTPRequest) SendAsyncStreaming(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	var pendingReq PendingRequest

	if err := fastlyHTTPReqSendAsyncStreaming(
		r.h,
		requestBody.h,
		prim.NewReadBufferFromString(backend).Wstring(),
		&pendingReq.h,
	).toError(); err != nil {
		return nil, err
	}

	requestBody.closable = true

	return &pendingReq, nil
}

// witx:
//
//	(@interface func (export "pending_req_poll")
//	   (param $h $pending_request_handle)
//	   (result $err $fastly_status)
//	   (result $is_done u32)
//	   (result $resp $response_handle)
//	   (result $resp_body $body_handle)
//	)
//
//go:wasm-module fastly_http_req
//export pending_req_poll
//go:noescape
func fastlyHTTPReqPendingReqPoll(
	h pendingRequestHandle,
	isDone *prim.U32,
	resp *responseHandle,
	respBody *bodyHandle,
) FastlyStatus

// Poll checks to see if the pending request is complete, returning immediately.
// The returned response and response body are valid only if done is true and
// err is nil.
func (r *PendingRequest) Poll() (done bool, response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		resp     HTTPResponse
		respBody HTTPBody
		isDone   prim.U32
	)

	if err := fastlyHTTPReqPendingReqPoll(
		r.h,
		&isDone,
		&resp.h,
		&respBody.h,
	).toError(); err != nil {
		return false, nil, nil, err
	}

	return isDone > 0, &resp, &respBody, nil
}

// witx:
//
//	(@interface func (export "pending_req_wait")
//	   (param $h $pending_request_handle)
//	   (result $err $fastly_status)
//	   (result $resp $response_handle)
//	   (result $resp_body $body_handle)
//	)
//
//go:wasm-module fastly_http_req
//export pending_req_wait
//go:noescape
func fastlyHTTPReqPendingReqWait(
	h pendingRequestHandle,
	resp *responseHandle,
	respBody *bodyHandle,
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

	if err := fastlyHTTPReqPendingReqWait(
		r.h,
		&resp.h,
		&respBody.h,
	).toError(); err != nil {
		return nil, nil, err
	}

	return resp, respBody, nil
}

// witx:
//
//	(@interface func (export "pending_req_select")
//	   (param $hs (array $pending_request_handle))
//	   (result $err $fastly_status)
//	   (result $done_idx u32)
//	   (result $resp $response_handle)
//	   (result $resp_body $body_handle)
//	)
//
//go:wasm-module fastly_http_req
//export pending_req_select
//go:noescape
func fastlyHTTPReqPendingReqSelect(
	hs []pendingRequestHandle, // TODO(pb): is correct?
	doneIdx *prim.U32,
	resp *responseHandle,
	respBody *bodyHandle,
) FastlyStatus

// PendingRequestSelect blocks until one of the provided pending requests is
// complete. Returns the completed request, and its associated response and
// response body. If more than one pending request is complete, returns one of
// them randomly.
//
// TODO(pb): is random correct?
func PendingRequestSelect(reqs ...*PendingRequest) (index int, done *PendingRequest, response *HTTPResponse, responseBody *HTTPBody, err error) {
	resp, err := NewHTTPResponse()
	if err != nil {
		return 0, nil, nil, nil, fmt.Errorf("response: %w", err)
	}

	respBody, err := NewHTTPBody()
	if err != nil {
		return 0, nil, nil, nil, fmt.Errorf("response body: %w", err)
	}

	hs := make([]pendingRequestHandle, len(reqs))
	for i := range reqs {
		hs[i] = reqs[i].h
	}

	var doneIdx prim.U32
	if err := fastlyHTTPReqPendingReqSelect(
		hs,
		&doneIdx,
		&resp.h,
		&respBody.h,
	).toError(); err != nil {
		return 0, nil, nil, nil, err
	}

	return int(doneIdx), reqs[doneIdx], resp, respBody, nil
}

// witx:
//
//	(@interface func (export "auto_decompress_response_set")
//	   (param $h $request_handle)
//	   (param $encodings $content_encodings)
//	   (result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_http_req
//export auto_decompress_response_set
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
//go:wasm-module fastly_http_req
//export framing_headers_mode_set
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

// witx:
//
// (@interface func (export "redirect_to_websocket_proxy")
//
//	(param $backend_name string)
//	(result $err (expected (error $fastly_status)))
//
// )
//
//go:wasm-module fastly_http_req
//export redirect_to_websocket_proxy
//go:noescape
func fastlyHTTPReqRedirectToWebsocketProxy(
	backend prim.Wstring,
) FastlyStatus

func HandoffWebsocket(backend string) error {
	if err := fastlyHTTPReqRedirectToWebsocketProxy(
		prim.NewReadBufferFromString(backend).Wstring(),
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
//go:wasm-module fastly_http_req
//export redirect_to_grip_proxy
//go:noescape
func fastlyHTTPReqRedirectToGripProxy(
	backend prim.Wstring,
) FastlyStatus

func HandoffFanout(backend string) error {
	if err := fastlyHTTPReqRedirectToGripProxy(
		prim.NewReadBufferFromString(backend).Wstring(),
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	(@interface func (export "register_dynamic_backend")
//		(param $name_prefix string)
//		(param $target string)
//		(param $backend_config_mask $backend_config_options)
//		(param $backend_configuration (@witx pointer $dynamic_backend_config))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_http_req
//export register_dynamic_backend
//go:noescape
func fastlyRegisterDynamicBackend(name prim.Wstring, target prim.Wstring, mask backendConfigOptionsMask, opts *backendConfigOptions) FastlyStatus

func RegisterDynamicBackend(name string, target string, opts *BackendConfigOptions) error {
	if err := fastlyRegisterDynamicBackend(
		prim.NewReadBufferFromString(name).Wstring(),
		prim.NewReadBufferFromString(target).Wstring(),
		opts.mask,
		&opts.opts,
	).toError(); err != nil {
		return err
	}
	return nil
}

// witx:
//
//	(module $fastly_backend
//		(@interface func (export "exists")
//			(param $backend string)
//			(result $err (expected
//			$backend_exists
//			(error $fastly_status)))
//		)
//
//go:wasm-module fastly_backend
//export exists
//go:noescape
func fastlyBackendExists(name prim.Wstring, exists *prim.U32) FastlyStatus

func BackendExists(name string) (bool, error) {
	var exists prim.U32
	if err := fastlyBackendExists(
		prim.NewReadBufferFromString(name).Wstring(),
		&exists,
	).toError(); err != nil {
		return false, err
	}
	return exists != 0, nil
}

// witx:
//
//	(@interface func (export "is_healthy")
//		(param $backend string)
//		(result $err (expected
//		$backend_health
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export is_healthy
//go:noescape
func fastlyBackendIsHealthy(name prim.Wstring, healthy *prim.U32) FastlyStatus

func BackendIsHealthy(name string) (BackendHealth, error) {
	var health prim.U32
	if err := fastlyBackendIsHealthy(
		prim.NewReadBufferFromString(name).Wstring(),
		&health,
	).toError(); err != nil {
		return BackendHealthUnknown, err
	}
	return BackendHealth(health), nil
}

// witx:
//
//	(@interface func (export "is_dynamic")
//		(param $backend string)
//		(result $err (expected
//		is_dyanmic
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export is_dynamic
//go:noescape
func fastlyBackendIsDynamic(name prim.Wstring, dynamic *prim.U32) FastlyStatus

func BackendIsDynamic(name string) (bool, error) {
	var dynamic prim.U32
	if err := fastlyBackendIsDynamic(
		prim.NewReadBufferFromString(name).Wstring(),
		&dynamic,
	).toError(); err != nil {
		return false, err
	}
	return dynamic != 0, nil
}

// witx:
//
//	(@interface func (export "get_host")
//		(param $backend string)
//		(param $value (@witx pointer (@witx char8)))
//		(param $value_max_len (@witx usize))
//		(param $nwritten_out (@witx pointer (@witx usize)))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_host
//go:noescape
func fastlyBackendGetHost(name prim.Wstring,
	host *prim.Char8,
	hostLen prim.Usize,
	hostWritten *prim.Usize,
) FastlyStatus

func BackendGetHost(name string) (string, error) {
	hostBuf := prim.NewWriteBuffer(defaultBufferLen)

	if err := fastlyBackendGetHost(
		prim.NewReadBufferFromString(name).Wstring(),

		hostBuf.Char8Pointer(),
		hostBuf.Cap(),
		hostBuf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return hostBuf.ToString(), nil
}

// witx:
//
//	(@interface func (export "get_override_host")
//		(param $backend string)
//		(param $value (@witx pointer (@witx char8)))
//		(param $value_max_len (@witx usize))
//		(param $nwritten_out (@witx pointer (@witx usize)))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_override_host
//go:noescape
func fastlyBackendGetOverrideHost(name prim.Wstring,
	host *prim.Char8,
	hostLen prim.Usize,
	hostWritten *prim.Usize,
) FastlyStatus

func BackendGetOverrideHost(name string) (string, error) {
	hostBuf := prim.NewWriteBuffer(defaultBufferLen)

	if err := fastlyBackendGetOverrideHost(
		prim.NewReadBufferFromString(name).Wstring(),
		hostBuf.Char8Pointer(),
		hostBuf.Cap(),
		hostBuf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return hostBuf.ToString(), nil
}

// witx:
//
//	(@interface func (export "get_port")
//		(param $backend string)
//		(result $err (expected
//		$port
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_port
//go:noescape
func fastlyBackendGetPort(name prim.Wstring, port *prim.U32) FastlyStatus

func BackendGetPort(name string) (int, error) {
	var port prim.U32
	if err := fastlyBackendGetPort(
		prim.NewReadBufferFromString(name).Wstring(),
		&port,
	).toError(); err != nil {
		return 0, err
	}
	return int(port), nil
}

// witx:
//
//	(@interface func (export "get_connect_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_connect_timeout_ms
//go:noescape
func fastlyBackendGetConnectTimeoutMs(name prim.Wstring, timeout *prim.U32) FastlyStatus

func BackendGetConnectTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	if err := fastlyBackendGetConnectTimeoutMs(
		prim.NewReadBufferFromString(name).Wstring(),
		&timeout,
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "get_first_byte_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_first_byte_timeout_ms
//go:noescape
func fastlyBackendGetFirstByteTimeoutMs(name prim.Wstring, timeout *prim.U32) FastlyStatus

func BackendGetFirstByteTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	if err := fastlyBackendGetFirstByteTimeoutMs(
		prim.NewReadBufferFromString(name).Wstring(),
		&timeout,
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "get_between_bytes_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_between_bytes_timeout_ms
//go:noescape
func fastlyBackendGetBetweenBytesTimeoutMs(name prim.Wstring, timeout *prim.U32) FastlyStatus

func BackendGetBetweenBytesTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	if err := fastlyBackendGetBetweenBytesTimeoutMs(
		prim.NewReadBufferFromString(name).Wstring(),
		&timeout,
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "is_ssl")
//		(param $backend string)
//		(result $err (expected
//		is_ssl
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export is_ssl
//go:noescape
func fastlyBackendIsSSL(name prim.Wstring, ssl *prim.U32) FastlyStatus

func BackendIsSSL(name string) (bool, error) {
	var ssl prim.U32
	if err := fastlyBackendIsSSL(
		prim.NewReadBufferFromString(name).Wstring(),
		&ssl,
	).toError(); err != nil {
		return false, err
	}
	return ssl != 0, nil
}

// witx:
//
//	(@interface func (export "get_ssl_min_version")
//		(param $backend string)
//		(result $err (expected
//		$tls_version
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_ssl_min_version
//go:noescape
func fastlyBackendGetSSLMinVersion(name prim.Wstring, version *prim.U32) FastlyStatus

func BackendGetSSLMinVersion(name string) (TLSVersion, error) {
	var version prim.U32
	if err := fastlyBackendGetSSLMinVersion(
		prim.NewReadBufferFromString(name).Wstring(),
		&version,
	).toError(); err != nil {
		return 0, err
	}
	return TLSVersion(version), nil
}

// witx:
//
//	(@interface func (export "get_ssl_max_version")
//		(param $backend string)
//		(result $err (expected
//		$tls_version
//		(error $fastly_status)))
//	)
//
//go:wasm-module fastly_backend
//export get_ssl_max_version
//go:noescape
func fastlyBackendGetSSLMaxVersion(name prim.Wstring, version *prim.U32) FastlyStatus

func BackendGetSSLMaxVersion(name string) (TLSVersion, error) {
	var version prim.U32
	if err := fastlyBackendGetSSLMaxVersion(
		prim.NewReadBufferFromString(name).Wstring(),
		&version,
	).toError(); err != nil {
		return 0, err
	}
	return TLSVersion(version), nil
}

// witx:
//
//	(module $fastly_http_resp
//	   (@interface func (export "new")
//	     (result $err $fastly_status)
//	     (result $h $response_handle)
//	   )
//
//go:wasm-module fastly_http_resp
//export new
//go:noescape
func fastlyHTTPRespNew(
	h *responseHandle,
) FastlyStatus

// HTTPResponse represents a response to an HTTP request.
// The zero value is invalid.
type HTTPResponse struct {
	h responseHandle
}

// NewHTTPREsponse returns a valid, empty HTTP response.
func NewHTTPResponse() (*HTTPResponse, error) {
	var resp HTTPResponse

	if err := fastlyHTTPRespNew(
		&resp.h,
	).toError(); err != nil {
		return nil, err
	}

	return &resp, nil
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
//go:wasm-module fastly_http_resp
//export header_names_get
//go:noescape
func fastlyHTTPRespHeaderNamesGet(
	h responseHandle,
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderNames returns an iterator that yields the names of each header of
// the response.
func (r *HTTPResponse) GetHeaderNames(maxHeaderNameLen int) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		return fastlyHTTPRespHeaderNamesGet(
			r.h,
			buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut,
		)
	}

	return newValues(adapter, maxHeaderNameLen)
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
//go:wasm-module fastly_http_resp
//export header_value_get
//go:noescape
func fastlyHTTPRespHeaderValueGet(
	h responseHandle,
	name prim.ArrayU8,
	value *prim.Char8,
	valueMaxLen prim.Usize,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// response, if any.
func (r *HTTPResponse) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	if err := fastlyHTTPRespHeaderValueGet(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
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
//go:wasm-module fastly_http_resp
//export header_values_get
//go:noescape
func fastlyHTTPRespHeaderValuesGet(
	h responseHandle,
	name prim.ArrayU8,
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// GetHeaderValues returns an iterator that yields the values for the named
// header that are of the response.
func (r *HTTPResponse) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
	adapter := func(
		buf *prim.Char8,
		bufLen prim.Usize,
		cursor multiValueCursor,
		endingCursorOut *multiValueCursorResult,
		nwrittenOut *prim.Usize,
	) FastlyStatus {
		return fastlyHTTPRespHeaderValuesGet(
			r.h,
			prim.NewReadBufferFromString(name).ArrayU8(),
			buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut,
		)
	}

	return newValues(adapter, maxHeaderValueLen)
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
//go:wasm-module fastly_http_resp
//export header_values_set
//go:noescape
func fastlyHTTPRespHeaderValuesSet(
	h responseHandle,
	name prim.ArrayU8,
	values prim.ArrayChar8, // multiple values separated by \0
) FastlyStatus

// SetHeaderValues sets the provided header(s) on the response.
//
// TODO(pb): does this overwrite any existing name headers?
func (r *HTTPResponse) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		fmt.Fprint(&buf, value)
		buf.WriteByte(0)
	}

	return fastlyHTTPRespHeaderValuesSet(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8(),
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
//go:wasm-module fastly_http_resp
//export header_insert
//go:noescape
func fastlyHTTPRespHeaderInsert(
	h responseHandle,
	name prim.ArrayU8,
	value prim.ArrayU8,
) FastlyStatus

// InsertHeader adds the provided header to the response.
func (r *HTTPResponse) InsertHeader(name, value string) error {
	var (
		nameBuf  = prim.NewReadBufferFromString(name)
		valueBuf = prim.NewReadBufferFromString(value)
	)
	return fastlyHTTPRespHeaderInsert(
		r.h,
		nameBuf.ArrayU8(),
		valueBuf.ArrayU8(),
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
//go:wasm-module fastly_http_resp
//export header_append
//go:noescape
func fastlyHTTPRespHeaderAppend(
	h responseHandle,
	name prim.ArrayU8,
	value prim.ArrayU8,
) FastlyStatus

// AppendHeader adds the provided header to the response.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPResponse) AppendHeader(name, value string) error {
	return fastlyHTTPRespHeaderAppend(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
		prim.NewReadBufferFromString(value).ArrayU8(),
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
//go:wasm-module fastly_http_resp
//export header_remove
//go:noescape
func fastlyHTTPRespHeaderRemove(
	h responseHandle,
	name prim.ArrayU8,
) FastlyStatus

// RemoveHeader removes the named header(s) from the response.
func (r *HTTPResponse) RemoveHeader(name string) error {
	return fastlyHTTPRespHeaderRemove(
		r.h,
		prim.NewReadBufferFromString(name).ArrayU8(),
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
//go:wasm-module fastly_http_resp
//export version_get
//go:noescape
func fastlyHTTPRespVersionGet(
	h responseHandle,
	version *HTTPVersion,
) FastlyStatus

// GetVersion returns the HTTP version of the request.
func (r *HTTPResponse) GetVersion() (proto string, major, minor int, err error) {
	var v HTTPVersion
	if err := fastlyHTTPRespVersionGet(
		r.h,
		&v,
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
//go:wasm-module fastly_http_resp
//export version_set
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
//go:wasm-module fastly_http_resp
//export send_downstream
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
//go:wasm-module fastly_http_resp
//export status_get
//go:noescape
func fastlyHTTPRespStatusGet(
	h responseHandle,
	status *httpStatus,
) FastlyStatus

// GetStatusCode returns the status code of the response.
func (r *HTTPResponse) GetStatusCode() (int, error) {
	var status httpStatus
	if err := fastlyHTTPRespStatusGet(
		r.h,
		&status,
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
//go:wasm-module fastly_http_resp
//export status_set
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
//go:wasm-module fastly_http_resp
//export framing_headers_mode_set
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
//	(module $fastly_dictionary
//	   (@interface func (export "open")
//	      (param $name string)
//	      (result $err $fastly_status)
//	      (result $h $dictionary_handle)
//	   )
//
//go:wasm-module fastly_dictionary
//export open
//go:noescape
func fastlyDictionaryOpen(
	name prim.Wstring,
	h *dictionaryHandle,
) FastlyStatus

// Dictionary represents a Fastly edge dictionary, a collection of read-only
// key/value pairs. For convenience, keys are modeled as Go strings, and values
// as byte slices.
type Dictionary struct {
	h dictionaryHandle
}

// OpenDictionary returns a reference to the named dictionary, if it exists.
func OpenDictionary(name string) (*Dictionary, error) {
	var d Dictionary

	if err := fastlyDictionaryOpen(
		prim.NewReadBufferFromString(name).Wstring(),
		&d.h,
	).toError(); err != nil {
		return nil, err
	}

	return &d, nil
}

// witx:
//
//	(@interface func (export "get")
//	   (param $h $dictionary_handle)
//	   (param $key string)
//	   (param $value (@witx pointer char8))
//	   (param $value_max_len (@witx usize))
//	   (result $err $fastly_status)
//	   (result $nwritten (@witx usize))
//	)
//
//go:wasm-module fastly_dictionary
//export get
//go:noescape
func fastlyDictionaryGet(
	h dictionaryHandle,
	key prim.Wstring,
	value *prim.Char8,
	valueMaxLen prim.Usize,
	nWritten *prim.Usize,
) FastlyStatus

// Get the value for key, if it exists.
func (d *Dictionary) Get(key string) (string, error) {
	buf := prim.NewWriteBuffer(dictionaryValueMaxLen)
	if err := fastlyDictionaryGet(
		d.h,
		prim.NewReadBufferFromString(key).Wstring(),
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(module $fastly_geo
//	  (@interface func (export "lookup")
//	     (param $addr_octets (@witx pointer (@witx char8)))
//	     (param $addr_len (@witx usize))
//	     (param $buf (@witx pointer (@witx char8)))
//	     (param $buf_len (@witx usize))
//	     (param $nwritten_out (@witx pointer (@witx usize)))
//	     (result $err (expected (error $fastly_status)))
//	  )
//
// )
//
//go:wasm-module fastly_geo
//export lookup
//go:noescape
func fastlyGeoLookup(
	addrOctets *prim.Char8,
	addLen prim.Usize,
	buf *prim.Char8,
	bufLen prim.Usize,
	nWrittenOut *prim.Usize,
) FastlyStatus

// GeoLookup returns the geographic data associated with the IP address.
func GeoLookup(ip net.IP) ([]byte, error) {
	buf := prim.NewWriteBuffer(1024) // initial geo buf size
	if x := ip.To4(); x != nil {
		ip = x
	}
	addrOctets := prim.NewReadBufferFromBytes(ip)
	if err := fastlyGeoLookup(
		addrOctets.Char8Pointer(),
		addrOctets.Len(),
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	).toError(); err != nil {
		return nil, err
	}

	return buf.AsBytes(), nil
}

// witx:
//
//   (module $fastly_object_store
//	   (@interface func (export "open")
//	     (param $name string)
//	     (result $err (expected $object_store_handle (error $fastly_status)))
//	  )

//go:wasm-module fastly_object_store
//export open
//go:noescape
func fastlyObjectStoreOpen(
	name prim.Wstring,
	h *objectStoreHandle,
) FastlyStatus

// objectStore represents a Fastly kv store, a collection of key/value pairs.
// For convenience, keys and values are both modelled as Go strings.
type KVStore struct {
	h objectStoreHandle
}

// KVStoreOpen returns a reference to the named kv store, if it exists.
func OpenKVStore(name string) (*KVStore, error) {
	var o KVStore

	if err := fastlyObjectStoreOpen(
		prim.NewReadBufferFromString(name).Wstring(),
		&o.h,
	).toError(); err != nil {
		return nil, err
	}

	return &o, nil
}

// witx:
//
//   (@interface func (export "lookup")
//	   (param $store $object_store_handle)
//	   (param $key string)
//	   (param $body_handle_out (@witx pointer $body_handle))
//	   (result $err (expected (error $fastly_status)))
//	)

//go:wasm-module fastly_object_store
//export lookup
//go:noescape
func fastlyObjectStoreLookup(
	h objectStoreHandle,
	key prim.Wstring,
	b *bodyHandle,
) FastlyStatus

// Lookup returns the value for key, if it exists.
func (o *KVStore) Lookup(key string) (io.Reader, error) {
	body := HTTPBody{h: invalidBodyHandle}

	if err := fastlyObjectStoreLookup(
		o.h,
		prim.NewReadBufferFromString(key).Wstring(),
		&body.h,
	).toError(); err != nil {
		return nil, err
	}

	// Didn't get a valid handle back.  This means there was no key
	// with that name.  Report this to the caller by returning `None`.
	if body.h == invalidBodyHandle {
		return nil, FastlyError{Status: FastlyStatusNone}
	}

	return &body, nil
}

// witx:
//
//  (@interface func (export "insert")
//	  (param $store $object_store_handle)
//	  (param $key string)
//	  (param $body_handle $body_handle)
//	  (result $err (expected (error $fastly_status)))
//	)

//go:wasm-module fastly_object_store
//export insert
//go:noescape
func fastlyObjectStoreInsert(
	h objectStoreHandle,
	key prim.Wstring,
	b bodyHandle,
) FastlyStatus

// Insert adds a key/value pair to the kv store.
func (o *KVStore) Insert(key string, value io.Reader) error {
	body, err := NewHTTPBody()
	if err != nil {
		return err
	}

	if _, err := io.Copy(body, value); err != nil {
		return err
	}

	if err := fastlyObjectStoreInsert(
		o.h,
		prim.NewReadBufferFromString(key).Wstring(),
		body.h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// SecretStore represents a Fastly secret store, a collection of
// key/value pairs for storing sensitive data.
type SecretStore struct {
	h secretStoreHandle
}

// Secret represents a secret value.  Data is encrypted at rest, and is
// only decrypted upon the first call to the secret's Plaintext method.
type Secret struct {
	h secretHandle
}

// witx:
//
//   (module $fastly_secret_store
//	   (@interface func (export "open")
//	     (param $name string)
//	     (result $err (expected $secret_store_handle (error $fastly_status)))
//	  )

//go:wasm-module fastly_secret_store
//export open
//go:noescape
func fastlySecretStoreOpen(
	name prim.Wstring,
	h *secretStoreHandle,
) FastlyStatus

// OpenSecretStore returns a reference to the named secret store, if it exists.
func OpenSecretStore(name string) (*SecretStore, error) {
	var st SecretStore

	if err := fastlySecretStoreOpen(
		prim.NewReadBufferFromString(name).Wstring(),
		&st.h,
	).toError(); err != nil {
		return nil, err
	}

	return &st, nil
}

// witx:
//
//   (module $fastly_secret_store
//     (@interface func (export "get")
//       (param $store $secret_store_handle)
//       (param $key string)
//       (result $err (expected $secret_handle (error $fastly_status)))
//     )
//   )

//go:wasm-module fastly_secret_store
//export get
//go:noescape
func fastlySecretStoreGet(
	h secretStoreHandle,
	key prim.Wstring,
	s *secretHandle,
) FastlyStatus

// Get returns a handle to the secret value for the given name, if it
// exists.
func (st *SecretStore) Get(name string) (*Secret, error) {
	var s Secret

	if err := fastlySecretStoreGet(
		st.h,
		prim.NewReadBufferFromString(name).Wstring(),
		&s.h,
	).toError(); err != nil {
		return nil, err
	}

	return &s, nil
}

// witx:
//
//   (module $fastly_secret_store
//     (@interface func (export "plaintext")
//       (param $secret $secret_handle)
//       (param $buf (@witx pointer (@witx char8)))
//       (param $buf_len (@witx usize))
//       (param $nwritten_out (@witx pointer (@witx usize)))
//       (result $err (expected (error $fastly_status)))
//     )
//   )

//go:wasm-module fastly_secret_store
//export plaintext
//go:noescape
func fastlySecretPlaintext(
	h secretHandle,
	buf *prim.Char8,
	bufLen prim.Usize,
	nwritten *prim.Usize,
) FastlyStatus

// Plaintext decrypts and returns the secret value as a byte slice.
func (s *Secret) Plaintext() ([]byte, error) {
	// Most secrets will fit into the initial secret buffer size, so
	// we'll start with that. If it doesn't fit, we'll know the exact
	// size of the buffer to try again.
	buf := prim.NewWriteBuffer(initialSecretLen)

	status := fastlySecretPlaintext(
		s.h,
		buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	)
	if status == FastlyStatusBufLen {
		// The buffer was too small, but it'll tell us how big it will
		// need to be in order to fit the plaintext.
		buf = prim.NewWriteBuffer(int(buf.NValue()))

		status = fastlySecretPlaintext(
			s.h,
			buf.Char8Pointer(),
			buf.Cap(),
			buf.NPointer(),
		)
	}

	if err := status.toError(); err != nil {
		return nil, err
	}

	return buf.AsBytes(), nil
}

// witx:
//
// (@interface func (export "from_bytes")
//     (param $buf (@witx pointer (@witx char8)))
//     (param $buf_len (@witx usize))
//     (result $err (expected $secret_handle (error $fastly_status)))
// )

//go:wasm-module fastly_secret_store
//export from_bytes
//go:noescape
func fastlySecretFromBytes(
	buf *prim.Char8,
	bufLen prim.Usize,
	h *secretHandle,
) FastlyStatus

// FromBytes creates a secret handle for the given byte slice.  This is
// for use with APIs that require a secret handle but cannot (for
// whatever reason) use a secret store.
func SecretFromBytes(b []byte) (*Secret, error) {
	var s Secret

	if err := fastlySecretFromBytes(
		prim.NewReadBufferFromBytes(b).Char8Pointer(),
		prim.Usize(len(b)),
		&s.h,
	).toError(); err != nil {
		return nil, err
	}

	return &s, nil
}

type CacheLookupOptions struct {
	opts cacheLookupOptions
	mask cacheLookupOptionsMask
}

func (o *CacheLookupOptions) SetRequest(req *HTTPRequest) {
	o.opts.requestHeaders = req.h
	o.mask |= cacheLookupOptionsMaskRequestHeaders
}

type CacheGetBodyOptions struct {
	opts cacheGetBodyOptions
	mask cacheGetBodyOptionsMask
}

func (o *CacheGetBodyOptions) From(from uint64) {
	o.opts.from = prim.U64(from)
	o.mask |= cacheGetBodyOptionsMaskFrom
}

func (o *CacheGetBodyOptions) To(to uint64) {
	o.opts.to = prim.U64(to)
	o.mask |= cacheGetBodyOptionsMaskTo
}

type CacheWriteOptions struct {
	opts cacheWriteOptions
	mask cacheWriteOptionsMask
}

func (o *CacheWriteOptions) MaxAge(v time.Duration) {
	o.opts.maxAgeNs = prim.U64(v.Nanoseconds())
}

func (o *CacheWriteOptions) SetRequest(req *HTTPRequest) {
	o.opts.requestHeaders = req.h
	o.mask |= cacheWriteOptionsMaskRequestHeaders
}

func (o *CacheWriteOptions) Vary(v []string) {
	vstr := strings.Join(v, " ")
	buf := prim.NewReadBufferFromString(vstr)
	o.opts.varyRulePtr = buf.Char8Pointer()
	o.opts.varyRuleLen = buf.Len()
	o.mask |= cacheWriteOptionsMaskVaryRule
}

func (o *CacheWriteOptions) InitialAge(v time.Duration) {
	o.opts.initialAgeNs = prim.U64(v.Nanoseconds())
	o.mask |= cacheWriteOptionsMaskInitialAgeNs
}

func (o *CacheWriteOptions) StaleWhileRevalidate(v time.Duration) {
	o.opts.staleWhileRevalidateNs = prim.U64(v.Nanoseconds())
	o.mask |= cacheWriteOptionsMaskStaleWhileRevalidateNs
}

func (o *CacheWriteOptions) SurrogateKeys(v []string) {
	vstr := strings.Join(v, " ")
	buf := prim.NewReadBufferFromString(vstr)
	o.opts.surrogateKeysPtr = buf.Char8Pointer()
	o.opts.surrogateKeysLen = buf.Len()
	o.mask |= cacheWriteOptionsMaskSurrogateKeys
}

func (o *CacheWriteOptions) ContentLength(v uint64) {
	o.opts.length = prim.U64(v)
	o.mask |= cacheWriteOptionsMaskLength
}

func (o *CacheWriteOptions) UserMetadata(v []byte) {
	buf := prim.NewReadBufferFromBytes(v)
	o.opts.userMetadataPtr = buf.U8Pointer()
	o.opts.userMetadataLen = buf.Len()
	o.mask |= cacheWriteOptionsMaskUserMetadata
}

func (o *CacheWriteOptions) SensitiveData(v bool) {
	if v {
		o.mask |= cacheWriteOptionsMaskSensitiveData
	} else {
		o.mask &^= cacheWriteOptionsMaskSensitiveData
	}
}

type CacheEntry struct {
	h cacheHandle
}

// witx:
//
//	(module $fastly_cache
//	  (@interface func (export "lookup")
//	    (param $cache_key (list u8))
//	    (param $options_mask $cache_lookup_options_mask)
//	    (param $options (@witx pointer $cache_lookup_options))
//	    (result $err (expected $cache_handle (error $fastly_status)))
//	  )
//	)
//
//go:wasm-module fastly_cache
//export lookup
//go:noescape
func fastlyCacheLookup(
	key prim.ArrayU8,
	mask cacheLookupOptionsMask,
	opts *cacheLookupOptions,
	h *cacheHandle,
) FastlyStatus

func CacheLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	var entry CacheEntry

	if err := fastlyCacheLookup(
		prim.NewReadBufferFromBytes(key).ArrayU8(),
		opts.mask,
		&opts.opts,
		&entry.h,
	).toError(); err != nil {
		return nil, err
	}

	return &entry, nil
}

// witx:
//
//	 ;;; Performs a non-request-collapsing cache insertion (or update).
//	 ;;;
//	 ;;; The returned handle is to a streaming body that is used for writing the object into
//	 ;;; the cache.
//	 (@interface func (export "insert")
//		  (param $cache_key (list u8))
//		  (param $options_mask $cache_write_options_mask)
//		  (param $options (@witx pointer $cache_write_options))
//		  (result $err (expected $body_handle (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export insert
//go:noescape
func fastlyCacheInsert(
	key prim.ArrayU8,
	mask cacheWriteOptionsMask,
	opts *cacheWriteOptions,
	h *bodyHandle,
) FastlyStatus

func CacheInsert(key []byte, opts CacheWriteOptions) (*HTTPBody, error) {
	body := HTTPBody{closable: true}

	if err := fastlyCacheInsert(
		prim.NewReadBufferFromBytes(key).ArrayU8(),
		opts.mask,
		&opts.opts,
		&body.h,
	).toError(); err != nil {
		return nil, err
	}

	return &body, nil
}

// witx:
//
//	 ;;; The entrypoint to the request-collapsing cache transaction API.
//	 ;;;
//	 ;;; This operation always participates in request collapsing and may return stale objects. To bypass
//	 ;;; request collapsing, use `lookup` and `insert` instead.
//	 (@interface func (export "transaction_lookup")
//		  (param $cache_key (list u8))
//		  (param $options_mask $cache_lookup_options_mask)
//		  (param $options (@witx pointer $cache_lookup_options))
//		  (result $err (expected $cache_handle (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export transaction_lookup
//go:noescape
func fastlyCacheTransactionLookup(
	key prim.ArrayU8,
	mask cacheLookupOptionsMask,
	opts *cacheLookupOptions,
	h *cacheHandle,
) FastlyStatus

func CacheTransactionLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	var entry CacheEntry

	if err := fastlyCacheTransactionLookup(
		prim.NewReadBufferFromBytes(key).ArrayU8(),
		opts.mask,
		&opts.opts,
		&entry.h,
	).toError(); err != nil {
		return nil, err
	}

	return &entry, nil
}

// witx:
//
//	;;; Insert an object into the cache with the given metadata.
//	;;;
//	;;; Can only be used in if the cache handle state includes the `$must_insert_or_update` flag.
//	;;;
//	;;; The returned handle is to a streaming body that is used for writing the object into
//	;;; the cache.
//	(@interface func (export "transaction_insert")
//	  (param $handle $cache_handle)
//	  (param $options_mask $cache_write_options_mask)
//	  (param $options (@witx pointer $cache_write_options))
//	  (result $err (expected $body_handle (error $fastly_status)))
//	)
//
//go:wasm-module fastly_cache
//export transaction_insert
//go:noescape
func fastlyCacheTransactionInsert(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts *cacheWriteOptions,
	body *bodyHandle,
) FastlyStatus

func (c *CacheEntry) Insert(opts CacheWriteOptions) (*HTTPBody, error) {
	body := HTTPBody{closable: true}

	if err := fastlyCacheTransactionInsert(
		c.h,
		opts.mask,
		&opts.opts,
		&body.h,
	).toError(); err != nil {
		return nil, err
	}

	return &body, nil
}

// witx:
//
//	;;; Insert an object into the cache with the given metadata, and return a readable stream of the
//	;;; bytes as they are stored.
//	;;;
//	;;; This helps avoid the "slow reader" problem on a teed stream, for example when a program wishes
//	;;; to store a backend request in the cache while simultaneously streaming to a client in an HTTP
//	;;; response.
//	;;;
//	;;; The returned body handle is to a streaming body that is used for writing the object _into_
//	;;; the cache. The returned cache handle provides a separate transaction for reading out the
//	;;; newly cached object to send elsewhere.
//	(@interface func (export "transaction_insert_and_stream_back")
//	  (param $handle $cache_handle)
//	  (param $options_mask $cache_write_options_mask)
//	  (param $options (@witx pointer $cache_write_options))
//	  (result $err (expected (tuple $body_handle $cache_handle) (error $fastly_status)))
//	)
//
//go:wasm-module fastly_cache
//export transaction_insert_and_stream_back
//go:noescape
func fastlyCacheTransactionInsertAndStreamBack(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts *cacheWriteOptions,
	body *bodyHandle,
	stream *cacheHandle,
) FastlyStatus

func (c *CacheEntry) InsertAndStreamBack(opts CacheWriteOptions) (*HTTPBody, *CacheEntry, error) {
	var entry CacheEntry
	body := HTTPBody{closable: true}

	if err := fastlyCacheTransactionInsertAndStreamBack(
		c.h,
		opts.mask,
		&opts.opts,
		&body.h,
		&entry.h,
	).toError(); err != nil {
		return nil, nil, err
	}

	return &body, &entry, nil
}

// witx:
//
//	;;; Update the metadata of an object in the cache without changing its data.
//	;;;
//	;;; Can only be used in if the cache handle state includes both of the flags:
//	;;; - `$found`
//	;;; - `$must_insert_or_update`
//	(@interface func (export "transaction_update")
//	  (param $handle $cache_handle)
//	  (param $options_mask $cache_write_options_mask)
//	  (param $options (@witx pointer $cache_write_options))
//	  (result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_cache
//export transaction_update
//go:noescape
func fastlyCacheTransactionUpdate(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts *cacheWriteOptions,
) FastlyStatus

func (c *CacheEntry) Update(opts CacheWriteOptions) error {
	return fastlyCacheTransactionUpdate(
		c.h,
		opts.mask,
		&opts.opts,
	).toError()
}

// witx:
//
//	 ;;; Cancel an obligation to provide an object to the cache.
//	 ;;;
//	 ;;; Useful if there is an error before streaming is possible, e.g. if a backend is unreachable.
//	 (@interface func (export "transaction_cancel")
//		  (param $handle $cache_handle)
//		  (result $err (expected (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export transaction_cancel
//go:noescape
func fastlyCacheTransactionCancel(h cacheHandle) FastlyStatus

func (c *CacheEntry) Cancel() error {
	return fastlyCacheTransactionCancel(c.h).toError()
}

// witx:
//
//	(@interface func (export "close")
//	  (param $handle $cache_handle)
//	  (result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_cache
//export close
//go:noescape
func fastlyCacheClose(h cacheHandle) FastlyStatus

func (c *CacheEntry) Close() error {
	return fastlyCacheClose(c.h).toError()
}

// witx:
//
//	(@interface func (export "get_state")
//	  (param $handle $cache_handle)
//	  (result $err (expected $cache_lookup_state (error $fastly_status)))
//	)
//
//go:wasm-module fastly_cache
//export get_state
//go:noescape
func fastlyCacheGetState(h cacheHandle, st *CacheLookupState) FastlyStatus

func (c *CacheEntry) State() (CacheLookupState, error) {
	var state CacheLookupState
	if err := fastlyCacheGetState(c.h, &state).toError(); err != nil {
		return 0, err
	}

	return state, nil
}

// witx:
//
//	 ;;; Gets the user metadata of the found object, returning the `$none` error if there
//	 ;;; was no found object.
//	 (@interface func (export "get_user_metadata")
//		  (param $handle $cache_handle)
//		  (param $user_metadata_out_ptr (@witx pointer u8))
//		  (param $user_metadata_out_len (@witx usize))
//		  (param $nwritten_out (@witx pointer (@witx usize)))
//		  (result $err (expected (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_user_metadata
//go:noescape
func fastlyCacheGetUserMetadata(
	h cacheHandle,
	buf *prim.U8,
	bufLen prim.Usize,
	nwritten *prim.Usize,
) FastlyStatus

func (c *CacheEntry) UserMetadata() ([]byte, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	status := fastlyCacheGetUserMetadata(
		c.h,
		buf.U8Pointer(),
		buf.Cap(),
		buf.NPointer(),
	)
	if status == FastlyStatusBufLen {
		// The buffer was too small, but it'll tell us how big it will
		// need to be in order to fit the content.
		buf = prim.NewWriteBuffer(int(buf.NValue()))

		status = fastlyCacheGetUserMetadata(
			c.h,
			buf.U8Pointer(),
			buf.Cap(),
			buf.NPointer(),
		)
	}

	if err := status.toError(); err != nil {
		return nil, err
	}

	return buf.AsBytes(), nil
}

// witx:
//
//	 ;;; Gets a range of the found object body, returning the `$none` error if there
//	 ;;; was no found object.
//	 (@interface func (export "get_body")
//	   (param $handle $cache_handle)
//		  (param $options_mask $cache_get_body_options_mask)
//		  (param $options $cache_get_body_options)
//		  (result $err (expected $body_handle (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_body
//go:noescape
func fastlyCacheGetBody(
	h cacheHandle,
	mask cacheGetBodyOptionsMask,
	opts *cacheGetBodyOptions,
	body *bodyHandle,
) FastlyStatus

func (c *CacheEntry) Body(opts CacheGetBodyOptions) (*HTTPBody, error) {
	var b HTTPBody

	if err := fastlyCacheGetBody(
		c.h,
		opts.mask,
		&opts.opts,
		&b.h,
	).toError(); err != nil {
		return nil, err
	}

	b.closable = true

	return &b, nil
}

// witx:
//
//	 ;;; Gets the content length of the found object, returning the `$none` error if there
//	 ;;; was no found object, or no content length was provided.
//	 (@interface func (export "get_length")
//		  (param $handle $cache_handle)
//		  (result $err (expected $cache_object_length (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_length
//go:noescape
func fastlyCacheGetLength(h cacheHandle, l *prim.U64) FastlyStatus

func (c *CacheEntry) Length() (uint64, error) {
	var l prim.U64
	if err := fastlyCacheGetLength(c.h, &l).toError(); err != nil {
		return 0, err
	}

	return uint64(l), nil
}

// witx:
//
//	 ;;; Gets the configured max age of the found object, returning the `$none` error if there
//	 ;;; was no found object.
//	 (@interface func (export "get_max_age_ns")
//		  (param $handle $cache_handle)
//		  (result $err (expected $cache_duration_ns (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_max_age_ns
//go:noescape
func fastlyCacheGetMaxAgeNs(h cacheHandle, d *prim.U64) FastlyStatus

func (c *CacheEntry) MaxAge() (time.Duration, error) {
	var d prim.U64
	if err := fastlyCacheGetMaxAgeNs(c.h, &d).toError(); err != nil {
		return 0, err
	}

	return time.Duration(d), nil
}

// witx:
//
//	 ;;; Gets the configured stale-while-revalidate period of the found object, returning the
//	 ;;; `$none` error if there was no found object.
//	 (@interface func (export "get_stale_while_revalidate_ns")
//		  (param $handle $cache_handle)
//		  (result $err (expected $cache_duration_ns (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_stale_while_revalidate_ns
//go:noescape
func fastlyCacheGetStaleWhileRevalidateNs(h cacheHandle, d *prim.U64) FastlyStatus

func (c *CacheEntry) StaleWhileRevalidate() (time.Duration, error) {
	var d prim.U64
	if err := fastlyCacheGetStaleWhileRevalidateNs(c.h, &d).toError(); err != nil {
		return 0, err
	}

	return time.Duration(d), nil
}

// witx:
//
//	 ;;; Gets the age of the found object, returning the `$none` error if there
//	 ;;; was no found object.
//	 (@interface func (export "get_age_ns")
//		  (param $handle $cache_handle)
//		  (result $err (expected $cache_duration_ns (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_age_ns
//go:noescape
func fastlyCacheGetAgeNs(h cacheHandle, d *prim.U64) FastlyStatus

func (c *CacheEntry) Age() (time.Duration, error) {
	var d prim.U64
	if err := fastlyCacheGetAgeNs(c.h, &d).toError(); err != nil {
		return 0, err
	}

	return time.Duration(d), nil
}

// witx:
//
//	 ;;; Gets the number of cache hits for the found object, returning the `$none` error if there
//	 ;;; was no found object.
//	 (@interface func (export "get_hits")
//		  (param $handle $cache_handle)
//		  (result $err (expected $cache_hit_count (error $fastly_status)))
//	 )
//
//go:wasm-module fastly_cache
//export get_hits
//go:noescape
func fastlyCacheGetHits(h cacheHandle, d *prim.U64) FastlyStatus

func (c *CacheEntry) Hits() (uint64, error) {
	var d prim.U64
	if err := fastlyCacheGetHits(c.h, &d).toError(); err != nil {
		return 0, err
	}

	return uint64(d), nil
}

type PurgeOptions struct {
	mask purgeOptionsMask
	opts purgeOptions
}

func (o *PurgeOptions) SoftPurge(v bool) {
	if v {
		o.mask |= purgeOptionsMaskSoftPurge
	} else {
		o.mask &^= purgeOptionsMaskSoftPurge
	}
}

// witx:
//
//	(@interface func (export "purge_surrogate_key")
//	    (param $surrogate_key string)
//	    (param $options_mask $purge_options_mask)
//	    (param $options (@witx pointer $purge_options))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasm-module fastly_purge
//export purge_surrogate_key
//go:noescape
func fastlyPurgeSurrogateKey(surrogateKey prim.Wstring, mask purgeOptionsMask, opts *purgeOptions) FastlyStatus

func PurgeSurrogateKey(surrogateKey string, opts PurgeOptions) error {
	return fastlyPurgeSurrogateKey(
		prim.NewReadBufferFromString(surrogateKey).Wstring(),
		opts.mask,
		&opts.opts,
	).toError()
}
