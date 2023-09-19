//lint:file-ignore U1000 Ignore all unused code
//revive:disable:exported

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

type handle uint32

// FastlyStatus models a response status enum.
type FastlyStatus uint32

// witx:
//    (typename $fastly_status
//    	(enum u32
//          $ok
//          $error
//          $inval
//          $badf
//          $buflen
//          $unsupported
//          $badalign
//          $httpinvalid
//          $httpuser
//          $httpincomplete
//          $none
//          $httpheadtoolarge
//          $httpinvalidstatus
//          $limitexceeded))

const (
	// FastlyStatusOK maps to $fastly_status $ok.
	// TODO(pb): is this the only non-error status?
	FastlyStatusOK FastlyStatus = 0

	// FastlyStatusError maps to $fastly_status $error.
	FastlyStatusError FastlyStatus = 1

	// FastlyStatusInval maps to $fastly_status $inval.
	FastlyStatusInval FastlyStatus = 2

	// FastlyStatusBadf maps to $fastly_status $badf.
	FastlyStatusBadf FastlyStatus = 3

	// FastlyStatusBufLen maps to $fastly_status $buflen.
	FastlyStatusBufLen FastlyStatus = 4

	// FastlyStatusUnsupported maps to $fastly_status $unsupported.
	FastlyStatusUnsupported FastlyStatus = 5

	// FastlyStatusBadAlign maps to $fastly_status $badalign.
	FastlyStatusBadAlign FastlyStatus = 6

	// FastlyStatusHTTPInvalid maps to $fastly_status $httpinvalid.
	FastlyStatusHTTPInvalid FastlyStatus = 7

	// FastlyStatusHTTPUser maps to $fastly_status $httpuser.
	FastlyStatusHTTPUser FastlyStatus = 8

	// FastlyStatusHTTPIncomplete maps to $fastly_status $httpincomplete.
	FastlyStatusHTTPIncomplete FastlyStatus = 9

	// FastlyStatusNone maps to $fastly_status $none.
	FastlyStatusNone FastlyStatus = 10

	// FastlyStatusHTTPHeadTooLarge maps to $fastly_status $httpheadtoolarge.
	FastlyStatusHTTPHeadTooLarge FastlyStatus = 11

	// FastlyStatusHTTPInvalidStatus maps to $fastly_status $httpinvalidstatus.
	FastlyStatusHTTPInvalidStatus FastlyStatus = 12

	// FastlyStatusLimitExceeded maps to $fastly_status $limitexceeded.
	FastlyStatusLimitExceeded FastlyStatus = 13
)

// String implements fmt.Stringer.
func (s FastlyStatus) String() string {
	switch s {
	case FastlyStatusOK:
		return "OK"
	case FastlyStatusError:
		return "Error"
	case FastlyStatusInval:
		return "Inval"
	case FastlyStatusBadf:
		return "Badf"
	case FastlyStatusBufLen:
		return "BufLen"
	case FastlyStatusUnsupported:
		return "Unsupported"
	case FastlyStatusBadAlign:
		return "BadAlign"
	case FastlyStatusHTTPInvalid:
		return "HTTPInvalid"
	case FastlyStatusHTTPUser:
		return "HTTPUser"
	case FastlyStatusHTTPIncomplete:
		return "HTTPIncomplete"
	case FastlyStatusNone:
		return "None"
	case FastlyStatusHTTPHeadTooLarge:
		return "HTTPHeadTooLarge"
	case FastlyStatusHTTPInvalidStatus:
		return "HTTPInvalidStatus"
	case FastlyStatusLimitExceeded:
		return "LimitExceeded"
	default:
		return fmt.Sprintf("FastlyStatus(%d)", s)
	}
}

func (s FastlyStatus) toError() error {
	switch s {
	case FastlyStatusOK:
		return nil
	default:
		return FastlyError{Status: s}
	}
}

func (s FastlyStatus) toErrorDetailed(d sendErrorDetail) error {
	switch s {
	case FastlyStatusOK:
		return nil
	default:
		return FastlyError{Status: s, detail: d}
	}
}

// FastlyError decorates error-class FastlyStatus values and implements the
// error interface.
//
// Note that TinyGo currently doesn't support errors.As. Callers can use the
// IsFastlyError helper instead.
type FastlyError struct {
	Status FastlyStatus
	detail sendErrorDetail
}

// Error implements the error interface.
func (e FastlyError) Error() string {
	if detail := e.detail.String(); detail != "" {
		return fmt.Sprintf("Fastly error: send error: %s", detail)
	}

	return fmt.Sprintf("Fastly error: %s", e.Status.String())
}

func (e FastlyError) getStatus() FastlyStatus {
	return e.Status
}

// IsFastlyError detects and unwraps a FastlyError to its component parts.
func IsFastlyError(err error) (FastlyStatus, bool) {
	for {
		switch e := err.(type) {
		case interface{ getStatus() FastlyStatus }:
			return e.getStatus(), true

		case interface{ Unwrap() error }:
			err = e.Unwrap()
			if err == nil {
				return 0, false
			}

		default:
			return 0, false
		}
	}
}

