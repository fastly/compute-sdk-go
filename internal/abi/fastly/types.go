//lint:file-ignore U1000 Ignore all unused code
//revive:disable:exported

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"io"
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

	// FastlyStatusAgain maps to $fastly_status $again.
	FastlyStatusAgain FastlyStatus = 14
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
	case FastlyStatusAgain:
		return "Again"
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

func (s FastlyStatus) toSendError(d SendErrorDetail) error {
	if s == FastlyStatusOK {
		return nil
	}

	if !d.valid() {
		return FastlyError{Status: s}
	}

	return FastlyError{Status: s, Detail: d}
}

func ignoreNoneError(err error) error {
	status, ok := IsFastlyError(err)
	if ok && status == FastlyStatusNone {
		return nil
	}
	return err
}

// FastlyError decorates error-class FastlyStatus values and implements the
// error interface.
//
// Note that TinyGo currently doesn't support errors.As. Callers can use the
// IsFastlyError helper instead.
type FastlyError struct {
	Status FastlyStatus

	// Detail contains an additional detailed error, if any.
	Detail error
}

// Error implements the error interface.
func (e FastlyError) Error() string {
	if e.Detail != nil && e.Detail.Error() != "" {
		return "Fastly error: " + e.Detail.Error()
	}

	return "Fastly error: " + e.Status.String()
}

