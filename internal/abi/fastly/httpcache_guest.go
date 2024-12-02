//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package fastly

import (
	"unsafe"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

type HTTPCacheLookupOptions struct {
	mask httpCacheLookupOptionsMask
	opts httpCacheLookupOptions
}

func (o *HTTPCacheLookupOptions) SetOverrideKey(key []byte) {
	o.mask |= httpCacheLookupOptionsFlagOverrideKey
	buf := prim.NewReadBufferFromBytes(key)
	o.opts.overrideKeyPtr = prim.ToPointer(buf.Char8Pointer())
	o.opts.overrideKeyLen = buf.Len()
}

type HTTPCacheWriteOptions struct {
	mask httpCacheWriteOptionsMask
	opts httpCacheWriteOptions
}

func (o *HTTPCacheWriteOptions) SetMaxAgeNs(maxAge httpCacheDurationNs) {
	o.opts.maxAgeNs = maxAge
	// This field is required; there is no mask bit set.
}

func (o *HTTPCacheWriteOptions) MaxAgeNs() httpCacheDurationNs {
	return o.opts.maxAgeNs
}

func (o *HTTPCacheWriteOptions) SetVaryRule(rule string) {
	buf := prim.NewReadBufferFromString(rule)
	o.opts.varyRulePtr = prim.ToPointer(buf.Char8Pointer())
	o.opts.varyRuleLen = buf.Len()
	o.mask |= httpCacheWriteOptionsFlagVaryRule
}

func (o *HTTPCacheWriteOptions) VaryRule() (string, bool) {
	if o.mask&httpCacheWriteOptionsFlagVaryRule == 0 {
		return "", false
	}

	return unsafe.String((*byte)(o.opts.varyRulePtr.Ptr()), o.opts.varyRuleLen), true
}

func (o *HTTPCacheWriteOptions) SetInitialAgeNs(initialAge httpCacheDurationNs) {
	o.opts.initialAgeNs = initialAge
	o.mask |= httpCacheWriteOptionsFlagInitialAge
}

func (o *HTTPCacheWriteOptions) InitialAgeNs(initialAge httpCacheDurationNs) (httpCacheDurationNs, bool) {
	return o.opts.initialAgeNs, o.mask&httpCacheWriteOptionsFlagInitialAge == httpCacheWriteOptionsFlagInitialAge
}

func (o *HTTPCacheWriteOptions) SetStaleWhileRevalidateNs(staleWhileRevalidateNs httpCacheDurationNs) {
	o.opts.staleWhileRevalidateNs = staleWhileRevalidateNs
	o.mask |= httpCacheWriteOptionsFlagStaleWhileRevalidate
}

func (o *HTTPCacheWriteOptions) StaleWhileRevalidate(staleWhileRevalidateNs httpCacheDurationNs) (httpCacheDurationNs, bool) {
	return o.opts.staleWhileRevalidateNs, o.mask&httpCacheWriteOptionsFlagStaleWhileRevalidate == httpCacheWriteOptionsFlagStaleWhileRevalidate
}

func (o *HTTPCacheWriteOptions) SetSurrogateKeys(keys string) {
	buf := prim.NewReadBufferFromString(keys)
	o.opts.surrogateKeysPtr = prim.ToPointer(buf.Char8Pointer())
	o.opts.surrogateKeysLen = buf.Len()
	o.mask |= httpCacheWriteOptionsFlagSurrogateKeys
}
func (o *HTTPCacheWriteOptions) SetLength(length httpCacheObjectLength) {
	o.opts.length = length
	o.mask |= httpCacheWriteOptionsFlagLength
}

func (o *HTTPCacheWriteOptions) Length(length httpCacheObjectLength) (httpCacheObjectLength, bool) {
	return o.opts.length, o.mask&httpCacheWriteOptionsFlagLength == httpCacheWriteOptionsFlagLength
}

// (module $fastly_http_cache

// witx;
//
//	;;; Determine whether a request is cacheable per conservative RFC 9111 semantics.
//	;;;
//	;;; In particular, this function checks whether the request method is `GET` or `HEAD`, and
//	;;; considers requests with other methods uncacheable. Applications where it is safe to cache
//	;;; responses to other methods should consider using their own cacheability check instead of
//	;;; this function.
//	(@interface func (export "is_request_cacheable")
//	    (param $req_handle $request_handle)
//	    (result $err (expected $is_cacheable (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache is_request_cacheable
//go:noescape
func fastlyHTTPCacheIsRequestCacheable(
	h requestHandle,
	isCacheable prim.Pointer[httpIsCacheable],
) FastlyStatus

func HTTPCacheIsRequestCacheable(req *HTTPRequest) (bool, error) {

	var isCacheable httpIsCacheable

	if err := fastlyHTTPCacheIsRequestCacheable(
		req.h,
		prim.ToPointer(&isCacheable),
	).toError(); err != nil {
		return false, err
	}

	if isCacheable == 0 {
		return false, nil
	}

	return true, nil
}

// witx:
//
//	;;; Retrieves the default cache key for the request.
//	;;;
//	;;; The `$key_out` parameter must point to an array of size `key_out_len`.
//	;;;
//	;;; If the guest-provided output parameter is not long enough to contain the full key,
//	;;; the required size is written by the host to `nwritten_out` and the `$buflen`
//	;;; error is returned.
//	;;;
//	;;; At the moment, HTTP cache keys must always be 32 bytes.
//	(@interface func (export "get_suggested_cache_key")
//	    (param $req_handle $request_handle)
//	    (param $key_out_ptr (@witx pointer (@witx char8)))
//	    (param $key_out_len (@witx usize))
//	    (param $nwritten_out (@witx pointer (@witx usize)))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache get_suggested_cache_key
//go:noescape
func fastlyHTTPCacheGetSuggestedCacheKey(
	h requestHandle,
	keyPtr prim.Pointer[prim.U8], keyLen prim.Usize,
	nWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

func HTTPCacheGetSuggestedCacheKey(req *HTTPRequest) ([]byte, error) {
	n := 32 // Cache keys are 32 bytes, per doc comment above.
	for {
		buf := prim.NewWriteBuffer(n)
		status := fastlyHTTPCacheGetSuggestedCacheKey(
			req.h,
			prim.ToPointer(buf.U8Pointer()),
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

// witx:
//
//	;;; Perform a cache lookup based on the given request without participating in request
//	;;; collapsing.
//	;;;
//	;;; The request is not consumed.
//	(@interface func (export "lookup")
//	    (param $req_handle $request_handle)
//	    (param $options_mask $http_cache_lookup_options_mask)
//	    (param $options (@witx pointer $http_cache_lookup_options))
//	    (result $err (expected $http_cache_handle (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache lookup
//go:noescape
func fastlyHTTPCacheLookup(
	h requestHandle,
	mask httpCacheLookupOptionsMask,
	opts prim.Pointer[httpCacheLookupOptions],
	cacheHandle prim.Pointer[httpCacheHandle],
) FastlyStatus

func HTTPCacheLookup(req *HTTPRequest, opts *HTTPCacheLookupOptions) (httpCacheHandle, error) {
	var h httpCacheHandle

	if err := fastlyHTTPCacheLookup(
		req.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&h),
	).toError(); err != nil {
		return 0, err
	}

	return h, nil
}

// witx:
//
//    ;;; Perform a cache lookup based on the given request.
//    ;;;
//    ;;; This operation always participates in request collapsing and may return an obligation to
//    ;;; insert or update responses, and/or stale responses. To bypass request collapsing, use
//    ;;; `lookup` instead.
//    ;;;
//    ;;; The request is not consumed.
//    (@interface func (export "transaction_lookup")
//        (param $req_handle $request_handle)
//        (param $options_mask $http_cache_lookup_options_mask)
//        (param $options (@witx pointer $http_cache_lookup_options))
//        (result $err (expected $http_cache_handle (error $fastly_status)))
//    )
//
//go:wasmimport fastly_http_cache transaction_lookup
//go:noescape

func fastlyHTTPCacheTransactionLookup(
	h requestHandle,
	mask httpCacheLookupOptionsMask,
	opts prim.Pointer[httpCacheLookupOptions],
	cacheHandle prim.Pointer[httpCacheHandle],
) FastlyStatus

func HTTPCacheTransactionLookup(req *HTTPRequest, opts *HTTPCacheLookupOptions) (httpCacheHandle, error) {
	var h httpCacheHandle

	if err := fastlyHTTPCacheLookup(
		req.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&h),
	).toError(); err != nil {
		return 0, err
	}

	return h, nil
}

// witx:
//
//	;;; Insert a response into the cache with the given options, returning a streaming body handle
//	;;; that is ready for writing or appending.
//	;;;
//	;;; Can only be used if the cache handle state includes the `$must_insert_or_update` flag.
//	;;;
//	;;; The response is consumed.
//	(@interface func (export "transaction_insert")
//	    (param $handle $http_cache_handle)
//	    (param $resp_handle $response_handle)
//	    (param $options_mask $http_cache_write_options_mask)
//	    (param $options (@witx pointer $http_cache_write_options))
//	    (result $err (expected $body_handle (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_insert
//go:noescape
func fastlyHTTPCacheTransactionInsert(
	h httpCacheHandle,
	r responseHandle,
	mask httpCacheWriteOptionsMask,
	opts prim.Pointer[httpCacheWriteOptions],
	bodyHandle prim.Pointer[bodyHandle],
) FastlyStatus

func HTTPCacheTransactionInsert(h httpCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, error) {
	var body bodyHandle = invalidBodyHandle

	if err := fastlyHTTPCacheTransactionInsert(
		h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body),
	).toError(); err != nil {
		return nil, err
	}

	// TODO(dgryski): check for body == invalidBodyHandle

	return &HTTPBody{h: body}, nil
}

// witx:
//
//	;;; Insert a response into the cache with the given options, and return a fresh cache handle
//	;;; that can be used to retrieve and stream the response while it's being inserted.
//	;;;
//	;;; This helps avoid the "slow reader" problem on a teed stream, for example when a program wishes
//	;;; to store a backend request in the cache while simultaneously streaming to a client in an HTTP
//	;;; response.
//	;;;
//	;;; The response is consumed.
//	(@interface func (export "transaction_insert_and_stream_back")
//	    (param $handle $http_cache_handle)
//	    (param $resp_handle $response_handle)
//	    (param $options_mask $http_cache_write_options_mask)
//	    (param $options (@witx pointer $http_cache_write_options))
//	    (result $err (expected (tuple $body_handle $http_cache_handle) (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_insert_and_stream_back
//go:noescape
func fastlyHTTPCacheTransactionInsertAndStreamBack(
	h httpCacheHandle,
	r responseHandle,
	mask httpCacheWriteOptionsMask,
	opts prim.Pointer[httpCacheWriteOptions],
	bodyHandle prim.Pointer[bodyHandle],
	newh prim.Pointer[httpCacheHandle],
) FastlyStatus

func HTTPCacheTransactionInsertAndStreamback(h httpCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, httpCacheHandle, error) {
	var body bodyHandle = invalidBodyHandle
	var newh httpCacheHandle = invalidHTTPCacheHandle

	if err := fastlyHTTPCacheTransactionInsertAndStreamBack(
		h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body),
		prim.ToPointer(&newh),
	).toError(); err != nil {
		return nil, 0, err
	}

	// TODO(dgryski): check for body == invalidBodyHandle
	// TODO(dgryski): check for newh == invalidHTTPCacheHandle

	return &HTTPBody{h: body}, newh, nil
}

// witx:
//
//	;;; Update freshness lifetime, response headers, and caching settings without updating the
//	;;; response body.
//	;;;
//	;;; Can only be used in if the cache handle state includes both of the flags:
//	;;; - `$found`
//	;;; - `$must_insert_or_update`
//	;;;
//	;;; The response is consumed.
//	(@interface func (export "transaction_update")
//	    (param $handle $http_cache_handle)
//	    (param $resp_handle $response_handle)
//	    (param $options_mask $http_cache_write_options_mask)
//	    (param $options (@witx pointer $http_cache_write_options))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_update
//go:noescape
func fastlyHTTPCacheTransactionUpdate(
	h httpCacheHandle,
	r responseHandle,
	mask httpCacheWriteOptionsMask,
	opts prim.Pointer[httpCacheWriteOptions],
) FastlyStatus

func HTTPCacheTransactionUpdate(h httpCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) error {
	if err := fastlyHTTPCacheTransactionUpdate(
		h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	;;; Update freshness lifetime, response headers, and caching settings without updating the
//	;;; response body, and return a fresh cache handle that can be used to retrieve and stream the
//	;;; stored response.
//	;;;
//	;;; Can only be used in if the cache handle state includes both of the flags:
//	;;; - `$found`
//	;;; - `$must_insert_or_update`
//	;;;
//	;;; The response is consumed.
//	(@interface func (export "transaction_update_and_return_fresh")
//	    (param $handle $http_cache_handle)
//	    (param $resp_handle $response_handle)
//	    (param $options_mask $http_cache_write_options_mask)
//	    (param $options (@witx pointer $http_cache_write_options))
//	    (result $err (expected $http_cache_handle (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_update_and_return_fresh
//go:noescape
func fastlyHTTPCacheTransactionUpdateAndReturnFresh(
	h httpCacheHandle,
	r responseHandle,
	mask httpCacheWriteOptionsMask,
	opts prim.Pointer[httpCacheWriteOptions],
	newh prim.Pointer[httpCacheHandle],
) FastlyStatus

func HTTPCacheTransactionUpdateAndReturnFresh(h httpCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (httpCacheHandle, error) {
	var newh = invalidHTTPCacheHandle

	if err := fastlyHTTPCacheTransactionUpdateAndReturnFresh(
		h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&newh),
	).toError(); err != nil {
		return 0, err
	}

	// TODO(dgryski): check newh == invalidHTTPCacheHandle

	return newh, nil
}

// witx:
//
//	;;; Disable request collapsing and response caching for this cache entry.
//	;;;
//	;;; In Varnish terms, this function stores a hit-for-pass object.
//	;;;
//	;;; Only the max age and, optionally, the vary rule are read from the options mask and struct
//	;;; for this function.
//	(@interface func (export "transaction_record_not_cacheable")
//	    (param $handle $http_cache_handle)
//	    (param $options_mask $http_cache_write_options_mask)
//	    (param $options (@witx pointer $http_cache_write_options))
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_record_not_cacheable
//go:noescape
func fastlyHTTPCacheTransactionRecordNotCacheable(
	h httpCacheHandle,
	mask httpCacheWriteOptionsMask,
	opts prim.Pointer[httpCacheWriteOptions],
) FastlyStatus

func HTTPCacheTransactionRecordNotCacheable(h httpCacheHandle, opts *HTTPCacheWriteOptions) error {
	if err := fastlyHTTPCacheTransactionRecordNotCacheable(
		h,
		opts.mask,
		prim.ToPointer(&opts.opts),
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	;;; Abandon an obligation to provide a response to the cache.
//	;;;
//	;;; Useful if there is an error before streaming is possible, e.g. if a backend is unreachable.
//	;;;
//	;;; If there are other requests collapsed on this transaction, one of those other requests will
//	;;; be awoken and given the obligation to provide a response. Note that if subsequent requests
//	;;; are unlikely to yield cacheable responses, this may lead to undesired serialization of
//	;;; requests. Consider using `transaction_record_not_cacheable` to make lookups for this request
//	;;; bypass the cache.
//	(@interface func (export "transaction_abandon")
//	    (param $handle $http_cache_handle)
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_abandon
//go:noescape
func fastlyHTTPCacheTransactionAbandon(
	h httpCacheHandle,
) FastlyStatus

func HTTPCacheTransactionAbandon(h httpCacheHandle) error {
	if err := fastlyHTTPCacheTransactionAbandon(
		h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//	;;; Close an ongoing interaction with the cache.
//	;;;
//	;;; If the cache handle state includes `$must_insert_or_update` (and hence no insert or update
//	;;; has been performed), closing the handle cancels any request collapsing, potentially choosing
//	;;; a new waiter to perform the insertion/update.
//	(@interface func (export "close")
//	    (param $handle $http_cache_handle)
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache close
//go:noescape
func fastlyHTTPCacheTransactionClose(
	h httpCacheHandle,
) FastlyStatus

func HTTPCacheTransactionClose(h httpCacheHandle) error {
	if err := fastlyHTTPCacheTransactionClose(
		h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//    ;;; Prepare a suggested request to make to a backend to satisfy the looked-up request.
//    ;;;
//    ;;; If there is a stored, stale response, this suggested request may be for revalidation. If the
//    ;;; looked-up request is ranged, the suggested request will be unranged in order to try caching
//    ;;; the entire response.
//    (@interface func (export "get_suggested_backend_request")
//        (param $handle $http_cache_handle)
//        (result $err (expected $request_handle (error $fastly_status)))
//    )
//
//

//go:wasmimport fastly_http_cache get_suggested_backend_request
//go:noescape
func fastlyHTTPCacheGetSuggestedBackendRequest(
	h httpCacheHandle,
	req prim.Pointer[requestHandle],
) FastlyStatus

func HTTPCacheGetSuggestedBackendRequest(h httpCacheHandle) (*HTTPRequest, error) {
	var req requestHandle = invalidRequestHandle
	if err := fastlyHTTPCacheGetSuggestedBackendRequest(
		h,
		prim.ToPointer(&req),
	).toError(); err != nil {
		return nil, err
	}

	// TODO(dgryski): check req == invalidRequestHandle

	return &HTTPRequest{h: req}, nil
}

// witx:
//
//    ;;; Prepare a suggested set of cache write options for a given request and response pair.
//    ;;;
//    ;;; The ABI of this function includes several unusual types of input and output parameters.
//    ;;;
//    ;;; The bits set in the `options_mask` input parameter describe which cache options the guest is
//    ;;; requesting that the host provide.
//    ;;;
//    ;;; The `options` input parameter allows the guest to provide output parameters for
//    ;;; pointer/length options. When the corresponding bit is set in `options_mask`, the pointer and
//    ;;; length should be set in this record to be used by the host to provide the output.
//    ;;;
//    ;;; The `options_mask_out` output parameter is only used by the host to indicate the status of
//    ;;; pointer/length data in the `options_out` record. The flag for a given pointer/length
//    ;;; parameter is set by the host if the corresponding flag was set in `options_mask`, and the
//    ;;; value is present in the suggested options. If the host returns a status of `$buflen`, the
//    ;;; same set of flags will be set, but the length value of the corresponding fields in
//    ;;; `options_out` are set to the lengths that would be required to read the full value from the
//    ;;; host on a subsequent call.
//    ;;;
//    ;;; The `options_out` output parameter is where the host writes the suggested options that were
//    ;;; requested by the guest in `options_mask`. For pointer/length data, if there was enough room
//    ;;; to write the suggested option, the length field will contain the length of the data actually
//    ;;; written, while the pointer field will match the input pointer.
//    ;;;
//    ;;; The response is not consumed.
//    (@interface func (export "get_suggested_cache_options")
//        (param $handle $http_cache_handle)
//        (param $response $response_handle)
//        (param $options_mask $http_cache_write_options_mask)
//        (param $options (@witx pointer $http_cache_write_options))
//        (param $options_mask_out (@witx pointer $http_cache_write_options_mask))
//        (param $options_out (@witx pointer $http_cache_write_options))
//        (result $err (expected (error $fastly_status)))
//    )

//go:wasmimport fastly_http_cache get_suggested_cache_options
//go:noescape
func fastlyHTTPCacheGetSuggestedCacheOptions(
	h httpCacheHandle,
	r responseHandle,
	inMask httpCacheWriteOptionsMask,
	inOpts prim.Pointer[httpCacheWriteOptions],
	outMask prim.Pointer[httpCacheWriteOptionsMask],
	outOpts prim.Pointer[httpCacheWriteOptions],
) FastlyStatus

func HTTPCacheGetSuggestedCacheOptions(h httpCacheHandle, r *HTTPResponse, opts *HTTPCacheWriteOptions) error {
	var out HTTPCacheWriteOptions

	for {
		status := fastlyHTTPCacheGetSuggestedCacheOptions(
			h,
			r.h,
			opts.mask,
			prim.ToPointer(&opts.opts),
			prim.ToPointer(&out.mask),
			prim.ToPointer(&out.opts),
		)

		if status == FastlyStatusBufLen {
			// reallocate buffers in the output struct with their requested lengths

			if out.mask&httpCacheWriteOptionsFlagVaryRule == httpCacheWriteOptionsFlagVaryRule {
				n := int(out.opts.varyRuleLen)
				buf := prim.NewWriteBuffer(n)
				opts.opts.varyRulePtr = prim.ToPointer(buf.Char8Pointer())
				opts.opts.varyRuleLen = buf.NValue()
			}

			if out.mask&httpCacheWriteOptionsFlagSurrogateKeys == httpCacheWriteOptionsFlagSurrogateKeys {
				n := int(out.opts.surrogateKeysLen)
				buf := prim.NewWriteBuffer(n)
				opts.opts.surrogateKeysPtr = prim.ToPointer(buf.Char8Pointer())
				opts.opts.surrogateKeysLen = buf.NValue()
			}

			continue
		}

		if err := status.toError(); err != nil {
			return err
		}

		break
	}

	return nil
}

// witx:
//
//	;;; Adjust a response into the appropriate form for storage and provides a storage action recommendation.
//	;;;
//	;;; For example, if the looked-up request contains conditional headers, this function will
//	;;; interpret a `304 Not Modified` response for revalidation by updating headers.
//	;;;
//	;;; In addition to the updated response, this function returns the recommended storage action.
//	(@interface func (export "prepare_response_for_storage")
//	    (param $handle $http_cache_handle)
//	    (param $response $response_handle)
//	    (result $err (expected (tuple $http_storage_action $response_handle) (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache prepare_response_for_storage
//go:noescape
func fastlyHTTPCachePrepareResponseForStorage(
	h httpCacheHandle,
	r responseHandle,
	action prim.Pointer[httpStorageAction],
	newr prim.Pointer[responseHandle],
) FastlyStatus

func HTTPCachePrepareResponseForStorage(h httpCacheHandle, r *HTTPResponse) (httpStorageAction, *HTTPResponse, error) {
	var action httpStorageAction
	var newr responseHandle

	if err := fastlyHTTPCachePrepareResponseForStorage(
		h,
		r.h,
		prim.ToPointer(&action),
		prim.ToPointer(&newr),
	).toError(); err != nil {
		return 0, nil, err
	}

	return action, &HTTPResponse{h: newr}, nil
}