// HTTPVersion describes an HTTP protocol version.
type HTTPVersion uint32

// witx:
//  (typename $http_version
// 	(enum u32
// 	  $http_09
// 	  $http_10
// 	  $http_11
// 	  $h2
// 	  $h3))

const (
	// HTTPVersionHTTP09 describes HTTP/0.9.
	HTTPVersionHTTP09 HTTPVersion = 0

	// HTTPVersionHTTP10 describes HTTP/1.0.
	HTTPVersionHTTP10 HTTPVersion = 1

	// HTTPVersionHTTP11 describes HTTP/1.1.
	HTTPVersionHTTP11 HTTPVersion = 2

	// HTTPVersionH2 describes HTTP/2.
	HTTPVersionH2 HTTPVersion = 3

	// HTTPVersionH3 describes HTTP/3.
	HTTPVersionH3 HTTPVersion = 4
)

func (v HTTPVersion) splat() (proto string, major, minor int, err error) {
	switch v {
	case HTTPVersionHTTP09:
		return "HTTP/0.9", 0, 9, nil
	case HTTPVersionHTTP10:
		return "HTTP/1.0", 1, 0, nil
	case HTTPVersionHTTP11:
		return "HTTP/1.1", 1, 1, nil
	case HTTPVersionH2:
		return "HTTP/2.0", 2, 0, nil
	case HTTPVersionH3:
		return "HTTP/3.0", 3, 0, nil
	default:
		return "", 0, 0, fmt.Errorf("unknown protocol version %d", v)
	}
}

// witx:
//
//	(typename $http_status u16)
type httpStatus uint32

// witx:
//
//	 (typename $body_write_end
//		(enum u32
//		  $back
//		  $front))
type bodyWriteEnd uint32

const (
	bodyWriteEndBack  bodyWriteEnd = 0 // $back
	bodyWriteEndFront bodyWriteEnd = 1 // $front
)

// witx:
//
//	(typename $body_handle (handle))
type bodyHandle handle

const (
	invalidBodyHandle = bodyHandle(math.MaxUint32 - 1)
)

// witx:
//
//	(typename $request_handle (handle))
type requestHandle handle

// witx:
//
//	(typename $response_handle (handle))
type responseHandle handle

// witx:
//
//	(typename $pending_request_handle (handle))
type pendingRequestHandle handle

// witx:
//
//	(typename $endpoint_handle (handle))
type endpointHandle handle

// witx:
//
//	(typename $dictionary_handle (handle))
type dictionaryHandle handle

// witx:
//
//	(typename $multi_value_cursor u32)
type multiValueCursor uint32

// witx:
//
//	(typename $multi_value_cursor_result s64)
type multiValueCursorResult int64

// -1 represents "finished", non-negative represents a $multi_value_cursor:
func (r multiValueCursorResult) isFinished() bool { return r < 0 }

func (r multiValueCursorResult) toCursor() multiValueCursor { return multiValueCursor(r) }

// witx:
//
//	 (typename $cache_override_tag
//		(flags u32
//		  $pass
//		  $ttl
//		  $stale_while_revalidate
//		  $pci))
type cacheOverrideTag uint32

const (
	cacheOverrideTagNone                 cacheOverrideTag = 0b0000_0000
	cacheOverrideTagPass                 cacheOverrideTag = 0b0000_0001 // $pass
	cacheOverrideTagTTL                  cacheOverrideTag = 0b0000_0010 // $ttl
	cacheOverrideTagStaleWhileRevalidate cacheOverrideTag = 0b0000_0100 // $stale_while_revalidate
	cacheOverrideTagPCI                  cacheOverrideTag = 0b0000_1000 // $pci
)

const (
	// DefaultMaxHeaderNameLen is the default header name length limit
	DefaultMaxHeaderNameLen = 8192
	// DefaultMaxHeaderValueLen is the default header value length limit
	DefaultMaxHeaderValueLen = 8192
	// DefaultMaxMethodLen is the default method length limit
	DefaultMaxMethodLen = 1024
	// DefaultMaxURLLen is the default URL length limit
	DefaultMaxURLLen = 8192

	dictionaryValueMaxLen = 8192 // https://docs.fastly.com/en/guides/about-edge-dictionaries#limitations-and-considerations
	defaultBufferLen      = 16 * 1024

	initialSecretLen = 1024
)

// CacheOverrideOptions collects specific, caching-related options for outbound
// requests. See the equivalent CacheOverrideOptions type in package fsthttp for
// more detailed descriptions of each field.
type CacheOverrideOptions struct {
	Pass                 bool
	PCI                  bool
	TTL                  uint32 // seconds
	StaleWhileRevalidate uint32 // seconds
	SurrogateKey         string
}

// multiValueHostcall partially models hostcalls that provide an iterator-like
// API over multiple values. Callers need to write a small, probably inline
// adapter function from their specific hostcall to this more general form.
type multiValueHostcall func(
	buf *prim.Char8,
	bufLen prim.Usize,
	cursor multiValueCursor,
	endingCursorOut *multiValueCursorResult,
	nwrittenOut *prim.Usize,
) FastlyStatus

