//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls
// +build tinygo.wasm,wasi wasip1
// +build !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net"

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
func fastlyUAPParse(userAgent_Data uint32, userAgent_Len uint32, family *prim.Char8,
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
	patchNWrittenOut *prim.Usize) FastlyStatus

// ParseUserAgent parses the user agent string into its component parts.
func ParseUserAgent(userAgent string) (family, major, minor, patch string, err error) {
	var (
		cap       = len(userAgent)
		familyBuf = prim.NewWriteBuffer(cap)
		majorBuf  = prim.NewWriteBuffer(cap)
		minorBuf  = prim.NewWriteBuffer(cap)
		patchBuf  = prim.NewWriteBuffer(cap)
	)

	userAgent_ws := prim.NewReadBufferFromString(userAgent).Wstring()
	if err := fastlyUAPParse(userAgent_ws.Data, userAgent_ws.Len, familyBuf.Char8Pointer(),
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
		patchBuf.NPointer()).toError(); err != nil {
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
//go:wasmimport fastly_http_body read
//go:noescape
func fastlyHTTPBodyRead(
	h uint32,
	buf *prim.U8,
	bufLen prim.Usize,
	nRead *prim.Usize,
) FastlyStatus

// Read implements io.Reader, reading up to len(p) bytes from the body into p.
// Returns the number of bytes read, and any error encountered.
func (b *HTTPBody) Read(p []byte) (int, error) {
	buf := prim.NewWriteBufferFromBytes(p)
	if err := fastlyHTTPBodyRead(
		uint32(b.h),
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
//go:wasmimport fastly_http_body write
//go:noescape
func fastlyHTTPBodyWrite(h bodyHandle, buf_Data uint32, buf_Len uint32, end bodyWriteEnd,
	nWritten *prim.Usize) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the body.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (b *HTTPBody) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		p_u8 := prim.NewReadBufferFromBytes(p[n:]).ArrayU8()
		if err = fastlyHTTPBodyWrite(b.h, p_u8.Data, p_u8.Len, bodyWriteEndBack,
			&nWritten).toError(); err == nil {
			n += int(nWritten)
		}
	}
	return n, err
}

// witx:
//
//	(@interface func (export "close")
//	  (param $h $body_handle)
//	  (result $err $fastly_status)
//	)
//
// )
//
//go:wasmimport fastly_http_body close
//go:noescape
func fastlyHTTPBodyClose(
	h bodyHandle,
) FastlyStatus

// Close the body. Once closed, a body cannot be used again.
// Close is a no-op unless the body's "streaming bit" is set.
func (b *HTTPBody) Close() error {
	if !b.closable {
		return nil
	}

	return fastlyHTTPBodyClose(
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
func fastlyLogEndpointGet(name_Data uint32, name_Len uint32, endpointHandleOut *endpointHandle) FastlyStatus

// GetLogEndpoint opens the log endpoint identified by name.
func GetLogEndpoint(name string) (*LogEndpoint, error) {
	var e LogEndpoint

	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	if err := fastlyLogEndpointGet(name_u8.Data, name_u8.Len, &e.h).toError(); err != nil {
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
func fastlyLogWrite(h endpointHandle, msg_Data uint32, msg_Len uint32, nWrittenOut *prim.Usize) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the endpoint.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (e *LogEndpoint) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		p_u8 := prim.NewReadBufferFromBytes(p[n:]).ArrayU8()
		if err = fastlyLogWrite(e.h, p_u8.Data, p_u8.Len, &nWritten).toError(); err == nil {
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
func fastlyHTTPReqCacheOverrideV2Set(h requestHandle,
	tag cacheOverrideTag,
	ttl prim.U32,
	staleWhileRevalidate prim.U32, sk_Data uint32, sk_Len uint32) FastlyStatus

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

	options_SurrogateKey_u8 := prim.NewReadBufferFromString(options.SurrogateKey).ArrayU8()
	return fastlyHTTPReqCacheOverrideV2Set(r.h,
		tag,
		prim.U32(options.TTL),
		prim.U32(options.StaleWhileRevalidate), options_SurrogateKey_u8.Data, options_SurrogateKey_u8.Len).toError()
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
//go:wasmimport fastly_http_req downstream_tls_cipher_openssl_name
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
//go:wasmimport fastly_http_req downstream_tls_protocol
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
//go:wasmimport fastly_http_req downstream_tls_client_hello
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
//go:wasmimport fastly_http_req new
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
//go:wasmimport fastly_http_req header_names_get
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
//go:wasmimport fastly_http_req original_header_names_get
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
//go:wasmimport fastly_http_req original_header_count
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
//go:wasmimport fastly_http_req header_value_get
//go:noescape
func fastlyHTTPReqHeaderValueGet(h requestHandle, name_Data uint32, name_Len uint32, value *prim.Char8,
	valueMaxLen prim.Usize,
	nwrittenOut *prim.Usize) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// request, if any.
func (r *HTTPRequest) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	if err := fastlyHTTPReqHeaderValueGet(r.h, name_u8.Data, name_u8.Len, buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer()).toError(); err != nil {
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
func fastlyHTTPReqHeaderValuesGet(h requestHandle, name_Data uint32, name_Len uint32, buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize) FastlyStatus

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
		name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
		return fastlyHTTPReqHeaderValuesGet(r.h, name_u8.Data, name_u8.Len, buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut)
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
func fastlyHTTPReqHeaderValuesSet(h requestHandle, name_Data uint32, name_Len uint32, values_Data uint32, values_Len uint32) FastlyStatus

// SetHeaderValues sets the provided header(s) on the request.
func (r *HTTPRequest) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		fmt.Fprint(&buf, value)
		buf.WriteByte(0)
	}

	buf_a8 := prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8()
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPReqHeaderValuesSet(r.h, name_u8.Data, name_u8.Len, buf_a8.Data, buf_a8.Len).toError()
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
func fastlyHTTPReqHeaderInsert(h requestHandle, name_Data uint32, name_Len uint32, value_Data uint32, value_Len uint32) FastlyStatus

// InsertHeader adds the provided header to the request.
func (r *HTTPRequest) InsertHeader(name, value string) error {
	value_u8 := prim.NewReadBufferFromString(value).ArrayU8()
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPReqHeaderInsert(r.h, name_u8.Data, name_u8.Len, value_u8.Data, value_u8.Len).toError()
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
func fastlyHTTPReqHeaderAppend(h requestHandle, name_Data uint32, name_Len uint32, value_Data uint32, value_Len uint32) FastlyStatus

// AppendHeader adds the provided header to the request.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPRequest) AppendHeader(name, value string) error {
	value_u8 := prim.NewReadBufferFromString(value).ArrayU8()
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPReqHeaderAppend(r.h, name_u8.Data, name_u8.Len, value_u8.Data, value_u8.Len).toError()
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
func fastlyHTTPReqHeaderRemove(h requestHandle, name_Data uint32, name_Len uint32) FastlyStatus

// RemoveHeader removes the named header(s) from the request.
func (r *HTTPRequest) RemoveHeader(name string) error {
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPReqHeaderRemove(r.h, name_u8.Data, name_u8.Len).toError()
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
//go:wasmimport fastly_http_req method_set
//go:noescape
func fastlyHTTPReqMethodSet(h requestHandle, method_Data uint32, method_Len uint32) FastlyStatus

// SetMethod sets the HTTP method of the request.
func (r *HTTPRequest) SetMethod(method string) error {
	method_ws := prim.NewReadBufferFromString(method).Wstring()
	return fastlyHTTPReqMethodSet(r.h, method_ws.Data, method_ws.Len).toError()
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
//go:wasmimport fastly_http_req uri_set
//go:noescape
func fastlyHTTPReqURISet(h requestHandle, uri_Data uint32, uri_Len uint32) FastlyStatus

// SetURI sets the request's fully qualified URI.
func (r *HTTPRequest) SetURI(uri string) error {
	uri_ws := prim.NewReadBufferFromString(uri).Wstring()
	return fastlyHTTPReqURISet(r.h, uri_ws.Data, uri_ws.Len).toError()
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
//	(@interface func (export "send")
//	   (param $h $request_handle)
//	   (param $b $body_handle)
//	   (param $backend string)
//	   (result $err $fastly_status)
//	   (result $resp $response_handle)
//	   (result $resp_body $body_handle)
//	)
//
//go:wasmimport fastly_http_req send
//go:noescape
func fastlyHTTPReqSend(h requestHandle,
	b bodyHandle, backend_Data uint32, backend_Len uint32, resp *responseHandle,
	respBody *bodyHandle) FastlyStatus

// Send the request, with the provided body, to the named backend. The body is
// buffered and sent all at once. Blocks until the request is complete, and
// returns the response and response body, or an error.
func (r *HTTPRequest) Send(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		resp     HTTPResponse
		respBody HTTPBody
	)

	backend_ws := prim.NewReadBufferFromString(backend).Wstring()
	if err := fastlyHTTPReqSend(
		r.h,
		requestBody.h,
		backend_ws.Data, backend_ws.Len,
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
//go:wasmimport fastly_http_req send_async
//go:noescape
func fastlyHTTPReqSendAsync(h requestHandle,
	b bodyHandle, backend_Data uint32, backend_Len uint32, pendingReq *pendingRequestHandle) FastlyStatus

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

	backend_ws := prim.NewReadBufferFromString(backend).Wstring()
	if err := fastlyHTTPReqSendAsync(
		r.h,
		requestBody.h,
		backend_ws.Data, backend_ws.Len,
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
//go:wasmimport fastly_http_req send_async_streaming
//go:noescape
func fastlyHTTPReqSendAsyncStreaming(h requestHandle,
	b bodyHandle, backend_Data uint32, backend_Len uint32, pendingReq *pendingRequestHandle) FastlyStatus

// SendAsyncStreaming sends the request, with the provided body, to the named
// backend. Unlike Send or SendAsync, the request body is streamed, rather than
// buffered and sent all at once. Returns immediately with a reference to the
// newly created request.
func (r *HTTPRequest) SendAsyncStreaming(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	var pendingReq PendingRequest

	backend_ws := prim.NewReadBufferFromString(backend).Wstring()
	if err := fastlyHTTPReqSendAsyncStreaming(
		r.h,
		requestBody.h,
		backend_ws.Data, backend_ws.Len,
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
//go:wasmimport fastly_http_req pending_req_poll
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
//go:wasmimport fastly_http_req pending_req_wait
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
/*
   //go:wasmimport fastly_http_req pending_req_select
   //go:noescape
*/
func fastlyHTTPReqPendingReqSelect(
	hs []pendingRequestHandle, // TODO(pb): is correct?
	doneIdx *prim.U32,
	resp *responseHandle,
	respBody *bodyHandle,
) FastlyStatus {
	return FastlyStatusOK
}

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
//	(module $fastly_http_resp
//	   (@interface func (export "new")
//	     (result $err $fastly_status)
//	     (result $h $response_handle)
//	   )
//
//go:wasmimport fastly_http_resp new
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
//go:wasmimport fastly_http_resp header_names_get
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
//go:wasmimport fastly_http_resp header_value_get
//go:noescape
func fastlyHTTPRespHeaderValueGet(h responseHandle, name_Data uint32, name_Len uint32, value *prim.Char8,
	valueMaxLen prim.Usize,
	nwrittenOut *prim.Usize) FastlyStatus

// GetHeaderValue returns the first header value of the given header name on the
// response, if any.
func (r *HTTPResponse) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	buf := prim.NewWriteBuffer(maxHeaderValueLen)
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	if err := fastlyHTTPRespHeaderValueGet(r.h, name_u8.Data, name_u8.Len, buf.Char8Pointer(),
		buf.Cap(),
		buf.NPointer()).toError(); err != nil {
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
func fastlyHTTPRespHeaderValuesGet(h responseHandle, name_Data uint32, name_Len uint32, buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize) FastlyStatus

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
		name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
		return fastlyHTTPRespHeaderValuesGet(r.h, name_u8.Data, name_u8.Len, buf,
			bufLen,
			cursor,
			endingCursorOut,
			nwrittenOut)
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
func fastlyHTTPRespHeaderValuesSet(h responseHandle, name_Data uint32, name_Len uint32, values_Data uint32, values_Len uint32) FastlyStatus

// SetHeaderValues sets the provided header(s) on the response.
//
// TODO(pb): does this overwrite any existing name headers?
func (r *HTTPResponse) SetHeaderValues(name string, values []string) error {
	var buf bytes.Buffer
	for _, value := range values {
		fmt.Fprint(&buf, value)
		buf.WriteByte(0)
	}

	buf_a8 := prim.NewReadBufferFromBytes(buf.Bytes()).ArrayChar8()
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPRespHeaderValuesSet(r.h, name_u8.Data, name_u8.Len, buf_a8.Data, buf_a8.Len).toError()
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
func fastlyHTTPRespHeaderInsert(h responseHandle, name_Data uint32, name_Len uint32, value_Data uint32, value_Len uint32) FastlyStatus

// InsertHeader adds the provided header to the response.
func (r *HTTPResponse) InsertHeader(name, value string) error {
	var (
		nameBuf  = prim.NewReadBufferFromString(name)
		valueBuf = prim.NewReadBufferFromString(value)
	)

	nameBuf_u8 := nameBuf.ArrayU8()
	valueBuf_u8 := valueBuf.ArrayU8()

	return fastlyHTTPRespHeaderInsert(
		r.h,
		nameBuf_u8.Data, nameBuf_u8.Len,
		valueBuf_u8.Data, valueBuf_u8.Len,
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
func fastlyHTTPRespHeaderAppend(h responseHandle, name_Data uint32, name_Len uint32, value_Data uint32, value_Len uint32) FastlyStatus

// AppendHeader adds the provided header to the response.
//
// TODO(pb): what is the difference to InsertHeader?
func (r *HTTPResponse) AppendHeader(name, value string) error {
	value_u8 := prim.NewReadBufferFromString(value).ArrayU8()
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPRespHeaderAppend(r.h, name_u8.Data, name_u8.Len, value_u8.Data, value_u8.Len).toError()
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
func fastlyHTTPRespHeaderRemove(h responseHandle, name_Data uint32, name_Len uint32) FastlyStatus

// RemoveHeader removes the named header(s) from the response.
func (r *HTTPResponse) RemoveHeader(name string) error {
	name_u8 := prim.NewReadBufferFromString(name).ArrayU8()
	return fastlyHTTPRespHeaderRemove(r.h, name_u8.Data, name_u8.Len).toError()
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
	status *uint32,
) FastlyStatus

// GetStatusCode returns the status code of the response.
func (r *HTTPResponse) GetStatusCode() (int, error) {
	var status uint32
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
//go:wasmimport fastly_http_resp status_set
//go:noescape
func fastlyHTTPRespStatusSet(
	h responseHandle,
	status uint32,
) FastlyStatus

// SetStatusCode sets the status code of the response.
func (r *HTTPResponse) SetStatusCode(code int) error {
	status := uint32(code)
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
func fastlyDictionaryOpen(name_Data uint32, name_Len uint32, h *dictionaryHandle) FastlyStatus

// Dictionary represents a Fastly edge dictionary, a collection of read-only
// key/value pairs. For convenience, keys are modeled as Go strings, and values
// as byte slices.
type Dictionary struct {
	h dictionaryHandle
}

// OpenDictionary returns a reference to the named dictionary, if it exists.
func OpenDictionary(name string) (*Dictionary, error) {
	var d Dictionary

	name_ws := prim.NewReadBufferFromString(name).Wstring()
	if err := fastlyDictionaryOpen(name_ws.Data, name_ws.Len, &d.h).toError(); err != nil {
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
func fastlyDictionaryGet(h dictionaryHandle, key_Data uint32, key_Len uint32, value *prim.Char8,
	valueMaxLen prim.Usize,
	nWritten *prim.Usize) FastlyStatus

// Get the value for key, if it exists.
func (d *Dictionary) Get(key string) (string, error) {
	buf := prim.NewWriteBuffer(dictionaryValueMaxLen)
	key_ws := prim.NewReadBufferFromString(key).Wstring()
	if err := fastlyDictionaryGet(
		d.h,
		key_ws.Data, key_ws.Len,
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
//go:wasmimport fastly_geo lookup
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

//go:wasmimport fastly_object_store open
//go:noescape
func fastlyObjectStoreOpen(name_Data uint32, name_Len uint32, h *objectStoreHandle) FastlyStatus

// ObjectStore represents a Fastly object store, a collection of key/value pairs.
// For convenience, keys and values are both modelled as Go strings.
type ObjectStore struct {
	h objectStoreHandle
}

// ObjectStoreOpen returns a reference to the named object store, if it exists.
func OpenObjectStore(name string) (*ObjectStore, error) {
	var o ObjectStore

	name_ws := prim.NewReadBufferFromString(name).Wstring()
	if err := fastlyObjectStoreOpen(name_ws.Data, name_ws.Len, &o.h).toError(); err != nil {
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
func fastlyObjectStoreLookup(h objectStoreHandle, key_Data uint32, key_Len uint32, b *bodyHandle) FastlyStatus

// Lookup returns the value for key, if it exists.
func (o *ObjectStore) Lookup(key string) (io.Reader, error) {
	body := HTTPBody{h: invalidBodyHandle}

	key_ws := prim.NewReadBufferFromString(key).Wstring()
	if err := fastlyObjectStoreLookup(
		o.h,
		key_ws.Data, key_ws.Len,
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

//go:wasmimport fastly_object_store insert
//go:noescape
func fastlyObjectStoreInsert(h objectStoreHandle, key_Data uint32, key_Len uint32, b bodyHandle) FastlyStatus

// Insert adds a key/value pair to the object store.
func (o *ObjectStore) Insert(key string, value io.Reader) error {
	body, err := NewHTTPBody()
	if err != nil {
		return err
	}

	if _, err := io.Copy(body, value); err != nil {
		return err
	}

	key_ws := prim.NewReadBufferFromString(key).Wstring()
	if err := fastlyObjectStoreInsert(
		o.h,
		key_ws.Data, key_ws.Len,
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
func fastlySecretStoreOpen(name_Data uint32, name_Len uint32, h *secretStoreHandle) FastlyStatus

// OpenSecretStore returns a reference to the named secret store, if it exists.
func OpenSecretStore(name string) (*SecretStore, error) {
	var st SecretStore

	name_ws := prim.NewReadBufferFromString(name).Wstring()
	if err := fastlySecretStoreOpen(name_ws.Data, name_ws.Len, &st.h).toError(); err != nil {
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
func fastlySecretStoreGet(h secretStoreHandle, key_Data uint32, key_Len uint32, s *secretHandle) FastlyStatus

// Get returns a handle to the secret value for the given name, if it
// exists.
func (st *SecretStore) Get(name string) (*Secret, error) {
	var s Secret

	name_ws := prim.NewReadBufferFromString(name).Wstring()
	if err := fastlySecretStoreGet(
		st.h,
		name_ws.Data, name_ws.Len,
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

//go:wasmimport fastly_secret_store plaintext
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
