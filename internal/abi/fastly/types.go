//lint:file-ignore U1000 Ignore all unused code
//revive:disable:exported

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"bytes"
	"fmt"
	"math"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

type handle uintptr

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
//          $httpinvalidstatus))

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
	default:
		return "unknown"
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

// FastlyError decorates error-class FastlyStatus values and implements the
// error interface.
//
// Note that TinyGo currently doesn't support errors.As. Callers can use the
// IsFastlyError helper instead.
type FastlyError struct {
	Status FastlyStatus
}

// Error implements the error interface.
func (e FastlyError) Error() string {
	return fmt.Sprintf("Fastly error: %s", e.Status.String())
}

func (e FastlyError) getStatus() FastlyStatus {
	return e.Status
}

// IsFastlyError detects and unwraps a FastlyError to its component parts.
func IsFastlyError(err error) (FastlyStatus, bool) {
	if e, ok := err.(interface{ getStatus() FastlyStatus }); ok {
		return e.getStatus(), true
	}
	return 0, false
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
type httpStatus uint16

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
//		  $none
//		  $pass
//		  $ttl
//		  $stale_while_revalidate
//		  $pci))
type cacheOverrideTag uint32

const (
	cacheOverrideTagNone                 cacheOverrideTag = 0b0000_0000 // $none
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