// Values is the result of a multi-value hostcall. It offers an iterator API
// similar to bufio.Scanner or sql.Rows.
type Values struct {
	f        multiValueHostcall
	buffer   []byte           // written-to by hostcalls
	cursor   multiValueCursor //
	pending  []byte           // sliding window over buffer: result of most recent hostcall
	value    []byte           // sliding window over pending: extracted by Next, returned by Bytes
	finished bool             // no more hostcalls please
	err      error            //
}

// newValuesBuffer constructs a Values iterator over the provided hostcall. The buffer
// is used to receive writes from the hostcall. It must be large enough to avoid
// BufLen errors.
func newValuesBuffer(f multiValueHostcall, buffer []byte) *Values {
	return &Values{
		f:      f,
		buffer: buffer,
	}
}

// newValues is a helper constructor that allocates a buffer of capacity cap and
// provides it to newValuesBuffer.
func newValues(f multiValueHostcall, cap int) *Values {
	return newValuesBuffer(f, make([]byte, 0, cap))
}

// Next prepares the next value for reading with the Bytes method. It returns
// true on success, or false if there are no more values, or an error occurred.
// Err should be called to distinguish between those two cases. Every call to
// Bytes, even the first one, must be preceded by a call to Next.
func (v *Values) Next() bool {
	var (
		haveError         = v.err != nil
		hostcallsFinished = v.finished
		nothingPending    = len(v.pending) == 0
	)
	if haveError || (hostcallsFinished && nothingPending) {
		return false
	}

	// 1. Make the hostcall and have it write into v.buffer.
	// 2. Set v.pending to v.buffer, another "view" to the same backing array.
	// 3. Slide v.pending forward, value by value, for each call to Next.
	// 4. All values are consumed when len(v.pending) == 0.
	// 5. Repeat until the hostcall returns finished.
	//
	// We assume the hostcall always writes complete values to the buffer, never
	// splitting a value over multiple calls. Said another way: we assume every
	// value ends with a terminator.

	if nothingPending {
		var (
			buf    = prim.NewWriteBufferFromBytes(v.buffer)
			result = multiValueCursorResult(0)
		)
		if err := v.f(
			buf.Char8Pointer(),
			buf.Cap(),
			v.cursor,
			&result,
			buf.NPointer(),
		).toError(); err != nil {
			v.finished, v.err = true, err
			return false
		}

		// If nothing was written, we're done.
		if buf.NValue() == 0 {
			v.finished = true
			return false
		}

		// If we're finished, no more hostcalls, please.
		// Otherwise, update the cursor for the next hostcall.
		if result.isFinished() {
			v.finished = true
		} else {
			v.cursor = result.toCursor()
		}

		// Capture the result.
		v.pending = buf.AsBytes()
	}

	// Pending buffer has something.
	// Find the first terminator.
	idx := bytes.IndexByte(v.pending, 0)
	if idx < 0 {
		v.err = fmt.Errorf("missing terminator")
		return false
	}

	// Capture the first value.
	v.value = v.pending[:idx]

	// Slide the pending window forward.
	v.pending = v.pending[idx+1:] // +1 for terminator

	// We've got something.
	return true
}

// Err returns the error, if any, that was encountered during iteration.
func (v *Values) Err() error {
	return v.err
}

// Bytes returns the most recent value generated by a call to Next. The
// underlying array may point to data that will be overwritten by a subsequent
// call to Next. Bytes performs no allocation.
func (v *Values) Bytes() []byte {
	return v.value
}

// witx:
//
//	(typename $content_encodings
//	  (flags (@witx repr u32)
//	      $gzip))
type contentEncodings prim.U32

const (
	contentsEncodingsGzip contentEncodings = 0b0000_0001 // $gzip
)

// AutoDecompressResponseOptions collects the auto decompress response options
// for the request. See the equivalent DecompressResponseOptions type in package
// fsthttp for more detailed descriptions of each field.
type AutoDecompressResponseOptions struct {
	Gzip bool
}

// witx:
//
//	(typename $framing_headers_mode
//	   (enum (@witx tag u32)
//	       $automatic
//	       $manually_from_headers))
type framingHeadersMode prim.U32

const (
	framingHeadersModeAutomatic           framingHeadersMode = 0 // $automatic
	framingHeadersModeManuallyFromHeaders framingHeadersMode = 1 // $manually_from_headers
)

// witx:
//
//	(typename $object_store_handle (handle))
type objectStoreHandle handle

// witx:
//
//	(typename $secret_store_handle (handle))
//	(typename $secret_handle (handle))
type (
	secretStoreHandle handle
	secretHandle      handle
)

// witx:
//
//	;;; The outcome of a cache lookup (either bare or as part of a cache transaction)
//	(typename $cache_handle (handle))
type cacheHandle handle

// witx:
//
//	;;; Extensible options for cache lookup operations; currently used for both `lookup` and `transaction_lookup`.
//	(typename $cache_lookup_options
//	    (record
//	        (field $request_headers $request_handle) ;; a full request handle, but used only for its headers
//	    )
//	)
type cacheLookupOptions struct {
	requestHeaders requestHandle
}

