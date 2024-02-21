//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

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

func init() {
	fastlyABIInit(1)
}

// witx:
//
//	(module $fastly_abi
//	 (@interface func (export "init")
//	   (param $abi_version u64)
//	   (result $err $fastly_status))
//	)
//
//go:wasmimport fastly_abi init
//go:noescape
func fastlyABIInit(abiVersion prim.U64) FastlyStatus

// TODO(pb): this doesn't need to be exported, I don't think?
// Initialize the Fastly ABI at the given version.
//func Initialize(version uint64) error {

//	return fastlyABIInit(version).toError()
//}

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
	var b HTTPBody

	if err := fastlyHTTPBodyNew(
		prim.ToPointer(&b.h),
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
//go:wasmimport fastly_log endpoint_get
//go:noescape
func fastlyLogEndpointGet(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	endpointHandleOut prim.Pointer[endpointHandle],
) FastlyStatus

// GetLogEndpoint opens the log endpoint identified by name.
func GetLogEndpoint(name string) (*LogEndpoint, error) {
	var e LogEndpoint

	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	if err := fastlyLogEndpointGet(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&e.h),
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
//go:wasmimport fastly_log write
//go:noescape
func fastlyLogWrite(
	h endpointHandle,
	msgData prim.Pointer[prim.U8], msgLen prim.Usize,
	nWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the endpoint.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (e *LogEndpoint) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		p_n_Buffer := prim.NewReadBufferFromBytes(p[n:]).ArrayU8()

		if err = fastlyLogWrite(
			e.h,
			p_n_Buffer.Data, p_n_Buffer.Len,
			prim.ToPointer(&nWritten),
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
		rh requestHandle = requestHandle(math.MaxUint32 - 1)
		bh bodyHandle    = bodyHandle(math.MaxUint32 - 1)
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
//	(@interface func (export "downstream_client_ip_addr")
//	   ;; must be a 16-byte array
//	   (param $addr_octets_out (@witx pointer char8))
//	   (result $err $fastly_status)
//	   (result $nwritten_out (@witx usize))
//	)
//
//go:wasmimport fastly_http_req downstream_client_ip_addr
//go:noescape
func fastlyHTTPReqDownstreamClientIPAddr(
	addrOctetsOut prim.Pointer[prim.Char8], // must be 16-byte array
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamClientIPAddr returns the IP address of the downstream client that
// performed the singleton downstream request.
func DownstreamClientIPAddr() (net.IP, error) {
	buf := prim.NewWriteBuffer(16) // must be a 16-byte array

	if err := fastlyHTTPReqDownstreamClientIPAddr(
		prim.ToPointer(buf.Char8Pointer()),
		prim.ToPointer(buf.NPointer()),
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
//go:wasmimport fastly_http_req downstream_tls_cipher_openssl_name
//go:noescape
func fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
	cipherOut prim.Pointer[prim.Char8],
	cipherMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSCipherOpenSSLName returns the name of the OpenSSL TLS cipher
// used with the singleton downstream request, if any.
func DownstreamTLSCipherOpenSSLName() (string, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	if err := fastlyHTTPReqDownstreamTLSCipherOpenSSLName(
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
//	   (param $protocol_out (@witx pointer char8))
//	   (param $protocol_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req downstream_tls_protocol
//go:noescape
func fastlyHTTPReqDownstreamTLSProtocol(
	protocolOut prim.Pointer[prim.Char8],
	protocolMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSProtocol returns the name of the TLS protocol used with the
// singleton downstream request, if any.
func DownstreamTLSProtocol() (string, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	if err := fastlyHTTPReqDownstreamTLSProtocol(
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
//	   (param $chello_out (@witx pointer char8))
//	   (param $chello_max_len (@witx usize))
//	   (param $nwritten_out (@witx pointer (@witx usize)))
//	   (result $err $fastly_status)
//	)
//
//go:wasmimport fastly_http_req downstream_tls_client_hello
//go:noescape
func fastlyHTTPReqDownstreamTLSClientHello(
	chelloOut prim.Pointer[prim.Char8],
	chelloMaxLen prim.Usize,
	nwrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// DownstreamTLSClientHello returns the ClientHello message sent by the client
// in the singleton downstream request, if any.
func DownstreamTLSClientHello() ([]byte, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	if err := fastlyHTTPReqDownstreamTLSClientHello(
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
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
//go:wasmimport fastly_http_req new
//go:noescape
func fastlyHTTPReqNew(
	h prim.Pointer[requestHandle],
) FastlyStatus

// NewHTTPRequest returns a new, empty HTTP request.
func NewHTTPRequest() (*HTTPRequest, error) {
	var r HTTPRequest

	if err := fastlyHTTPReqNew(
		prim.ToPointer(&r.h),
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
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
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
//go:wasmimport fastly_http_req original_header_names_get
//go:noescape
func fastlyHTTPReqOriginalHeaderNamesGet(
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut prim.Pointer[multiValueCursorResult],
	nwrittenOut prim.Pointer[prim.Usize],
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
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
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
//go:wasmimport fastly_http_req original_header_count
//go:noescape
func fastlyHTTPReqOriginalHeaderCount(
	count prim.Pointer[prim.U32],
) FastlyStatus

// GetOriginalHeaderCount returns the number of headers of the singleton
// downstream request.
func GetOriginalHeaderCount() (int, error) {
	var count prim.U32

	if err := fastlyHTTPReqOriginalHeaderCount(
		prim.ToPointer(&count),
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
func (r *HTTPRequest) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	if err := fastlyHTTPReqHeaderValueGet(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
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
func (r *HTTPRequest) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
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
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
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
		fmt.Fprint(&buf, value)
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
func (r *HTTPRequest) GetMethod(maxMethodLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxMethodLen)

	if err := fastlyHTTPReqMethodGet(
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
func (r *HTTPRequest) GetURI(maxURLLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxURLLen)

	if err := fastlyHTTPReqURIGet(
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
		resp     HTTPResponse
		respBody HTTPBody
	)

	errDetail := newSendErrorDetail()
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendV2(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&resp.h),
		prim.ToPointer(&respBody.h),
	).toSendError(errDetail); err != nil {
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
	var pendingReq PendingRequest

	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendAsync(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&pendingReq.h),
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
	var pendingReq PendingRequest

	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()

	if err := fastlyHTTPReqSendAsyncStreaming(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		prim.ToPointer(&pendingReq.h),
	).toError(); err != nil {
		return nil, err
	}

	requestBody.closable = true

	return &pendingReq, nil
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
		resp      HTTPResponse
		respBody  HTTPBody
		isDone    prim.U32
		errDetail = newSendErrorDetail()
	)

	if err := fastlyHTTPReqPendingReqPollV2(
		r.h,
		prim.ToPointer(&errDetail),
		prim.ToPointer(&isDone),
		prim.ToPointer(&resp.h),
		prim.ToPointer(&respBody.h),
	).toSendError(errDetail); err != nil {
		return false, nil, nil, err
	}

	return isDone > 0, &resp, &respBody, nil
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
//go:wasmimport fastly_http_req register_dynamic_backend
//go:noescape
func fastlyRegisterDynamicBackend(nameData prim.Pointer[prim.U8], nameLen prim.Usize, targetData prim.Pointer[prim.U8], targetLen prim.Usize, mask backendConfigOptionsMask, opts prim.Pointer[backendConfigOptions]) FastlyStatus

func RegisterDynamicBackend(name string, target string, opts *BackendConfigOptions) error {
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()
	targetBuffer := prim.NewReadBufferFromString(target).Wstring()

	if err := fastlyRegisterDynamicBackend(
		nameBuffer.Data, nameBuffer.Len,
		targetBuffer.Data, targetBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
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
//go:wasmimport fastly_backend exists
//go:noescape
func fastlyBackendExists(nameData prim.Pointer[prim.U8], nameLen prim.Usize, exists prim.Pointer[prim.U32]) FastlyStatus

func BackendExists(name string) (bool, error) {
	var exists prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendExists(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&exists),
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
//go:wasmimport fastly_backend is_healthy
//go:noescape
func fastlyBackendIsHealthy(nameData prim.Pointer[prim.U8], nameLen prim.Usize, healthy prim.Pointer[prim.U32]) FastlyStatus

func BackendIsHealthy(name string) (BackendHealth, error) {
	var health prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsHealthy(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&health),
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
//go:wasmimport fastly_backend is_dynamic
//go:noescape
func fastlyBackendIsDynamic(nameData prim.Pointer[prim.U8], nameLen prim.Usize, dynamic prim.Pointer[prim.U32]) FastlyStatus

func BackendIsDynamic(name string) (bool, error) {
	var dynamic prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsDynamic(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&dynamic),
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
//go:wasmimport fastly_backend get_host
//go:noescape
func fastlyBackendGetHost(nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	host prim.Pointer[prim.Char8],
	hostLen prim.Usize,
	hostWritten prim.Pointer[prim.Usize],
) FastlyStatus

func BackendGetHost(name string) (string, error) {
	hostBuf := prim.NewWriteBuffer(defaultBufferLen)

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetHost(
		nameBuffer.Data, nameBuffer.Len,

		prim.ToPointer(hostBuf.Char8Pointer()),
		hostBuf.Cap(),
		prim.ToPointer(hostBuf.NPointer()),
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
//go:wasmimport fastly_backend get_override_host
//go:noescape
func fastlyBackendGetOverrideHost(nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	host prim.Pointer[prim.Char8],
	hostLen prim.Usize,
	hostWritten prim.Pointer[prim.Usize],
) FastlyStatus

func BackendGetOverrideHost(name string) (string, error) {
	hostBuf := prim.NewWriteBuffer(defaultBufferLen)

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetOverrideHost(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(hostBuf.Char8Pointer()),
		hostBuf.Cap(),
		prim.ToPointer(hostBuf.NPointer()),
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
//go:wasmimport fastly_backend get_port
//go:noescape
func fastlyBackendGetPort(nameData prim.Pointer[prim.U8], nameLen prim.Usize, port prim.Pointer[prim.U32]) FastlyStatus

func BackendGetPort(name string) (int, error) {
	var port prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetPort(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&port),
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
//go:wasmimport fastly_backend get_connect_timeout_ms
//go:noescape
func fastlyBackendGetConnectTimeoutMs(nameData prim.Pointer[prim.U8], nameLen prim.Usize, timeout prim.Pointer[prim.U32]) FastlyStatus

func BackendGetConnectTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetConnectTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
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
//go:wasmimport fastly_backend get_first_byte_timeout_ms
//go:noescape
func fastlyBackendGetFirstByteTimeoutMs(nameData prim.Pointer[prim.U8], nameLen prim.Usize, timeout prim.Pointer[prim.U32]) FastlyStatus

func BackendGetFirstByteTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetFirstByteTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
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
//go:wasmimport fastly_backend get_between_bytes_timeout_ms
//go:noescape
func fastlyBackendGetBetweenBytesTimeoutMs(nameData prim.Pointer[prim.U8], nameLen prim.Usize, timeout prim.Pointer[prim.U32]) FastlyStatus

func BackendGetBetweenBytesTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetBetweenBytesTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
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
//go:wasmimport fastly_backend is_ssl
//go:noescape
func fastlyBackendIsSSL(nameData prim.Pointer[prim.U8], nameLen prim.Usize, ssl prim.Pointer[prim.U32]) FastlyStatus

func BackendIsSSL(name string) (bool, error) {
	var ssl prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsSSL(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&ssl),
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
//go:wasmimport fastly_backend get_ssl_min_version
//go:noescape
func fastlyBackendGetSSLMinVersion(nameData prim.Pointer[prim.U8], nameLen prim.Usize, version prim.Pointer[prim.U32]) FastlyStatus

func BackendGetSSLMinVersion(name string) (TLSVersion, error) {
	var version prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetSSLMinVersion(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&version),
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
//go:wasmimport fastly_backend get_ssl_max_version
//go:noescape
func fastlyBackendGetSSLMaxVersion(nameData prim.Pointer[prim.U8], nameLen prim.Usize, version prim.Pointer[prim.U32]) FastlyStatus

func BackendGetSSLMaxVersion(name string) (TLSVersion, error) {
	var version prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetSSLMaxVersion(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&version),
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
	var resp HTTPResponse

	if err := fastlyHTTPRespNew(
		prim.ToPointer(&resp.h),
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
			prim.ToPointer(buf),
			bufLen,
			cursor,
			prim.ToPointer(endingCursorOut),
			prim.ToPointer(nwrittenOut),
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
func (r *HTTPResponse) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	if err := fastlyHTTPRespHeaderValueGet(
		r.h,
		nameBuffer.Data, nameBuffer.Len,
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
func (r *HTTPResponse) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
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
		fmt.Fprint(&buf, value)
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
//	(module $fastly_dictionary
//	   (@interface func (export "open")
//	      (param $name string)
//	      (result $err $fastly_status)
//	      (result $h $dictionary_handle)
//	   )
//
//go:wasmimport fastly_dictionary open
//go:noescape
func fastlyDictionaryOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[dictionaryHandle],
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

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyDictionaryOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&d.h),
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
//go:wasmimport fastly_dictionary get
//go:noescape
func fastlyDictionaryGet(
	h dictionaryHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	value prim.Pointer[prim.Char8],
	valueMaxLen prim.Usize,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

// Get the value for key, if it exists.
func (d *Dictionary) Get(key string) (string, error) {
	buf := prim.NewWriteBuffer(dictionaryValueMaxLen)
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	if err := fastlyDictionaryGet(
		d.h,
		keyBuffer.Data, keyBuffer.Len,
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
//go:wasmimport fastly_geo lookup
//go:noescape
func fastlyGeoLookup(
	addrOctets prim.Pointer[prim.Char8],
	addLen prim.Usize,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	nWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GeoLookup returns the geographic data associated with the IP address.
func GeoLookup(ip net.IP) ([]byte, error) {
	buf := prim.NewWriteBuffer(1024) // initial geo buf size
	if x := ip.To4(); x != nil {
		ip = x
	}
	addrOctets := prim.NewReadBufferFromBytes(ip)

	if err := fastlyGeoLookup(
		prim.ToPointer(addrOctets.Char8Pointer()),
		addrOctets.Len(),
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
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

//go:wasmimport fastly_object_store open
//go:noescape
func fastlyObjectStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[objectStoreHandle],
) FastlyStatus

// objectStore represents a Fastly kv store, a collection of key/value pairs.
// For convenience, keys and values are both modelled as Go strings.
type KVStore struct {
	h objectStoreHandle
}

// KVStoreOpen returns a reference to the named kv store, if it exists.
func OpenKVStore(name string) (*KVStore, error) {
	var o KVStore

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyObjectStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&o.h),
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

//go:wasmimport fastly_object_store lookup
//go:noescape
func fastlyObjectStoreLookup(
	h objectStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	b prim.Pointer[bodyHandle],
) FastlyStatus

// Lookup returns the value for key, if it exists.
func (o *KVStore) Lookup(key string) (io.Reader, error) {
	body := HTTPBody{h: invalidBodyHandle}

	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	if err := fastlyObjectStoreLookup(
		o.h,
		keyBuffer.Data, keyBuffer.Len,
		prim.ToPointer(&body.h),
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

//go:wasmimport fastly_object_store insert
//go:noescape
func fastlyObjectStoreInsert(
	h objectStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
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

	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	if err := fastlyObjectStoreInsert(
		o.h,
		keyBuffer.Data, keyBuffer.Len,
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

//go:wasmimport fastly_secret_store open
//go:noescape
func fastlySecretStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[secretStoreHandle],
) FastlyStatus

// OpenSecretStore returns a reference to the named secret store, if it exists.
func OpenSecretStore(name string) (*SecretStore, error) {
	var st SecretStore

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlySecretStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&st.h),
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

//go:wasmimport fastly_secret_store get
//go:noescape
func fastlySecretStoreGet(
	h secretStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	s prim.Pointer[secretHandle],
) FastlyStatus

// Get returns a handle to the secret value for the given name, if it
// exists.
func (st *SecretStore) Get(name string) (*Secret, error) {
	var s Secret

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlySecretStoreGet(
		st.h,
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&s.h),
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

//go:wasmimport fastly_secret_store plaintext
//go:noescape
func fastlySecretPlaintext(
	h secretHandle,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
) FastlyStatus

// Plaintext decrypts and returns the secret value as a byte slice.
func (s *Secret) Plaintext() ([]byte, error) {
	// Most secrets will fit into the initial secret buffer size, so
	// we'll start with that. If it doesn't fit, we'll know the exact
	// size of the buffer to try again.
	buf := prim.NewWriteBuffer(initialSecretLen)

	status := fastlySecretPlaintext(
		s.h,
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	)
	if status == FastlyStatusBufLen {
		// The buffer was too small, but it'll tell us how big it will
		// need to be in order to fit the plaintext.
		buf = prim.NewWriteBuffer(int(buf.NValue()))

		status = fastlySecretPlaintext(
			s.h,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
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

//go:wasmimport fastly_secret_store from_bytes
//go:noescape
func fastlySecretFromBytes(
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	h prim.Pointer[secretHandle],
) FastlyStatus

// FromBytes creates a secret handle for the given byte slice.  This is
// for use with APIs that require a secret handle but cannot (for
// whatever reason) use a secret store.
func SecretFromBytes(b []byte) (*Secret, error) {
	var s Secret

	buf := prim.NewReadBufferFromBytes(b)

	if err := fastlySecretFromBytes(
		prim.ToPointer(buf.Char8Pointer()),
		buf.Len(),
		prim.ToPointer(&s.h),
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
	o.opts.varyRulePtr = prim.ToPointer(buf.Char8Pointer())
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
	o.opts.surrogateKeysPtr = prim.ToPointer(buf.Char8Pointer())
	o.opts.surrogateKeysLen = buf.Len()
	o.mask |= cacheWriteOptionsMaskSurrogateKeys
}

func (o *CacheWriteOptions) ContentLength(v uint64) {
	o.opts.length = prim.U64(v)
	o.mask |= cacheWriteOptionsMaskLength
}

func (o *CacheWriteOptions) UserMetadata(v []byte) {
	buf := prim.NewReadBufferFromBytes(v)
	o.opts.userMetadataPtr = prim.ToPointer(buf.U8Pointer())
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
//go:wasmimport fastly_cache lookup
//go:noescape
func fastlyCacheLookup(
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	mask cacheLookupOptionsMask,
	opts prim.Pointer[cacheLookupOptions],
	h prim.Pointer[cacheHandle],
) FastlyStatus

func CacheLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	var entry CacheEntry

	keyBuffer := prim.NewReadBufferFromBytes(key).ArrayU8()

	if err := fastlyCacheLookup(
		keyBuffer.Data, keyBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&entry.h),
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
//go:wasmimport fastly_cache insert
//go:noescape
func fastlyCacheInsert(
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	mask cacheWriteOptionsMask,
	opts prim.Pointer[cacheWriteOptions],
	h prim.Pointer[bodyHandle],
) FastlyStatus

func CacheInsert(key []byte, opts CacheWriteOptions) (*HTTPBody, error) {
	body := HTTPBody{closable: true}

	keyBuffer := prim.NewReadBufferFromBytes(key).ArrayU8()

	if err := fastlyCacheInsert(
		keyBuffer.Data, keyBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body.h),
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
//go:wasmimport fastly_cache transaction_lookup
//go:noescape
func fastlyCacheTransactionLookup(
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	mask cacheLookupOptionsMask,
	opts prim.Pointer[cacheLookupOptions],
	h prim.Pointer[cacheHandle],
) FastlyStatus

func CacheTransactionLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	var entry CacheEntry

	keyBuffer := prim.NewReadBufferFromBytes(key).ArrayU8()

	if err := fastlyCacheTransactionLookup(
		keyBuffer.Data, keyBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&entry.h),
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
//go:wasmimport fastly_cache transaction_insert
//go:noescape
func fastlyCacheTransactionInsert(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts prim.Pointer[cacheWriteOptions],
	body prim.Pointer[bodyHandle],
) FastlyStatus

func (c *CacheEntry) Insert(opts CacheWriteOptions) (*HTTPBody, error) {
	body := HTTPBody{closable: true}

	if err := fastlyCacheTransactionInsert(
		c.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body.h),
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
//go:wasmimport fastly_cache transaction_insert_and_stream_back
//go:noescape
func fastlyCacheTransactionInsertAndStreamBack(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts prim.Pointer[cacheWriteOptions],
	body prim.Pointer[bodyHandle],
	stream prim.Pointer[cacheHandle],
) FastlyStatus

func (c *CacheEntry) InsertAndStreamBack(opts CacheWriteOptions) (*HTTPBody, *CacheEntry, error) {
	var entry CacheEntry
	body := HTTPBody{closable: true}

	if err := fastlyCacheTransactionInsertAndStreamBack(
		c.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body.h),
		prim.ToPointer(&entry.h),
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
//go:wasmimport fastly_cache transaction_update
//go:noescape
func fastlyCacheTransactionUpdate(
	h cacheHandle,
	mask cacheWriteOptionsMask,
	opts prim.Pointer[cacheWriteOptions],
) FastlyStatus

func (c *CacheEntry) Update(opts CacheWriteOptions) error {

	return fastlyCacheTransactionUpdate(
		c.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
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
//go:wasmimport fastly_cache transaction_cancel
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
//go:wasmimport fastly_cache close
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
//go:wasmimport fastly_cache get_state
//go:noescape
func fastlyCacheGetState(h cacheHandle, st prim.Pointer[CacheLookupState]) FastlyStatus

func (c *CacheEntry) State() (CacheLookupState, error) {
	var state CacheLookupState

	if err := fastlyCacheGetState(c.h, prim.ToPointer(&state)).toError(); err != nil {
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
//go:wasmimport fastly_cache get_user_metadata
//go:noescape
func fastlyCacheGetUserMetadata(
	h cacheHandle,
	buf prim.Pointer[prim.U8],
	bufLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
) FastlyStatus

func (c *CacheEntry) UserMetadata() ([]byte, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	status := fastlyCacheGetUserMetadata(
		c.h,
		prim.ToPointer(buf.U8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	)
	if status == FastlyStatusBufLen {
		// The buffer was too small, but it'll tell us how big it will
		// need to be in order to fit the content.
		buf = prim.NewWriteBuffer(int(buf.NValue()))

		status = fastlyCacheGetUserMetadata(
			c.h,
			prim.ToPointer(buf.U8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
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
//go:wasmimport fastly_cache get_body
//go:noescape
func fastlyCacheGetBody(
	h cacheHandle,
	mask cacheGetBodyOptionsMask,
	opts prim.Pointer[cacheGetBodyOptions],
	body prim.Pointer[bodyHandle],
) FastlyStatus

func (c *CacheEntry) Body(opts CacheGetBodyOptions) (*HTTPBody, error) {
	var b HTTPBody

	if err := fastlyCacheGetBody(
		c.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&b.h),
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
//go:wasmimport fastly_cache get_length
//go:noescape
func fastlyCacheGetLength(h cacheHandle, l prim.Pointer[prim.U64]) FastlyStatus

func (c *CacheEntry) Length() (uint64, error) {
	var l prim.U64

	if err := fastlyCacheGetLength(c.h, prim.ToPointer(&l)).toError(); err != nil {
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
//go:wasmimport fastly_cache get_max_age_ns
//go:noescape
func fastlyCacheGetMaxAgeNs(h cacheHandle, d prim.Pointer[prim.U64]) FastlyStatus

func (c *CacheEntry) MaxAge() (time.Duration, error) {
	var d prim.U64

	if err := fastlyCacheGetMaxAgeNs(c.h, prim.ToPointer(&d)).toError(); err != nil {
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
//go:wasmimport fastly_cache get_stale_while_revalidate_ns
//go:noescape
func fastlyCacheGetStaleWhileRevalidateNs(h cacheHandle, d prim.Pointer[prim.U64]) FastlyStatus

func (c *CacheEntry) StaleWhileRevalidate() (time.Duration, error) {
	var d prim.U64

	if err := fastlyCacheGetStaleWhileRevalidateNs(c.h, prim.ToPointer(&d)).toError(); err != nil {
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
//go:wasmimport fastly_cache get_age_ns
//go:noescape
func fastlyCacheGetAgeNs(h cacheHandle, d prim.Pointer[prim.U64]) FastlyStatus

func (c *CacheEntry) Age() (time.Duration, error) {
	var d prim.U64

	if err := fastlyCacheGetAgeNs(c.h, prim.ToPointer(&d)).toError(); err != nil {
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
//go:wasmimport fastly_cache get_hits
//go:noescape
func fastlyCacheGetHits(h cacheHandle, d prim.Pointer[prim.U64]) FastlyStatus

func (c *CacheEntry) Hits() (uint64, error) {
	var d prim.U64

	if err := fastlyCacheGetHits(c.h, prim.ToPointer(&d)).toError(); err != nil {
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
//go:wasmimport fastly_purge purge_surrogate_key
//go:noescape
func fastlyPurgeSurrogateKey(surrogateKeyData prim.Pointer[prim.U8], surrogateKeyLen prim.Usize, mask purgeOptionsMask, opts prim.Pointer[purgeOptions]) FastlyStatus

func PurgeSurrogateKey(surrogateKey string, opts PurgeOptions) error {
	surrogateKeyBuffer := prim.NewReadBufferFromString(surrogateKey).Wstring()

	return fastlyPurgeSurrogateKey(
		surrogateKeyBuffer.Data, surrogateKeyBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
	).toError()
}

// witx:
//
//	(module $fastly_device_detection
//	    (@interface func (export "lookup")
//	        (param $user_agent string)
//
//	        (param $buf (@witx pointer (@witx char8)))
//	        (param $buf_len (@witx usize))
//	        (param $nwritten_out (@witx pointer (@witx usize)))
//	        (result $err (expected (error $fastly_status)))
//	    )
//	)
//
//go:wasmimport fastly_device_detection lookup
//go:noescape
func fastlyDeviceDetectionLookup(
	userAgentData prim.Pointer[prim.U8], userAgentLen prim.Usize,
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

func DeviceLookup(userAgent string) ([]byte, error) {
	buf := prim.NewWriteBuffer(defaultBufferLen)

	userAgentBuffer := prim.NewReadBufferFromString(userAgent).Wstring()

	status := fastlyDeviceDetectionLookup(
		userAgentBuffer.Data, userAgentBuffer.Len,
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	)
	if status == FastlyStatusBufLen {
		// The buffer was too small, but it'll tell us how big it will
		// need to be in order to fit the content.
		buf = prim.NewWriteBuffer(int(buf.NValue()))

		status = fastlyDeviceDetectionLookup(
			userAgentBuffer.Data, userAgentBuffer.Len,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	}
	if err := status.toError(); err != nil {
		return nil, err
	}

	return buf.AsBytes(), nil
}

// witx:
//
//	(@interface func (export "check_rate")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $delta u32)
//	    (param $window u32)
//	    (param $limit u32)
//	    (param $pb string)
//	    (param $ttl u32)
//
//	    (result $err (expected $blocked (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl check_rate
//go:noescape
func fastlyERLCheckRate(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	delta prim.U32,
	window prim.U32,
	limit prim.U32,
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	ttl prim.U32,
	blocked prim.Pointer[prim.U32],
) FastlyStatus

func ERLCheckRate(rateCounter, entry string, delta uint32, window RateWindow, limit uint32, penaltyBox string, ttl time.Duration) (bool, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var blocked prim.U32

	if err := fastlyERLCheckRate(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(delta),
		prim.U32(window.value),
		prim.U32(limit),
		pbBuffer.Data, pbBuffer.Len,
		prim.U32(ttl.Seconds()),
		prim.ToPointer(&blocked),
	).toError(); err != nil {
		return false, err
	}

	return blocked != 0, nil
}

// witx:
//
//	(@interface func (export "ratecounter_increment")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $delta u32)
//
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_increment
//go:noescape
func fastlyERLRateCounterIncrement(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	delta prim.U32,
) FastlyStatus

func RateCounterIncrement(rateCounter, entry string, delta uint32) error {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	return fastlyERLRateCounterIncrement(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(delta),
	).toError()
}

// witx:
//
//	(@interface func (export "ratecounter_lookup_rate")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $window u32)
//
//	    (result $err (expected $rate (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_lookup_rate
//go:noescape
func fastlyERLRateCounterLookupRate(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	window prim.U32,
	rate prim.Pointer[prim.U32],
) FastlyStatus

func RateCounterLookupRate(rateCounter, entry string, window RateWindow) (uint32, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var rate prim.U32

	if err := fastlyERLRateCounterLookupRate(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(window.value),
		prim.ToPointer(&rate),
	).toError(); err != nil {
		return 0, err
	}

	return uint32(rate), nil
}

// witx:
//
//	(@interface func (export "ratecounter_lookup_count")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $duration u32)
//
//	    (result $err (expected $count (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_lookup_count
//go:noescape
func fastlyERLRateCounterLookupCount(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	duration prim.U32,
	count prim.Pointer[prim.U32],
) FastlyStatus

func RateCounterLookupCount(rateCounter, entry string, duration CounterDuration) (uint32, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var count prim.U32

	if err := fastlyERLRateCounterLookupCount(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(duration.value),
		prim.ToPointer(&count),
	).toError(); err != nil {
		return 0, err
	}

	return uint32(count), nil
}

// witx:
//
//	(@interface func (export "penaltybox_add")
//	    (param $pb string)
//	    (param $entry string)
//	    (param $ttl u32)
//
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl penaltybox_add
//go:noescape
func fastlyERLPenaltyBoxAdd(
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	ttl prim.U32,
) FastlyStatus

func PenaltyBoxAdd(penaltyBox, entry string, ttl time.Duration) error {
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	return fastlyERLPenaltyBoxAdd(
		pbBuffer.Data, pbBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(ttl.Seconds()),
	).toError()
}

// witx:
//
//	(@interface func (export "penaltybox_has")
//	    (param $pb string)
//	    (param $entry string)
//
//	    (result $err (expected $has (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl penaltybox_has
//go:noescape
func fastlyERLPenaltyBoxHas(
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	has prim.Pointer[prim.U32],
) FastlyStatus

func PenaltyBoxHas(penaltyBox, entry string) (bool, error) {
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var has prim.U32

	if err := fastlyERLPenaltyBoxHas(
		pbBuffer.Data, pbBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.ToPointer(&has),
	).toError(); err != nil {
		return false, err
	}

	return has != 0, nil
}