// Unwrap returns the Detail error, allowing errors.Is() and errors.As() to
// traverse into the wrapped error.
func (e FastlyError) Unwrap() error {
	return e.Detail
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

const (
	ipBufLen  = 16  // known size for IP address buffers
	dnsBufLen = 256 // known size for "DNS" values, enough to hold the longest possible hostname or domain name

	DefaultSmallBufLen  = 128  // default size for "typically-small" values with variable sizes: HTTP methods, header names, tls protocol names, cipher suites
	DefaultMediumBufLen = 1024 // default size for values between small and large, with variable sizes
	DefaultLargeBufLen  = 8192 // default size for "typically-large" values with variable sizes; header values, URLs.
)

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

const (
	invalidRequestHandle = requestHandle(math.MaxUint32 - 1)
)

// witx:
//
//	(typename $response_handle (handle))
type responseHandle handle

const (
	invalidResponseHandle = responseHandle(math.MaxUint32 - 1)
)

// witx:
//
//	(typename $request_promise_handle (handle))
type requestPromiseHandle handle

const (
	invalidRequestPromiseHandle = requestPromiseHandle(math.MaxUint32 - 1)
)

// witx:
//
//	(typename $pending_request_handle (handle))
type pendingRequestHandle handle

const (
	invalidPendingRequestHandle = pendingRequestHandle(math.MaxUint32 - 1)
)

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
//	(typename $config_store_handle (handle))
type configstoreHandle handle

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

// newValuesBuffer constructs a Values iterator over the provided hostcall. The
// buffer is used to receive writes from the hostcall. If it is too small, it
// will allocate and resize to accommodate the actual size of the hostcall
// output.
func newValuesBuffer(f multiValueHostcall, buffer []byte) *Values {
	if buffer == nil {
		buffer = make([]byte, 0, DefaultMediumBufLen)
	}
	return &Values{
		f:      f,
		buffer: buffer,
	}
}

// newValues is a helper that allocates a buffer of capacity cap and
// provides it to newValuesBuffer.
func newValues(f multiValueHostcall, cap int) *Values {
	return newValuesBuffer(f, make([]byte, 0, cap))
}

func (v *Values) nextValue() {
	var result multiValueCursorResult
	for {
		buf := prim.NewWriteBufferFromBytes(v.buffer)
		status := v.f(
			buf.Char8Pointer(),
			buf.Cap(),
			v.cursor,
			&result,
			buf.NPointer(),
		)
		if status == FastlyStatusBufLen && buf.NValue() > 0 {
			v.buffer = make([]byte, 0, int(buf.NValue()))
			continue
		}
		v.err = status.toError()
		if v.err != nil {
			return
		}
		v.cursor = result.toCursor()
		v.finished = result.isFinished()
		v.pending = buf.AsBytes()
		return
	}
}

// Next prepares the next value for reading with the Bytes method. It returns
// true on success, or false if there are no more values, or an error occurred.
// Err should be called to distinguish between those two cases. Every call to
// Bytes, even the first one, must be preceded by a call to Next.
func (v *Values) Next() bool {
	// Check first for no further values.
	if v.err != nil {
		return false
	}
	if len(v.pending) == 0 {
		if v.finished {
			return false
		}

		// Get more data from the hostcall.
		//
		// The hostcall always writes complete values, up to the buffer's capacity,
		// or returns BufLen, never splitting a value over multiple calls. Said
		// another way: every value ends with a terminator. So we process via these
		// steps:
		//
		// 1. Call nextValue() to update v.buffer, which resizes in response to BufLen.
		// 2. Set v.pending to the rest of v.buffer, a "view" on the same [...]byte.
		// 3. Set v.value to the bytes before the next \0 in v.pending,
		// value by value, for each call to Next().
		// 4. All values in pending are consumed when len(v.pending) == 0.
		// 5. Repeat until nextValue() sets v.finished and len(v.pending) == 0.

		v.nextValue()
		if v.err != nil {
			return false
		}
	}

	// Pending buffer has something from nextValue(). Find the first terminator
	// and advance the sliding windows.
	var term bool
	v.value, v.pending, term = bytes.Cut(v.pending, []byte{0})
	if !term && !v.finished {
		v.err = fmt.Errorf("missing terminator")
	}
	return term
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
//	(typename $kv_store_handle (handle))
//	(typename $kv_store_lookup_handle (handle))
//	(typename $kv_store_insert_handle (handle))
//	(typename $kv_store_delete_handle (handle))
//	(typename $kv_store_list_handle (handle))
type (
	kvstoreHandle       handle
	kvstoreLookupHandle handle
	kvstoreInsertHandle handle
	kvstoreDeleteHandle handle
	kvstoreListHandle   handle
)

const (
	invalidKVStoreHandle  = kvstoreHandle(math.MaxUint32 - 1)
	invalidKVLookupHandle = kvstoreLookupHandle(math.MaxUint32 - 1)
	invalidKVInsertHandle = kvstoreInsertHandle(math.MaxUint32 - 1)
	invalidKVDeleteHandle = kvstoreDeleteHandle(math.MaxUint32 - 1)
	invalidKVListHandle   = kvstoreListHandle(math.MaxUint32 - 1)
)

type kvLookupConfigMask prim.U32

const (
	kvLookupConfigFlagReserved kvLookupConfigMask = 1 << 0
)

type kvLookupConfig struct {
	reserved prim.U32
}

type kvDeleteConfigMask prim.U32

const (
	kvDeleteConfigFlagReserved = 1 << 0
)

type kvDeleteConfig struct {
	reserved prim.U32
}

type kvInsertConfigMask prim.U32

// witx:
//
//	    (typename $kv_insert_config_options
//	    (flags (@witx repr u32)
//	       $reserved
//	       $background_fetch
//	       $if_generation_match
//	       $metadata
//	       $time_to_live_sec
//	       ))

const (
	kvInsertConfigFlagReserved          kvInsertConfigMask = 1 << 0
	kvInsertConfigFlagBackgroundFetch   kvInsertConfigMask = 1 << 1
	kvInsertConfigFlagReserved2         kvInsertConfigMask = 1 << 2
	kvInsertConfigFlagMetadata          kvInsertConfigMask = 1 << 3
	kvInsertConfigFlagTTLSec            kvInsertConfigMask = 1 << 4
	kvInsertConfigFlagIfGenerationMatch kvInsertConfigMask = 1 << 5
)

// witx:
//
//	(typename $kv_insert_mode
//	    (enum (@witx tag u32)
//	       $overwrite
//	       $add
//	       $append
//	       $prepend))

type KVInsertMode prim.U32

const (
	KVInsertModeOverwrite KVInsertMode = 0
	KVInsertModeAdd       KVInsertMode = 1
	KVInsertModeAppend    KVInsertMode = 2
	KVInsertModePrepend   KVInsertMode = 3
)

// witx:
//	(typename $kv_insert_config
//	  (record
//	    (field $mode $kv_insert_mode)
//	    (field $unused u32)
//	    (field $metadata (@witx pointer (@witx char8)))
//	    (field $metadata_len u32)
//	    (field $time_to_live_sec u32)
//	    (field $if_generation_match u32)
//	    ))

type kvInsertConfig struct {
	mode              KVInsertMode
	_                 prim.U32
	metadataPtr       prim.Pointer[prim.Char8]
	metadataLen       prim.U32
	ttlSec            prim.U32
	ifGenerationMatch prim.U64
}

type KVInsertConfig struct {
	mask kvInsertConfigMask
	opts kvInsertConfig
}

func (c *KVInsertConfig) Mode(mode KVInsertMode) {
	c.opts.mode = mode
}

func (c *KVInsertConfig) BackgroundFetch() {
	c.mask |= kvInsertConfigFlagBackgroundFetch
}

func (c *KVInsertConfig) Metadata(meta []byte) {
	c.mask |= kvInsertConfigFlagMetadata
	buf := prim.NewReadBufferFromBytes(meta)
	c.opts.metadataPtr = prim.ToPointer(buf.Char8Pointer())
	c.opts.metadataLen = prim.U32(buf.Len())
}

func (c *KVInsertConfig) TTLSec(seconds uint32) {
	c.mask |= kvInsertConfigFlagTTLSec
	c.opts.ttlSec = prim.U32(seconds)
}

func (c *KVInsertConfig) IfGenerationMatch(generation uint64) {
	c.mask |= kvInsertConfigFlagIfGenerationMatch
	c.opts.ifGenerationMatch = prim.U64(generation)
}

// witx:
// (typename $kv_list_config_options
//     (flags (@witx repr u32)
//       $reserved
//       $cursor
//       $limit
//       $prefix
//       ))

type kvListConfigMask prim.U32

const (
	kvListConfigFlagReserved kvListConfigMask = (1 << 0)
	kvListConfigFlagCursor   kvListConfigMask = (1 << 1)
	kvListConfigFlagLimit    kvListConfigMask = (1 << 2)
	kvListConfigFlagPrefix   kvListConfigMask = (1 << 3)
)

// witx:
//
// (typename $kv_list_mode
//    (enum (@witx tag u32)
//       $strong
//       $eventual))

type KVListMode prim.U32

const (
	KVListModeStrong   KVListMode = 0
	KVListModeEventual KVListMode = 1
)

// witx:
//
// (typename $kv_list_config
//   (record
//     (field $mode $kv_list_mode)
//     (field $cursor (@witx pointer (@witx char8)))
//     (field $cursor_len u32)
//     (field $limit u32)
//     (field $prefix (@witx pointer (@witx char8)))
//     (field $prefix_len u32)
//     ))

type kvListConfig struct {
	mode      KVListMode
	cursorPtr prim.Pointer[prim.Char8]
	cursorLen prim.U32
	limit     prim.U32
	prefixPtr prim.Pointer[prim.Char8]
	prefixLen prim.U32
}

type KVListConfig struct {
	mask kvListConfigMask
	opts kvListConfig
}

func (c *KVListConfig) Mode(mode KVListMode) {
	c.opts.mode = mode
}

func (c *KVListConfig) Cursor(cursor []byte) {
	c.mask |= kvListConfigFlagCursor
	buf := prim.NewReadBufferFromBytes(cursor)
	c.opts.cursorPtr = prim.ToPointer(buf.Char8Pointer())
	c.opts.cursorLen = prim.U32(buf.Len())
}

func (c *KVListConfig) Limit(limit uint32) {
	c.mask |= kvListConfigFlagLimit
	c.opts.limit = prim.U32(limit)
}

func (c *KVListConfig) Prefix(cursor []byte) {
	c.mask |= kvListConfigFlagPrefix
	buf := prim.NewReadBufferFromBytes(cursor)
	c.opts.prefixPtr = prim.ToPointer(buf.Char8Pointer())
	c.opts.prefixLen = prim.U32(buf.Len())
}

const kvstoreMetadataMaxBufLen = 2000

// witx:
//
//	(typename $kv_error
//	    (enum (@witx tag u32)
//	        ;;; The $kv_error has not been set.
//	        $uninitialized
//	        ;;; There was no error.
//	        $ok
//	        ;;; KV store cannot or will not process the request due to something that is perceived to be a client error
//	        ;;; This will map to the api's 400 codes
//	        $bad_request
//	        ;;; KV store cannot find the requested resource
//	        ;;; This will map to the api's 404 codes
//	        $not_found
//	        ;;; KV store cannot fulfill the request, as definied by the client's prerequisites (ie. if-generation-match)
//	        ;;; This will map to the api's 412 codes
//	        $precondition_failed
//	        ;;; The size limit for a KV store key was exceeded.
//	        ;;; This will map to the api's 413 codes
//	        $payload_too_large
//	        ;;; The system encountered an unexpected internal error.
//	        ;;; This will map to all remaining http error codes
//	        $internal_error
//	        ;;; Too many requests have been made to the KV store.
//	        ;;; This will map to the api's 429 codes
//	        $too_many_requests
//	        ))

type KVError prim.U32

const (
	KVErrorUninitialized      KVError = 0
	KVErrorOK                 KVError = 1
	KVErrorBadRequest         KVError = 2
	KVErrorNotFound           KVError = 3
	KVErrorPreconditionFailed KVError = 4
	KVErrorPayloadTooLarge    KVError = 5
	KVErrorInternalError      KVError = 6
	KVErrorTooManyRequests    KVError = 7
)

func (e KVError) Error() string {
	switch e {

	case KVErrorUninitialized:
		return "uninitialized"
	case KVErrorOK:
		return "OK"
	case KVErrorBadRequest:
		return "bad request"
	case KVErrorNotFound:
		return "not found"
	case KVErrorPreconditionFailed:
		return "precondition failed"
	case KVErrorPayloadTooLarge:
		return "payload too large"
	case KVErrorInternalError:
		return "internal error"
	case KVErrorTooManyRequests:
		return "too many requests"
	}

	return "unknown"
}

type KVLookupResult struct {
	Body       io.Reader
	Meta       []byte
	Generation uint64
}

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
//	        $service_id
//	        $always_use_requested_range
//	    )
//	)
type cacheLookupOptionsMask prim.U32

const (
	cacheLookupOptionsMaskReserved                cacheLookupOptionsMask = 0b0000_0001 // $reserved
	cacheLookupOptionsMaskRequestHeaders          cacheLookupOptionsMask = 0b0000_0010 // $request_headers
	cacheLookupOptionsMaskAlwaysUseRequestedRange cacheLookupOptionsMask = 0b0000_1000 // $always_use_requested_range
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
//       $dont_pool
//       $client_cert
//       $grpc
//       $keepalive
//       ))

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
	backendConfigOptionsMaskClientCert          backendConfigOptionsMask = 1 << 13 // $client_cert
	backendConfigOptionsMaskGRPC                backendConfigOptionsMask = 1 << 14 // $grpc
	backendConfigOptionsMaskKeepalive           backendConfigOptionsMask = 1 << 15 // $keepalive
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
//        (field $client_certificate (@witx pointer (@witx char8)))
//        (field $client_certificate_len u32)
//        (field $client_key $secret_handle)
//        (field $http_keepalive_time_ms $timeout_ms)
//        (field $tcp_keepalive_enable u32)
//        (field $tcp_keepalive_interval_secs $timeout_secs)
//        (field $tcp_keepalive_probes $probe_count)
//        (field $tcp_keepalive_time_secs $timeout_secs)
//  	  ))

type backendConfigOptions struct {
	hostOverridePtr          prim.Pointer[prim.Char8]
	hostOverrideLen          prim.U32
	connectTimeoutMs         prim.U32
	firstByteTimeout         prim.U32
	betweenBytesTimeout      prim.U32
	sslMinVersion            TLSVersion
	sslMaxVersion            TLSVersion
	certHostnamePtr          prim.Pointer[prim.Char8]
	certHostnameLen          prim.U32
	caCertPtr                prim.Pointer[prim.Char8]
	caCertLen                prim.U32
	ciphersPtr               prim.Pointer[prim.Char8]
	ciphersLen               prim.U32
	sniHostnamePtr           prim.Pointer[prim.Char8]
	sniHostnameLen           prim.U32
	clientCertPtr            prim.Pointer[prim.Char8]
	clientCertLen            prim.U32
	clientCertKey            secretHandle
	httpKeepaliveTimeMs      prim.U32
	tcpKeepaliveEnable       prim.U32
	tcpKeepaliveIntervalSecs prim.U32
	tcpKeepaliveProbes       prim.U32
	tcpKeepaliveTimeSecs     prim.U32
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

func (b *BackendConfigOptions) PoolConnections(poolingOn bool) {
	if poolingOn {
		b.mask &^= backendConfigOptionsMaskDontPool
	} else {
		b.mask |= backendConfigOptionsMaskDontPool
	}
}

func (b *BackendConfigOptions) ClientCert(certificate string, key *Secret) {
	b.mask |= backendConfigOptionsMaskClientCert
	buf := prim.NewReadBufferFromString(certificate)
	b.opts.clientCertPtr = prim.ToPointer(buf.Char8Pointer())
	b.opts.clientCertLen = prim.U32(buf.Len())
	b.opts.clientCertKey = key.Handle()
}

func (b *BackendConfigOptions) UseGRPC(v bool) {
	if v {
		b.mask |= backendConfigOptionsMaskGRPC
	} else {
		b.mask &^= backendConfigOptionsMaskGRPC
	}
}

func (b *BackendConfigOptions) HTTPKeepaliveTime(t time.Duration) {
	b.mask |= backendConfigOptionsMaskKeepalive
	b.opts.httpKeepaliveTimeMs = prim.U32(t.Milliseconds())
}

func (b *BackendConfigOptions) TCPKeepaliveEnable(v bool) {
	b.mask |= backendConfigOptionsMaskKeepalive
	if v {
		b.opts.tcpKeepaliveEnable = prim.U32(1)
	} else {
		b.opts.tcpKeepaliveEnable = prim.U32(0)
	}
}

func (b *BackendConfigOptions) TCPKeepaliveInterval(t time.Duration) {
	b.mask |= backendConfigOptionsMaskKeepalive
	b.opts.tcpKeepaliveEnable = prim.U32(1)
	b.opts.tcpKeepaliveIntervalSecs = prim.U32(t.Seconds())
}

func (b *BackendConfigOptions) TCPKeepaliveProbes(count uint32) {
	if count > 0 {
		b.mask |= backendConfigOptionsMaskKeepalive
		b.opts.tcpKeepaliveEnable = prim.U32(1)
		b.opts.tcpKeepaliveProbes = prim.U32(count)
	}
}

func (b *BackendConfigOptions) TCPKeepaliveTime(t time.Duration) {
	b.mask |= backendConfigOptionsMaskKeepalive
	b.opts.tcpKeepaliveEnable = prim.U32(1)
	b.opts.tcpKeepaliveTimeSecs = prim.U32(t.Seconds())
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
type SendErrorDetailTag prim.U32

const (
	SendErrorDetailTagUninitialized                     SendErrorDetailTag = 0
	SendErrorDetailTagOK                                SendErrorDetailTag = 1
	SendErrorDetailTagDNSTimeout                        SendErrorDetailTag = 2
	SendErrorDetailTagDNSError                          SendErrorDetailTag = 3
	SendErrorDetailTagDestinationNotFound               SendErrorDetailTag = 4
	SendErrorDetailTagDestinationUnavailable            SendErrorDetailTag = 5
	SendErrorDetailTagDestinationIPUnroutable           SendErrorDetailTag = 6
	SendErrorDetailTagConnectionRefused                 SendErrorDetailTag = 7
	SendErrorDetailTagConnectionTerminated              SendErrorDetailTag = 8
	SendErrorDetailTagConnectionTimeout                 SendErrorDetailTag = 9
	SendErrorDetailTagConnectionLimitReached            SendErrorDetailTag = 10
	SendErrorDetailTagTLSCertificateError               SendErrorDetailTag = 11
	SendErrorDetailTagTLSConfigurationError             SendErrorDetailTag = 12
	SendErrorDetailTagHTTPIncompleteResponse            SendErrorDetailTag = 13
	SendErrorDetailTagHTTPResponseHeaderSectionTooLarge SendErrorDetailTag = 14
	SendErrorDetailTagHTTPResponseBodyTooLarge          SendErrorDetailTag = 15
	SendErrorDetailTagHTTPResponseTimeout               SendErrorDetailTag = 16
	SendErrorDetailTagHTTPResponseStatusInvalid         SendErrorDetailTag = 17
	SendErrorDetailTagHTTPUpgradeFailed                 SendErrorDetailTag = 18
	SendErrorDetailTagHTTPProtocolError                 SendErrorDetailTag = 19
	SendErrorDetailTagHTTPRequestCacheKeyInvalid        SendErrorDetailTag = 20
	SendErrorDetailTagHTTPRequestURIInvalid             SendErrorDetailTag = 21
	SendErrorDetailTagInternalError                     SendErrorDetailTag = 22
	SendErrorDetailTagTLSAlertReceived                  SendErrorDetailTag = 23
	SendErrorDetailTagTLSProtocolError                  SendErrorDetailTag = 24
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

// SendErrorDetail contains detailed error information from backend send operations.
type SendErrorDetail struct {
	Tag              SendErrorDetailTag
	mask             sendErrorDetailMask
	dnsErrorRCode    prim.U16
	dnsErrorInfoCode prim.U16
	tlsAlertID       prim.U8
}

func newSendErrorDetail() SendErrorDetail {
	return SendErrorDetail{
		mask: sendErrorDetailMaskDNSErrorRCode | sendErrorDetailMaskDNSErrorInfo | sendErrorDetailMaskTLSAlertID,
	}
}

// Cause returns the specific cause of the backend request failure.
func (d SendErrorDetail) Cause() SendErrorDetailTag {
	return d.Tag
}

func (d SendErrorDetail) valid() bool {
	switch d.Tag {
	case SendErrorDetailTagUninitialized:
		// Not enough information to convert to an error.  In this case,
		// the caller should use the FastlyStatus as the basis for the
		// error instead.
		return false

	case SendErrorDetailTagOK:
		// No error
		return false
	}

	return true
}

// DNSErrorRCode returns the DNS response code for DNS errors.
// DNS response codes are defined [by IANA].
// Common values include:
//   - 0: No error
//   - 1: Format error
//   - 2: Server failure
//   - 3: Non-existent domain
//   - 4: Not implemented
//   - 5: Refused
//
// [by IANA]: https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml#dns-parameters-6
func (d SendErrorDetail) DNSErrorRCode() uint16 {
	return uint16(d.dnsErrorRCode)
}

// DNSErrorInfoCode returns additional DNS error information for DNS errors.
// DNS info codes are defined [by IANA].
//
// [by IANA]: https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml#extended-dns-error-codes
func (d SendErrorDetail) DNSErrorInfoCode() uint16 {
	return uint16(d.dnsErrorInfoCode)
}

// TLSAlertID returns the TLS alert identifier for TLS errors.
// TLS alert IDs are defined [by IANA].
// Use TLSAlertDescription() for a human-readable description.
//
// [by IANA]: https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-6
func (d SendErrorDetail) TLSAlertID() uint8 {
	return uint8(d.tlsAlertID)
}

// TLSAlertDescription returns a human-readable description of the TLS alert.
func (d SendErrorDetail) TLSAlertDescription() string {
	return tlsAlertString(d.tlsAlertID)
}

func (d SendErrorDetail) Error() string {
	return "send error: " + d.String()
}

func (d SendErrorDetail) String() string {
	switch d.Tag {
	case SendErrorDetailTagDNSTimeout:
		return "DNS timeout"
	case SendErrorDetailTagDNSError:
		return fmt.Sprintf("DNS error (rcode=%d, info_code=%d)", d.dnsErrorRCode, d.dnsErrorInfoCode)
	case SendErrorDetailTagDestinationNotFound:
		return "destination not found"
	case SendErrorDetailTagDestinationUnavailable:
		return "destination unavailable"
	case SendErrorDetailTagDestinationIPUnroutable:
		return "destination IP unroutable"
	case SendErrorDetailTagConnectionRefused:
		return "connection refused"
	case SendErrorDetailTagConnectionTerminated:
		return "connection terminated"
	case SendErrorDetailTagConnectionTimeout:
		return "connection timeout"
	case SendErrorDetailTagConnectionLimitReached:
		return "connection limit reached"
	case SendErrorDetailTagTLSCertificateError:
		return "TLS certificate error"
	case SendErrorDetailTagTLSConfigurationError:
		return "TLS configuration error"
	case SendErrorDetailTagHTTPIncompleteResponse:
		return "incomplete HTTP response"
	case SendErrorDetailTagHTTPResponseHeaderSectionTooLarge:
		return "HTTP response header section too large"
	case SendErrorDetailTagHTTPResponseBodyTooLarge:
		return "HTTP response body too large"
	case SendErrorDetailTagHTTPResponseTimeout:
		return "HTTP response timeout"
	case SendErrorDetailTagHTTPResponseStatusInvalid:
		return "HTTP response status invalid"
	case SendErrorDetailTagHTTPUpgradeFailed:
		return "HTTP upgrade failed"
	case SendErrorDetailTagHTTPProtocolError:
		return "HTTP protocol error"
	case SendErrorDetailTagHTTPRequestCacheKeyInvalid:
		return "HTTP request cache key invalid"
	case SendErrorDetailTagHTTPRequestURIInvalid:
		return "HTTP request URI invalid"
	case SendErrorDetailTagInternalError:
		return "internal error"
	case SendErrorDetailTagTLSAlertReceived:
		return fmt.Sprintf("TLS alert received (%s)", tlsAlertString(d.tlsAlertID))
	case SendErrorDetailTagTLSProtocolError:
		return "TLS protocol error"

	case SendErrorDetailTagUninitialized:
		panic("should not be reached: SendErrorDetailTagUninitialized")
	case SendErrorDetailTagOK:
		panic("should not be reached: SendErrorDetailTagOK")

	default:
		return fmt.Sprintf("unknown error (%d)", d.Tag)
	}
}

// Source: https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-6
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

type RateWindow struct {
	value prim.U32
}

func (r RateWindow) String() string {
	return strconv.FormatUint(uint64(r.value), 10)
}

var (
	RateWindow1s  = RateWindow{value: 1}
	RateWindow10s = RateWindow{value: 10}
	RateWindow60s = RateWindow{value: 60}
)

type CounterDuration struct {
	value prim.U32
}

func (c CounterDuration) String() string {
	return strconv.FormatUint(uint64(c.value), 10)
}

var (
	CounterDuration10s = CounterDuration{value: 10}
	CounterDuration20s = CounterDuration{value: 20}
	CounterDuration30s = CounterDuration{value: 30}
	CounterDuration40s = CounterDuration{value: 40}
	CounterDuration50s = CounterDuration{value: 50}
	CounterDuration60s = CounterDuration{value: 60}
)

// witx:
//
//	;;; A handle to an ACL.
//	(typename $acl_handle (handle))
type aclHandle handle

type ACLError prim.U32

// witx:
//
//	(enum (@witx tag u32)
//	    ;;; The $acl_error has not been initialized.
//	    $uninitialized
//	    ;;; There was no error.
//	    $ok
//	    ;;; This will map to the api's 204 code.
//	    ;;; It indicates that the request succeeded, yet returned nothing.
//	    $no_content
//	    ;;; This will map to the api's 429 code.
//	    ;;; Too many requests have been made.
//	    $too_many_requests
//	   ))
const (
	ACLErrorUninitialized   ACLError = 0
	ACLErrorOK              ACLError = 1
	ACLErrorNoContent       ACLError = 2
	ACLErrorTooManyRequests ACLError = 3
)

func (e ACLError) Error() string {
	switch e {
	case ACLErrorUninitialized:
		return "uninitialized"
	case ACLErrorOK:
		return "ok"
	case ACLErrorNoContent:
		return "no content"
	case ACLErrorTooManyRequests:
		return "too many requests"
	}

	return "unknown"
}

// http-cache.witx

type httpCacheHandle handle

const invalidHTTPCacheHandle = httpCacheHandle(math.MaxUint32 - 1)

type httpIsCacheable prim.U32

type httpIsSensitive prim.U32

type HTTPCacheStorageAction prim.U32

type httpCacheHitCount prim.U64

const (
	// Insert the response into cache (`transaction_insert*`).
	HTTPCacheStorageActionInsert HTTPCacheStorageAction = 0

	// Update the stale response in cache (`transaction_update*`).
	HTTPCacheStorageActionUpdate HTTPCacheStorageAction = 1

	// Do not store this response.
	HTTPCacheStorageActionDoNotStore HTTPCacheStorageAction = 2

	// Do not store this response, and furthermore record its non-cacheability for other pending
	// requests (`transaction_record_not_cacheable`).
	HTTPCacheStorageActionRecordUncacheable HTTPCacheStorageAction = 3
)

type httpCacheLookupOptions struct {
	overrideKeyPtr prim.Pointer[prim.Char8]
	overrideKeyLen prim.Usize
}

type httpCacheLookupOptionsMask prim.U32

const (
	httpCacheLookupOptionsFlagReserved    httpCacheLookupOptionsMask = 1 << 0
	httpCacheLookupOptionsFlagOverrideKey httpCacheLookupOptionsMask = 1 << 1
)

type (
	httpCacheDurationNs   prim.U64
	httpCacheObjectLength prim.U64
)

type httpCacheWriteOptions struct {
	// The maximum age of the response before it is considered stale, in nanoseconds.
	//
	// This field is required; there is no flag for it in `http_cache_write_options_mask`.
	maxAgeNs httpCacheDurationNs

	// A list of header names to use when calculating variants for this response.
	//
	// The format is a string containing header names separated by spaces.
	varyRulePtr prim.Pointer[prim.Char8]
	varyRuleLen prim.Usize

	// The initial age of the response in nanoseconds.
	//
	// If this field is not set, the default value is zero.
	//
	// This age is used to determine the freshness lifetime of the response as well as to
	// prioritize which variant to return if a subsequent lookup matches more than one vary rule
	initialAgeNs httpCacheDurationNs

	// The maximum duration after `max_age` during which the response may be delivered stale
	// while being revalidated, in nanoseconds.
	//
	// If this field is not set, the default value is zero.
	staleWhileRevalidateNs httpCacheDurationNs

	// A list of surrogate keys that may be used to purge this response.
	//
	// The format is a string containing [valid surrogate
	// keys](https://www.fastly.com/documentation/reference/http/http-headers/Surrogate-Key/)
	// separated by spaces.
	//
	// If this field is not set, no surrogate keys will be associated with the response. This
	// means that the response cannot be purged except via a purge-all operation.
	surrogateKeysPtr prim.Pointer[prim.Char8]
	surrogateKeysLen prim.Usize

	// The length of the response body.
	//
	// If this field is not set, the length of the body is treated as unknown.
	//
	// When possible, this field should be set so that other clients waiting to retrieve the
	// body have enough information to synthesize a `content-length` even before the complete
	// body is inserted to the cache.
	length httpCacheObjectLength
}

type httpCacheWriteOptionsMask prim.U32

const (
	httpCacheWriteOptionsFlagReserved             httpCacheWriteOptionsMask = 1 << 0
	httpCacheWriteOptionsFlagVaryRule             httpCacheWriteOptionsMask = 1 << 1
	httpCacheWriteOptionsFlagInitialAge           httpCacheWriteOptionsMask = 1 << 2
	httpCacheWriteOptionsFlagStaleWhileRevalidate httpCacheWriteOptionsMask = 1 << 3
	httpCacheWriteOptionsFlagSurrogateKeys        httpCacheWriteOptionsMask = 1 << 4
	httpCacheWriteOptionsFlagLength               httpCacheWriteOptionsMask = 1 << 5
	httpCacheWriteOptionsFlagSensitiveData        httpCacheWriteOptionsMask = 1 << 6
)

// shielding.witx

type shieldingBackendOptionsMask prim.U32

const (
	shieldingBackendOptionsFlagReserved    shieldingBackendOptionsMask = 1 << 0
	shieldingBackendOptionsFlagUseCacheKey shieldingBackendOptionsMask = 1 << 1
)

type shieldingBackendOptions struct {
	// A list of surrogate keys that may be used to purge this response.
	//
	// The format is a string containing [valid surrogate
	// keys](https://www.fastly.com/documentation/reference/http/http-headers/Surrogate-Key/)
	// separated by spaces.
	//
	// If this field is not set, no surrogate keys will be associated with the response. This
	// means that the response cannot be purged except via a purge-all operation.
	cacheKeyPtr prim.Pointer[prim.Char8]
	cacheKeyLen prim.Usize
}

type ShieldingBackendOptions struct {
	mask shieldingBackendOptionsMask
	opts shieldingBackendOptions
}

func (s *ShieldingBackendOptions) CacheKey(key string) {
	s.mask |= shieldingBackendOptionsFlagUseCacheKey
	buf := prim.NewReadBufferFromString(key)
	s.opts.cacheKeyPtr = prim.ToPointer(buf.Char8Pointer())
	s.opts.cacheKeyLen = buf.Len()
}

type ShieldInfo struct {
	me        bool
	target    string
	sslTarget string
}

func (s *ShieldInfo) RunningOn() bool { return s.me }

func (s *ShieldInfo) Target() string { return s.target }

func (s *ShieldInfo) SSLTarget() string { return s.sslTarget }

// witx:
//

type NextRequestOptions struct {
	mask nextRequestOptionsMask
	opts nextRequestOptions
}

// witx:
//
// (typename $next_request_options_mask
//     (flags (@witx repr u32)
//         $reserved
//         $timeout
//     ))

type nextRequestOptionsMask prim.U32

const (
	nextRequestOptionsMaskReserved nextRequestOptionsMask = 0b0000_0001 // $reserved
	nextRequestOptionsMaskTimeout  nextRequestOptionsMask = 0b0000_0010 // $timeout
)

// witx:
//
// (typename $next_request_options
//
//	(record
//	    ;; A maximum amount of time to wait for a downstream request to appear, in milliseconds.
//	    (field $timeout_ms u64)
//	))
type nextRequestOptions struct {
	timeoutMs prim.U64
}

func (n *NextRequestOptions) Timeout(t time.Duration) {
	n.mask |= nextRequestOptionsMaskTimeout
	n.opts.timeoutMs = prim.U64(t.Milliseconds())
}

// witx:
//
// (typename $image_optimizer_transform_config_options
//     (flags (@witx repr u32)
//         $reserved
//         $sdk_claims_opts
//         ))

type imageOptimizerTransformConfigOptionsMask uint32

const (
	imageOptimizerTransformConfigOptionsReserved      imageOptimizerTransformConfigOptionsMask = 1 << 0
	imageOptimizerTransformConfigOptionsSDKClaimsOpts imageOptimizerTransformConfigOptionsMask = 1 << 1
)

// witx:
//
// (typename $image_optimizer_transform_config
//   (record
//     ;; sdk_claims_opts contains any Image Optimizer API parameters that were set
//     ;; as well as the Image Optimizer region the request is meant for.
//     (field $sdk_claims_opts (@witx pointer (@witx char8)))
//     (field $sdk_claims_opts_len u32)
//     ))

type imageOptimizerTransformConfig struct {
	sdkClaimsOptsPtr prim.Pointer[prim.Char8]
	sdkClaimsOptsLen prim.U32
}

// witx:
//
// (typename $image_optimizer_error_tag
//     (enum (@witx tag u32)
//         $uninitialized
//         $ok
//         $error
//         $warning
//     )
// )

type ImageOptoError prim.U32

const (
	ImageOptoErrorUninitialized ImageOptoError = 0
	ImageOptoErrorOK            ImageOptoError = 1
	ImageOptoErrorError         ImageOptoError = 2
	ImageOptoErrorWarning       ImageOptoError = 3
)

// witx:
//
// (typename $image_optimizer_error_detail
//
//	(record
//	    (field $tag $image_optimizer_error_tag)
//	    (field $message (@witx pointer (@witx char8)))
//	    (field $message_len u32)
//	)
//
// )
type imageOptimizerErrorDetail struct {
	tag         ImageOptoError
	message     prim.Pointer[prim.Char8]
	message_len prim.U32
}

func (ioErr *imageOptimizerErrorDetail) Error() string {
	errStr := prim.NewWstringFromChar8(ioErr.message, ioErr.message_len).String()

	if ioErr.tag == ImageOptoErrorError {
		return "image opto: " + errStr
	}

	if ioErr.tag == ImageOptoErrorWarning {
		return "image opto warning: " + errStr
	}

	return "image opto: unknown error: " + errStr
}