// witx:
//
//	(typename $cache_lookup_options_mask
//	    (flags (@witx repr u32)
//	        $reserved
//	        $request_headers
//	    )
//	)
type cacheLookupOptionsMask prim.U32

const (
	cacheLookupOptionsMaskReserved       cacheLookupOptionsMask = 0b0000_0001 // $reserved
	cacheLookupOptionsMaskRequestHeaders cacheLookupOptionsMask = 0b0000_0010 // $request_headers
)

// witx:
//
//	(typename $cache_object_length u64)
//	(typename $cache_duration_ns u64)
//	(typename $cache_hit_count u64)
//
//	;;; Configuration for several hostcalls that write to the cache:
//	;;; - `insert`
//	;;; - `transaction_insert`
//	;;; - `transaction_insert_and_stream_back`
//	;;; - `transaction_update`
//	;;;
//	;;; Some options are only allowed for certain of these hostcalls; see `cache_write_options_mask`.
//	(typename $cache_write_options
//	    (record
//	        (field $max_age_ns $cache_duration_ns) ;; this is a required field; there's no flag for it
//	        (field $request_headers $request_handle) ;; a full request handle, but used only for its headers
//	        (field $vary_rule_ptr (@witx pointer (@witx char8))) ;; a list of header names separated by spaces
//	        (field $vary_rule_len (@witx usize))
//	        ;; The initial age of the object in nanoseconds (default: 0).
//	        ;;
//	        ;; This age is used to determine the freshness lifetime of the object as well as to
//	        ;; prioritize which variant to return if a subsequent lookup matches more than one vary rule
//	        (field $initial_age_ns $cache_duration_ns)
//	        (field $stale_while_revalidate_ns $cache_duration_ns)
//	        (field $surrogate_keys_ptr (@witx pointer (@witx char8))) ;; a list of surrogate keys separated by spaces
//	        (field $surrogate_keys_len (@witx usize))
//	        (field $length $cache_object_length)
//	        (field $user_metadata_ptr (@witx pointer (@witx u8)))
//	        (field $user_metadata_len (@witx usize))
//	    )
//	)
type cacheWriteOptions struct {
	maxAgeNs               prim.U64
	requestHeaders         requestHandle
	varyRulePtr            prim.Pointer[prim.Char8]
	varyRuleLen            prim.Usize
	initialAgeNs           prim.U64
	staleWhileRevalidateNs prim.U64
	surrogateKeysPtr       prim.Pointer[prim.Char8]
	surrogateKeysLen       prim.Usize
	length                 prim.U64
	userMetadataPtr        prim.Pointer[prim.U8]
	userMetadataLen        prim.Usize
}

// witx:
//
//	(typename $cache_write_options_mask
//	    (flags (@witx repr u32)
//	        $reserved
//	        $request_headers ;;; Only allowed for non-transactional `insert`
//	        $vary_rule
//	        $initial_age_ns
//	        $stale_while_revalidate_ns
//	        $surrogate_keys
//	        $length
//	        $user_metadata
//	        $sensitive_data
//	    )
//	)
type cacheWriteOptionsMask prim.U32

const (
	cacheWriteOptionsMaskReserved               cacheWriteOptionsMask = 1 << 0 // $reserved
	cacheWriteOptionsMaskRequestHeaders         cacheWriteOptionsMask = 1 << 1 // $request_headers
	cacheWriteOptionsMaskVaryRule               cacheWriteOptionsMask = 1 << 2 // $vary_rule
	cacheWriteOptionsMaskInitialAgeNs           cacheWriteOptionsMask = 1 << 3 // $initial_age_ns
	cacheWriteOptionsMaskStaleWhileRevalidateNs cacheWriteOptionsMask = 1 << 4 // $stale_while_revalidate_ns
	cacheWriteOptionsMaskSurrogateKeys          cacheWriteOptionsMask = 1 << 5 // $surrogate_keys
	cacheWriteOptionsMaskLength                 cacheWriteOptionsMask = 1 << 6 // $length
	cacheWriteOptionsMaskUserMetadata           cacheWriteOptionsMask = 1 << 7 // $user_metadata
	cacheWriteOptionsMaskSensitiveData          cacheWriteOptionsMask = 1 << 8 // $sensitive_data
)

// witx:
//
//	(typename $cache_get_body_options
//	    (record
//	        (field $from u64)
//	        (field $to u64)
//	    )
//	)
type cacheGetBodyOptions struct {
	from prim.U64
	to   prim.U64
}

// witx:
//
//	(typename $cache_get_body_options_mask
//	    (flags (@witx repr u32)
//	        $reserved
//	        $from
//	        $to
//	    )
//	)
type cacheGetBodyOptionsMask prim.U32

const (
	cacheGetBodyOptionsMaskReserved cacheGetBodyOptionsMask = 0b0000_0001 // $reserved
	cacheGetBodyOptionsMaskFrom     cacheGetBodyOptionsMask = 0b0000_0010 // $from
	cacheGetBodyOptionsMaskTo       cacheGetBodyOptionsMask = 0b0000_0100 // $to
)

// witx:
//
//	;;; The status of this lookup (and potential transaction)
//	(typename $cache_lookup_state
//	    (flags (@witx repr u32)
//	        $found ;; a cached object was found
//	        $usable ;; the cached object is valid to use (implies $found)
//	        $stale ;; the cached object is stale (but may or may not be valid to use)
//	        $must_insert_or_update ;; this client is requested to insert or revalidate an object
//	    )
//	)
type CacheLookupState prim.U32

const (
	CacheLookupStateFound              CacheLookupState = 0b0000_0001 // $found
	CacheLookupStateUsable             CacheLookupState = 0b0000_0010 // $usable
	CacheLookupStateStale              CacheLookupState = 0b0000_0100 // $stale
	CacheLookupStateMustInsertOrUpdate CacheLookupState = 0b0000_1000 // $must_insert_or_update
)

// witx:
//
//	(typename $purge_options_mask
//	    (flags (@witx repr u32)
//	        $soft_purge
//	        $ret_buf ;; all ret_buf fields must be populated
//	    )
//	)
type purgeOptionsMask prim.U32

const (
	purgeOptionsMaskSoftPurge purgeOptionsMask = 1 << 0 // $soft_purge
	purgeOptionsMaskRetBuf    purgeOptionsMask = 1 << 1 // $ret_buf
)

// witx:
//
//	(typename $purge_options
//	    (record
//	        ;; JSON purge response as in https://developer.fastly.com/reference/api/purging/#purge-tag
//	        (field $ret_buf_ptr (@witx pointer u8))
//	        (field $ret_buf_len (@witx usize))
//	        (field $ret_buf_nwritten_out (@witx pointer (@witx usize)))
//	    )
//	)
type purgeOptions struct {
	retBufPtr         prim.Pointer[prim.U8]
	retBufLen         prim.Usize
	retBufNwrittenOut prim.Pointer[prim.Usize]
}

// witx:
//
//   (typename $backend_config_options
//      (flags (@witx repr u32)
//       $reserved
//       $host_override
//       $connect_timeout
//       $first_byte_timeout
//       $between_bytes_timeout
//       $use_ssl
//       $ssl_min_version
//       $ssl_max_version
//       $cert_hostname
//       $ca_cert
//       $ciphers
//       $sni_hostname
//       $dont_pool))

type backendConfigOptionsMask prim.U32

const (
	backendConfigOptionsMaskReserved            backendConfigOptionsMask = 1 << 0  // $reserved
	backendConfigOptionsMaskHostOverride        backendConfigOptionsMask = 1 << 1  // $host_override
	backendConfigOptionsMaskConnectTimeout      backendConfigOptionsMask = 1 << 2  // $connect_timeout
	backendConfigOptionsMaskFirstByteTimeout    backendConfigOptionsMask = 1 << 3  // $first_byte_timeout
	backendConfigOptionsMaskBetweenBytesTimeout backendConfigOptionsMask = 1 << 4  // $between_bytes_timeout
	backendConfigOptionsMaskUseSSL              backendConfigOptionsMask = 1 << 5  // $use_ssl
	backendConfigOptionsMaskSSLMinVersion       backendConfigOptionsMask = 1 << 6  // $ssl_min_version
	backendConfigOptionsMaskSSLMaxVersion       backendConfigOptionsMask = 1 << 7  // $ssl_max_version
	backendConfigOptionsMaskCertHostname        backendConfigOptionsMask = 1 << 8  // $cert_hostname
	backendConfigOptionsMaskCACert              backendConfigOptionsMask = 1 << 9  // $ca_cert
	backendConfigOptionsMaskCiphers             backendConfigOptionsMask = 1 << 10 // $ciphers
	backendConfigOptionsMaskSNIHostname         backendConfigOptionsMask = 1 << 11 // $sni_hostame
	backendConfigOptionsMaskDontPool            backendConfigOptionsMask = 1 << 12 // $dont_pool
)

// witx:
//
//  (typename $dynamic_backend_config
//  	(record
//  	  (field $host_override (@witx pointer (@witx char8)))
//  	  (field $host_override_len u32)
//  	  (field $connect_timeout_ms u32)
//  	  (field $first_byte_timeout_ms u32)
//  	  (field $between_bytes_timeout_ms u32)
//  	  (field $ssl_min_version $tls_version)
//  	  (field $ssl_max_version $tls_version)
//  	  (field $cert_hostname (@witx pointer (@witx char8)))
//  	  (field $cert_hostname_len u32)
//  	  (field $ca_cert (@witx pointer (@witx char8)))
//  	  (field $ca_cert_len u32)
//  	  (field $ciphers (@witx pointer (@witx char8)))
//  	  (field $ciphers_len u32)
//  	  (field $sni_hostname (@witx pointer (@witx char8)))
//  	  (field $sni_hostname_len u32)
//  	  ))

type backendConfigOptions struct {
	hostOverridePtr     prim.Pointer[prim.Char8]
	hostOverrideLen     prim.U32
	connectTimeoutMs    prim.U32
	firstByteTimeout    prim.U32
	betweenBytesTimeout prim.U32
	sslMinVersion       TLSVersion
	sslMaxVersion       TLSVersion
	certHostnamePtr     prim.Pointer[prim.Char8]
	certHostnameLen     prim.U32
	caCertPtr           prim.Pointer[prim.Char8]
	caCertLen           prim.U32
	ciphersPtr          prim.Pointer[prim.Char8]
	ciphersLen          prim.U32
	sniHostnamePtr      prim.Pointer[prim.Char8]
	sniHostnameLen      prim.U32
}

// witx:
//
//	(typename $backend_health
//	    (enum (@witx tag u32)
//	        $unknown
//	        $healthy
//	        $unhealthy))
type BackendHealth prim.U32

const (
	BackendHealthUnknown   BackendHealth = 0
	BackendHealthHealthy   BackendHealth = 1
	BackendHealthUnhealthy BackendHealth = 2
)

// witx:
//
//	(typename $tls_version
//	    (enum (@witx tag u32)
//	      $tls_1
//	      $tls_1_1
//	      $tls_1_2
//	      $tls_1_3))
type TLSVersion prim.U32

const (
	TLSVersion1_0 TLSVersion = 0
	TLSVersion1_1 TLSVersion = 1
	TLSVersion1_2 TLSVersion = 2
	TLSVersion1_3 TLSVersion = 3
)

type BackendConfigOptions struct {
	mask backendConfigOptionsMask
	opts backendConfigOptions
}

func (b *BackendConfigOptions) HostOverride(host string) {
	b.mask |= backendConfigOptionsMaskHostOverride
	buf := prim.NewReadBufferFromString(host)
	b.opts.hostOverridePtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.hostOverrideLen = prim.U32(buf.Len())
}

func (b *BackendConfigOptions) ConnectTimeout(t time.Duration) {
	b.mask |= backendConfigOptionsMaskConnectTimeout
	b.opts.connectTimeoutMs = prim.U32(t.Milliseconds())
}

func (b *BackendConfigOptions) FirstByteTimeout(t time.Duration) {
	b.mask |= backendConfigOptionsMaskFirstByteTimeout
	b.opts.firstByteTimeout = prim.U32(t.Milliseconds())
}

func (b *BackendConfigOptions) BetweenBytesTimeout(t time.Duration) {
	b.mask |= backendConfigOptionsMaskBetweenBytesTimeout
	b.opts.betweenBytesTimeout = prim.U32(t.Milliseconds())
}

func (b *BackendConfigOptions) UseSSL(v bool) {
	if v {
		b.mask |= backendConfigOptionsMaskUseSSL
	} else {
		b.mask &^= backendConfigOptionsMaskUseSSL
	}
}

func (b *BackendConfigOptions) SSLMinVersion(v TLSVersion) {
	b.mask |= backendConfigOptionsMaskSSLMinVersion
	b.opts.sslMinVersion = v
}

func (b *BackendConfigOptions) SSLMaxVersion(v TLSVersion) {
	b.mask |= backendConfigOptionsMaskSSLMaxVersion
	b.opts.sslMaxVersion = v
}

func (b *BackendConfigOptions) CertHostname(certHostname string) {
	b.mask |= backendConfigOptionsMaskCertHostname
	buf := prim.NewReadBufferFromString(certHostname)
	b.opts.certHostnamePtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.certHostnameLen = prim.U32(buf.Len())
}

func (b *BackendConfigOptions) CACert(caCert string) {
	b.mask |= backendConfigOptionsMaskCACert
	buf := prim.NewReadBufferFromString(caCert)
	b.opts.caCertPtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.caCertLen = prim.U32(buf.Len())
}

func (b *BackendConfigOptions) Ciphers(ciphers string) {
	b.mask |= backendConfigOptionsMaskCiphers
	buf := prim.NewReadBufferFromString(ciphers)
	b.opts.ciphersPtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.ciphersLen = prim.U32(buf.Len())
}

func (b *BackendConfigOptions) SNIHostname(sniHostname string) {
	b.mask |= backendConfigOptionsMaskSNIHostname
	buf := prim.NewReadBufferFromString(sniHostname)
	b.opts.sniHostnamePtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.sniHostnameLen = prim.U32(buf.Len())
}

// witx:
//
//	(typename $send_error_detail_tag
//	    (enum (@witx tag u32)
//	        ;;; The $send_error_detail struct has not been populated.
//	        $uninitialized
//	        ;;; There was no send error.
//	        $ok
//	        ;;; The system encountered a timeout when trying to find an IP address for the backend
//	        ;;; hostname.
//	        $dns_timeout
//	        ;;; The system encountered a DNS error when trying to find an IP address for the backend
//	        ;;; hostname. The fields $dns_error_rcode and $dns_error_info_code may be set in the
//	        ;;; $send_error_detail.
//	        $dns_error
//	        ;;; The system cannot determine which backend to use, or the specified backend was invalid.
//	        $destination_not_found
//	        ;;; The system considers the backend to be unavailable; e.g., recent attempts to communicate
//	        ;;; with it may have failed, or a health check may indicate that it is down.
//	        $destination_unavailable
//	        ;;; The system cannot find a route to the next-hop IP address.
//	        $destination_ip_unroutable
//	        ;;; The system's connection to the backend was refused.
//	        $connection_refused
//	        ;;; The system's connection to the backend was closed before a complete response was
//	        ;;; received.
//	        $connection_terminated
//	        ;;; The system's attempt to open a connection to the backend timed out.
//	        $connection_timeout
//	        ;;; The system is configured to limit the number of connections it has to the backend, and
//	        ;;; that limit has been exceeded.
//	        $connection_limit_reached
//	        ;;; The system encountered an error when verifying the certificate presented by the backend.
//	        $tls_certificate_error
//	        ;;; The system encountered an error with the backend TLS configuration.
//	        $tls_configuration_error
//	        ;;; The system received an incomplete response to the request from the backend.
//	        $http_incomplete_response
//	        ;;; The system received a response to the request whose header section was considered too
//	        ;;; large.
//	        $http_response_header_section_too_large
//	        ;;; The system received a response to the request whose body was considered too large.
//	        $http_response_body_too_large
//	        ;;; The system reached a configured time limit waiting for the complete response.
//	        $http_response_timeout
//	        ;;; The system received a response to the request whose status code or reason phrase was
//	        ;;; invalid.
//	        $http_response_status_invalid
//	        ;;; The process of negotiating an upgrade of the HTTP version between the system and the
//	        ;;; backend failed.
//	        $http_upgrade_failed
//	        ;;; The system encountered an HTTP protocol error when communicating with the backend. This
//	        ;;; error will only be used when a more specific one is not defined.
//	        $http_protocol_error
//	        ;;; An invalid cache key was provided for the request.
//	        $http_request_cache_key_invalid
//	        ;;; An invalid URI was provided for the request.
//	        $http_request_uri_invalid
//	        ;;; The system encountered an unexpected internal error.
//	        $internal_error
//	        ;;; The system received a TLS alert from the backend. The field $tls_alert_id may be set in
//	        ;;; the $send_error_detail.
//	        $tls_alert_received
//	        ;;; The system encountered a TLS error when communicating with the backend, either during
//	        ;;; the handshake or afterwards.
//	        $tls_protocol_error
//	        ))
type sendErrorDetailTag prim.U32

const (
	sendErrorDetailTagUninitialized                     sendErrorDetailTag = 0
	sendErrorDetailTagOK                                                   = 1
	sendErrorDetailTagDNSTimeout                                           = 2
	sendErrorDetailTagDNSError                                             = 3
	sendErrorDetailTagDestinationNotFound                                  = 4
	sendErrorDetailTagDestinationUnavailable                               = 5
	sendErrorDetailTagDestinationIPUnroutable                              = 6
	sendErrorDetailTagConnectionRefused                                    = 7
	sendErrorDetailTagConnectionTerminated                                 = 8
	sendErrorDetailTagConnectionTimeout                                    = 9
	sendErrorDetailTagConnectionLimitReached                               = 10
	sendErrorDetailTagTLSCertificateError                                  = 11
	sendErrorDetailTagTLSConfigurationError                                = 12
	sendErrorDetailTagHTTPIncompleteResponse                               = 13
	sendErrorDetailTagHTTPResponseHeaderSectionTooLarge                    = 14
	sendErrorDetailTagHTTPResponseBodyTooLarge                             = 15
	sendErrorDetailTagHTTPResponseTimeout                                  = 16
	sendErrorDetailTagHTTPResponseStatusInvalid                            = 17
	sendErrorDetailTagHTTPUpgradeFailed                                    = 18
	sendErrorDetailTagHTTPProtocolError                                    = 19
	sendErrorDetailTagHTTPRequestCacheKeyInvalid                           = 20
	sendErrorDetailTagHTTPRequestURIInvalid                                = 21
	sendErrorDetailTagInternalError                                        = 22
	sendErrorDetailTagTLSAlertReceived                                     = 23
	sendErrorDetailTagTLSProtocolError                                     = 24
)

// witx:
//
//	;;; Mask representing which fields are understood by the guest, and which have been set by the host.
//	;;;
//	;;; When the guest calls hostcalls with a mask, it should set every bit in the mask that corresponds
//	;;; to a defined flag. This signals the host to write only to fields with a set bit, allowing
//	;;; forward compatibility for existing guest programs even after new fields are added to the struct.
//	(typename $send_error_detail_mask
//	    (flags (@witx repr u32)
//	       $reserved
//	       $dns_error_rcode
//	       $dns_error_info_code
//	       $tls_alert_id
//	       ))
type sendErrorDetailMask prim.U32

const (
	sendErrorDetailMaskReserved      = 1 << 0 // $reserved
	sendErrorDetailMaskDNSErrorRCode = 1 << 1 // $dns_error_rcode
	sendErrorDetailMaskDNSErrorInfo  = 1 << 2 // $dns_error_info_code
	sendErrorDetailMaskTLSAlertID    = 1 << 3 // $tls_alert_id
)

// witx:
//
//	(typename $send_error_detail
//	  (record
//	    (field $tag $send_error_detail_tag)
//	    (field $mask $send_error_detail_mask)
//	    (field $dns_error_rcode u16)
//	    (field $dns_error_info_code u16)
//	    (field $tls_alert_id u8)
//	    ))
type sendErrorDetail struct {
	tag              sendErrorDetailTag
	mask             sendErrorDetailMask
	dnsErrorRCode    prim.U16
	dnsErrorInfoCode prim.U16
	tlsAlertID       prim.U8
}

func newSendErrorDetail() sendErrorDetail {
	return sendErrorDetail{
		mask: sendErrorDetailMaskDNSErrorRCode | sendErrorDetailMaskDNSErrorInfo | sendErrorDetailMaskTLSAlertID,
	}
}

func (d sendErrorDetail) String() string {
	switch d.tag {
	case sendErrorDetailTagUninitialized:
		// Not enough information to convert to an error.  In this case,
		// the caller should use the FastlyStatus as the basis for the
		// error instead.
		return ""

	case sendErrorDetailTagOK:
		// No error
		return ""

	case sendErrorDetailTagDNSTimeout:
		return "DNS timeout"
	case sendErrorDetailTagDNSError:
		return fmt.Sprintf("DNS error (rcode=%d, info_code=%d)", d.dnsErrorRCode, d.dnsErrorInfoCode)
	case sendErrorDetailTagDestinationNotFound:
		return "destination not found"
	case sendErrorDetailTagDestinationUnavailable:
		return "destination unavailable"
	case sendErrorDetailTagDestinationIPUnroutable:
		return "destination IP unroutable"
	case sendErrorDetailTagConnectionRefused:
		return "connection refused"
	case sendErrorDetailTagConnectionTerminated:
		return "connection terminated"
	case sendErrorDetailTagConnectionTimeout:
		return "connection timeout"
	case sendErrorDetailTagConnectionLimitReached:
		return "connection limit reached"
	case sendErrorDetailTagTLSCertificateError:
		return "TLS certificate error"
	case sendErrorDetailTagTLSConfigurationError:
		return "TLS configuration error"
	case sendErrorDetailTagHTTPIncompleteResponse:
		return "incomplete HTTP response"
	case sendErrorDetailTagHTTPResponseHeaderSectionTooLarge:
		return "HTTP response header section too large"
	case sendErrorDetailTagHTTPResponseBodyTooLarge:
		return "HTTP response body too large"
	case sendErrorDetailTagHTTPResponseTimeout:
		return "HTTP response timeout"
	case sendErrorDetailTagHTTPResponseStatusInvalid:
		return "HTTP response status invalid"
	case sendErrorDetailTagHTTPUpgradeFailed:
		return "HTTP upgrade failed"
	case sendErrorDetailTagHTTPProtocolError:
		return "HTTP protocol error"
	case sendErrorDetailTagHTTPRequestCacheKeyInvalid:
		return "HTTP request cache key invalid"
	case sendErrorDetailTagHTTPRequestURIInvalid:
		return "HTTP request URI invalid"
	case sendErrorDetailTagInternalError:
		return "internal error"
	case sendErrorDetailTagTLSAlertReceived:
		return fmt.Sprintf("TLS alert received (%s)", tlsAlertString(d.tlsAlertID))
	case sendErrorDetailTagTLSProtocolError:
		return "TLS protocol error"
	default:
		return fmt.Sprintf("unknown error (%d)", d.tag)
	}
}

func tlsAlertString(id prim.U8) string {
	switch id {
	case 0:
		return "close notify"
	case 10:
		return "unexpected message"
	case 20:
		return "bad record MAC"
	case 21:
		return "decryption failed"
	case 22:
		return "record overflow"
	case 30:
		return "decompression failure"
	case 40:
		return "handshake failure"
	case 41:
		return "no certificate"
	case 42:
		return "bad certificate"
	case 43:
		return "unsupported certificate"
	case 44:
		return "certificate revoked"
	case 45:
		return "certificate expired"
	case 46:
		return "certificate unknown"
	case 47:
		return "illegal parameter"
	case 48:
		return "unknown certificate authority"
	case 49:
		return "access denied"
	case 50:
		return "error decoding message"
	case 51:
		return "error decrypting message"
	case 60:
		return "export restriction"
	case 70:
		return "protocol version not supported"
	case 71:
		return "insufficient security level"
	case 80:
		return "internal error"
	case 86:
		return "inappropriate fallback"
	case 90:
		return "user canceled"
	case 100:
		return "no renegotiation"
	case 109:
		return "missing extension"
	case 110:
		return "unsupported extension"
	case 111:
		return "certificate unobtainable"
	case 112:
		return "unrecognized name"
	case 113:
		return "bad certificate status response"
	case 114:
		return "bad certificate hash value"
	case 115:
		return "unknown PSK identity"
	case 116:
		return "certificate required"
	case 120:
		return "no application protocol"
	default:
		return strconv.Itoa(int(id))
	}
}
