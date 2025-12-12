//go:build wasip1 && !nofastlyhostcalls

package fastly

import (
	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

const TRACE = true

type HTTPCacheLookupOptions struct {
	mask httpCacheLookupOptionsMask
	opts httpCacheLookupOptions
}

func (o *HTTPCacheLookupOptions) OverrideKey(key string) {
	k := []byte(key)
	buf := prim.NewReadBufferFromBytes(k)
	o.opts.overrideKeyPtr = prim.ToPointer(buf.Char8Pointer())
	o.opts.overrideKeyLen = buf.Len()
	o.mask |= httpCacheLookupOptionsFlagOverrideKey
}

type HTTPCacheWriteOptions struct {
	mask httpCacheWriteOptionsMask
	opts httpCacheWriteOptions

	vary      *prim.WriteBuffer
	surrogate *prim.WriteBuffer
}

func (o *HTTPCacheWriteOptions) SetMaxAgeNs(maxAge uint64) {
	o.opts.maxAgeNs = httpCacheDurationNs(maxAge)
	// This field is required; there is no mask bit set.
}

func (o *HTTPCacheWriteOptions) MaxAgeNs() uint64 {
	return uint64(o.opts.maxAgeNs)
}

func (o *HTTPCacheWriteOptions) SetVaryRule(rule string) {
	b := []byte(rule)
	o.vary = prim.NewWriteBufferFromBytes(b)
	o.opts.varyRulePtr = prim.ToPointer(o.vary.Char8Pointer())
	o.opts.varyRuleLen = o.vary.Len()
	o.mask |= httpCacheWriteOptionsFlagVaryRule
}

func (o *HTTPCacheWriteOptions) VaryRule() (string, bool) {
	if o.mask&httpCacheWriteOptionsFlagVaryRule == 0 {
		return "", false
	}

	p := o.vary.NPointer()
	*p = o.opts.varyRuleLen

	return o.vary.ToString(), true
}

func (o *HTTPCacheWriteOptions) SetInitialAgeNs(initialAge uint64) {
	o.opts.initialAgeNs = httpCacheDurationNs(initialAge)
	o.mask |= httpCacheWriteOptionsFlagInitialAge
}

func (o *HTTPCacheWriteOptions) InitialAgeNs() (uint64, bool) {
	return uint64(o.opts.initialAgeNs), o.mask&httpCacheWriteOptionsFlagInitialAge == httpCacheWriteOptionsFlagInitialAge
}

func (o *HTTPCacheWriteOptions) SetStaleWhileRevalidateNs(staleWhileRevalidateNs uint64) {
	o.opts.staleWhileRevalidateNs = httpCacheDurationNs(staleWhileRevalidateNs)
	o.mask |= httpCacheWriteOptionsFlagStaleWhileRevalidate
}

func (o *HTTPCacheWriteOptions) StaleWhileRevalidateNs() (uint64, bool) {
	return uint64(o.opts.staleWhileRevalidateNs), o.mask&httpCacheWriteOptionsFlagStaleWhileRevalidate == httpCacheWriteOptionsFlagStaleWhileRevalidate
}

func (o *HTTPCacheWriteOptions) SetSurrogateKeys(keys string) {
	b := []byte(keys)
	o.surrogate = prim.NewWriteBufferFromBytes(b)
	o.opts.surrogateKeysPtr = prim.ToPointer(o.surrogate.Char8Pointer())
	o.opts.surrogateKeysLen = o.surrogate.Len()
	o.mask |= httpCacheWriteOptionsFlagSurrogateKeys
}

func (o *HTTPCacheWriteOptions) SurrogateKeys() (string, bool) {
	if o.mask&httpCacheWriteOptionsFlagSurrogateKeys == 0 {
		return "", false
	}

	p := o.surrogate.NPointer()
	*p = o.opts.surrogateKeysLen

	return o.surrogate.ToString(), true
}

func (o *HTTPCacheWriteOptions) SetLength(length uint64) {
	o.opts.length = httpCacheObjectLength(length)
	o.mask |= httpCacheWriteOptionsFlagLength
}

func (o *HTTPCacheWriteOptions) Length() (uint64, bool) {
	return uint64(o.opts.length), o.mask&httpCacheWriteOptionsFlagLength == httpCacheWriteOptionsFlagLength
}

func (o *HTTPCacheWriteOptions) SetSensitiveData(b bool) {
	if b {
		o.mask |= httpCacheWriteOptionsFlagSensitiveData
	} else {
		o.mask &^= httpCacheWriteOptionsFlagSensitiveData
	}
}

func (o *HTTPCacheWriteOptions) SensitiveData() bool {
	return o.mask&httpCacheWriteOptionsFlagLength == httpCacheWriteOptionsFlagLength
}

func (o *HTTPCacheWriteOptions) FillConfigMask() {
	o.mask = 0 |
		httpCacheWriteOptionsFlagReserved |
		httpCacheWriteOptionsFlagVaryRule |
		httpCacheWriteOptionsFlagInitialAge |
		httpCacheWriteOptionsFlagStaleWhileRevalidate |
		httpCacheWriteOptionsFlagSurrogateKeys |
		httpCacheWriteOptionsFlagLength |
		httpCacheWriteOptionsFlagSensitiveData
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
	// Cache keys are 32 bytes, per doc comment above. This is future-proofing.
	value, err := withAdaptiveBuffer(32, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPCacheGetSuggestedCacheKey(
			req.h,
			prim.ToPointer(buf.U8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}
	return value.AsBytes(), nil
}

type HTTPCacheHandle struct {
	h httpCacheHandle
}

// witx:
//
//	;;; Perform a cache lookup based on the given request.
//	;;;
//	;;; This operation always participates in request collapsing and may return an obligation to
//	;;; insert or update responses, and/or stale responses. To bypass request collapsing, use
//	;;; `lookup` instead.
//	;;;
//	;;; The request is not consumed.
//	(@interface func (export "transaction_lookup")
//	    (param $req_handle $request_handle)
//	    (param $options_mask $http_cache_lookup_options_mask)
//	    (param $options (@witx pointer $http_cache_lookup_options))
//	    (result $err (expected $http_cache_handle (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_cache transaction_lookup
//go:noescape
func fastlyHTTPCacheTransactionLookup(
	h requestHandle,
	mask httpCacheLookupOptionsMask,
	opts prim.Pointer[httpCacheLookupOptions],
	cacheHandle prim.Pointer[httpCacheHandle],
) FastlyStatus

func HTTPCacheTransactionLookup(req *HTTPRequest, opts *HTTPCacheLookupOptions) (*HTTPCacheHandle, error) {
	var h httpCacheHandle = invalidHTTPCacheHandle

	if err := fastlyHTTPCacheTransactionLookup(
		req.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&h),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPCacheHandle{h: h}, nil
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

func HTTPCacheTransactionInsert(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, error) {
	var body bodyHandle = invalidBodyHandle

	if err := fastlyHTTPCacheTransactionInsert(
		h.h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPBody{h: body, closable: true}, nil
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

func HTTPCacheTransactionInsertAndStreamback(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, *HTTPCacheHandle, error) {
	var body bodyHandle = invalidBodyHandle
	var newh httpCacheHandle = invalidHTTPCacheHandle

	if err := fastlyHTTPCacheTransactionInsertAndStreamBack(
		h.h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&body),
		prim.ToPointer(&newh),
	).toError(); err != nil {
		return nil, nil, err
	}

	return &HTTPBody{h: body, closable: true}, &HTTPCacheHandle{h: newh}, nil
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

func HTTPCacheTransactionUpdate(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) error {
	if err := fastlyHTTPCacheTransactionUpdate(
		h.h,
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

func HTTPCacheTransactionUpdateAndReturnFresh(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPCacheHandle, error) {
	var newh = invalidHTTPCacheHandle

	if err := fastlyHTTPCacheTransactionUpdateAndReturnFresh(
		h.h,
		resp.h,
		opts.mask,
		prim.ToPointer(&opts.opts),
		prim.ToPointer(&newh),
	).toError(); err != nil {
		return nil, err
	}

	return &HTTPCacheHandle{h: newh}, nil
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

func HTTPCacheTransactionRecordNotCacheable(h *HTTPCacheHandle, opts *HTTPCacheWriteOptions) error {
	if err := fastlyHTTPCacheTransactionRecordNotCacheable(
		h.h,
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

func HTTPCacheTransactionAbandon(h *HTTPCacheHandle) error {
	if err := fastlyHTTPCacheTransactionAbandon(
		h.h,
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

func HTTPCacheTransactionClose(h *HTTPCacheHandle) error {
	if err := fastlyHTTPCacheTransactionClose(
		h.h,
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

func HTTPCacheGetSuggestedBackendRequest(h *HTTPCacheHandle) (*HTTPRequest, error) {
	var req requestHandle = invalidRequestHandle
	if err := fastlyHTTPCacheGetSuggestedBackendRequest(
		h.h,
		prim.ToPointer(&req),
	).toError(); err != nil {
		return nil, err
	}

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

func HTTPCacheGetSuggestedCacheOptions(h *HTTPCacheHandle, r *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPCacheWriteOptions, error) {
	var out HTTPCacheWriteOptions

	out.vary = prim.NewWriteBuffer(DefaultSmallBufLen)
	opts.opts.varyRulePtr = prim.ToPointer(out.vary.Char8Pointer())
	opts.opts.varyRuleLen = out.vary.Cap()

	out.surrogate = prim.NewWriteBuffer(DefaultMediumBufLen)
	opts.opts.surrogateKeysPtr = prim.ToPointer(out.surrogate.Char8Pointer())
	opts.opts.surrogateKeysLen = out.surrogate.Cap()

	for {
		status := fastlyHTTPCacheGetSuggestedCacheOptions(
			h.h,
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
				if n == 0 {
					// handle empty?
					n = 1
				}
				out.vary = prim.NewWriteBuffer(n)
				opts.opts.varyRulePtr = prim.ToPointer(out.vary.Char8Pointer())
				opts.opts.varyRuleLen = out.vary.Cap()
			}

			if out.mask&httpCacheWriteOptionsFlagSurrogateKeys == httpCacheWriteOptionsFlagSurrogateKeys {
				n := int(out.opts.surrogateKeysLen)
				out.surrogate = prim.NewWriteBuffer(n)
				opts.opts.surrogateKeysPtr = prim.ToPointer(out.surrogate.Char8Pointer())
				opts.opts.surrogateKeysLen = out.surrogate.Cap()
			}

			// reset out.mask for next call
			out.mask = 0

			continue
		}

		if err := status.toError(); err != nil {
			return nil, err
		}

		break
	}

	// make sure out.mask is set correctly for current filled in options
	out.mask = opts.mask
	return &out, nil
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
	action prim.Pointer[HTTPCacheStorageAction],
	newr prim.Pointer[responseHandle],
) FastlyStatus

func HTTPCachePrepareResponseForStorage(h *HTTPCacheHandle, r *HTTPResponse) (HTTPCacheStorageAction, *HTTPResponse, error) {
	var action HTTPCacheStorageAction
	var newr responseHandle = invalidResponseHandle

	if err := fastlyHTTPCachePrepareResponseForStorage(
		h.h,
		r.h,
		prim.ToPointer(&action),
		prim.ToPointer(&newr),
	).toError(); err != nil {
		return 0, nil, err
	}

	return action, &HTTPResponse{h: newr}, nil
}

// witx:
//
//    ;;; Retrieve a stored response from the cache, returning the `$none` error if there was no found
//    ;;; response.
//    ;;;
//    ;;; If `transform_for_client` is set, the response will be adjusted according to the looked-up
//    ;;; request. For example, a response retrieved for a range request may be transformed into a
//    ;;; `206 Partial Content` response with an appropriate `content-range` header.
//    (@interface func (export "get_found_response")
//        (param $handle $http_cache_handle)
//        (param $transform_for_client u32)
//        (result $err (expected (tuple $response_handle $body_handle) (error $fastly_status)))
//    )

//go:wasmimport fastly_http_cache get_found_response
//go:noescape
func fastlyHTTPCacheGetFoundResponse(
	h httpCacheHandle,
	transform prim.U32, // bool
	r prim.Pointer[responseHandle],
	b prim.Pointer[bodyHandle],
) FastlyStatus

func HTTPCacheGetFoundResponse(h *HTTPCacheHandle, transform bool) (*HTTPResponse, *HTTPBody, error) {
	var r responseHandle = invalidResponseHandle
	var b bodyHandle = invalidBodyHandle

	var t prim.U32
	if transform {
		t = 1
	}

	if err := fastlyHTTPCacheGetFoundResponse(
		h.h,
		t,
		prim.ToPointer(&r),
		prim.ToPointer(&b),
	).toError(); err != nil {
		return nil, nil, err
	}

	return &HTTPResponse{h: r}, &HTTPBody{h: b}, nil
}

// witx:
//
//    ;;; Get the state of a cache transaction.
//    ;;;
//    ;;; Primarily useful after performing the lookup to determine what subsequent operations are
//    ;;; possible and whether any insertion or update obligations exist.
//    (@interface func (export "get_state")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_lookup_state (error $fastly_status)))
//    )
//
//

//go:wasmimport fastly_http_cache get_state
//go:noescape
func fastlyHTTPCacheGetState(
	h httpCacheHandle,
	s prim.Pointer[CacheLookupState],
) FastlyStatus

func HTTPCacheGetState(h *HTTPCacheHandle) (CacheLookupState, error) {
	var s CacheLookupState

	if err := fastlyHTTPCacheGetState(
		h.h,
		prim.ToPointer(&s),
	).toError(); err != nil {
		return 0, err
	}

	return s, nil
}

// witx:
//
//    ;;; Get the length of the found response, returning the `$none` error if there was no found
//    ;;; response or no length was provided.
//    (@interface func (export "get_length")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_object_length (error $fastly_status)))
//    )

//go:wasmimport fastly_http_cache get_length
//go:noescape
func fastlyHTTPCacheGetLength(
	h httpCacheHandle,
	l prim.Pointer[httpCacheObjectLength],
) FastlyStatus

func HTTPCacheGetLength(h *HTTPCacheHandle) (httpCacheObjectLength, error) {
	var l httpCacheObjectLength

	if err := fastlyHTTPCacheGetLength(
		h.h,
		prim.ToPointer(&l),
	).toError(); err != nil {
		return 0, err
	}

	return l, nil
}

// witx:
//
//    ;;; Get the configured max age of the found response in nanoseconds, returning the `$none` error
//    ;;; if there was no found response.
//    (@interface func (export "get_max_age_ns")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_duration_ns (error $fastly_status)))
//    )

//go:wasmimport fastly_http_cache get_max_age_ns
//go:noescape
func fastlyHTTPCacheGetMaxAgeNs(
	h httpCacheHandle,
	d prim.Pointer[httpCacheDurationNs],
) FastlyStatus

func HTTPCacheGetMaxAgeNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	var d httpCacheDurationNs

	if err := fastlyHTTPCacheGetMaxAgeNs(
		h.h,
		prim.ToPointer(&d),
	).toError(); err != nil {
		return 0, err
	}

	return d, nil
}

// witx:
//
//    ;;; Get the configured stale-while-revalidate period of the found response in nanoseconds,
//    ;;; returning the `$none` error if there was no found response.
//    (@interface func (export "get_stale_while_revalidate_ns")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_duration_ns (error $fastly_status)))
//    )
//
//

//go:wasmimport fastly_http_cache get_stale_while_revalidate_ns
//go:noescape
func fastlyHTTPCacheGetStaleWhileRevalidateNs(
	h httpCacheHandle,
	d prim.Pointer[httpCacheDurationNs],
) FastlyStatus

func HTTPCacheGetStaleWhileRevalidateNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	var d httpCacheDurationNs

	if err := fastlyHTTPCacheGetStaleWhileRevalidateNs(
		h.h,
		prim.ToPointer(&d),
	).toError(); err != nil {
		return 0, err
	}

	return d, nil
}

// witx:
//
//    ;;; Get the age of the found response in nanoseconds, returning the `$none` error if there was
//    ;;; no found response.
//    (@interface func (export "get_age_ns")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_duration_ns (error $fastly_status)))
//    )
//
//

//go:wasmimport fastly_http_cache get_age_ns
//go:noescape
func fastlyHTTPCacheGetAgeNs(
	h httpCacheHandle,
	d prim.Pointer[httpCacheDurationNs],
) FastlyStatus

func HTTPCacheGetAgeNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	var d httpCacheDurationNs

	if err := fastlyHTTPCacheGetAgeNs(
		h.h,
		prim.ToPointer(&d),
	).toError(); err != nil {
		return 0, err
	}

	return d, nil
}

// witx:
//
//    ;;; Get the number of cache hits for the found response, returning the `$none` error if there
//    ;;; was no found response.
//    ;;;
//    ;;; Note that this figure only reflects hits for a stored response in a particular cache server
//    ;;; or cluster, not the entire Fastly network.
//    (@interface func (export "get_hits")
//        (param $handle $http_cache_handle)
//        (result $err (expected $cache_hit_count (error $fastly_status)))
//    )
//

//go:wasmimport fastly_http_cache get_hits
//go:noescape
func fastlyHTTPCacheGetHits(
	h httpCacheHandle,
	c prim.Pointer[httpCacheHitCount],
) FastlyStatus

func HTTPCacheGetHits(h *HTTPCacheHandle) (httpCacheHitCount, error) {
	var c httpCacheHitCount

	if err := fastlyHTTPCacheGetHits(
		h.h,
		prim.ToPointer(&c),
	).toError(); err != nil {
		return 0, err
	}

	return c, nil
}

// witx:
//
//    ;;; Get whether a found response is marked as containing sensitive data, returning the `$none`
//    ;;; error if there was no found response.
//    (@interface func (export "get_sensitive_data")
//        (param $handle $http_cache_handle)
//        (result $err (expected $is_sensitive (error $fastly_status)))
//    )
//
//

//go:wasmimport fastly_http_cache get_sensitive_data
//go:noescape
func fastlyHTTPCacheGetSensitiveData(
	h httpCacheHandle,
	b prim.Pointer[httpIsSensitive],
) FastlyStatus

func HTTPCacheGetSensitiveData(h *HTTPCacheHandle) (bool, error) {
	var b httpIsSensitive

	if err := fastlyHTTPCacheGetSensitiveData(
		h.h,
		prim.ToPointer(&b),
	).toError(); err != nil {
		return false, err
	}

	return b != 0, nil
}

// witx:
//
//    ;;; Get the surrogate keys of the found response, returning the `$none` error if there was no
//    ;;; found response.
//    ;;;
//    ;;; The output is a list of surrogate keys separated by spaces.
//    ;;;
//    ;;; If the guest-provided output parameter is not long enough to contain the full list of
//    ;;; surrogate keys, the required size is written by the host to `nwritten_out` and the `$buflen`
//    ;;; error is returned.
//    (@interface func (export "get_surrogate_keys")
//        (param $handle $http_cache_handle)
//        (param $surrogate_keys_out_ptr (@witx pointer u8))
//        (param $surrogate_keys_out_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//

//go:wasmimport fastly_http_cache get_surrogate_keys
//go:noescape
func fastlyHTTPCacheGetSurrogateKeys(
	h httpCacheHandle,
	buf prim.Pointer[prim.U8],
	bufLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
) FastlyStatus

func HTTPCacheGetSurrogateKeys(h *HTTPCacheHandle) (string, error) {
	n := DefaultMediumBufLen

	for {
		buf := prim.NewWriteBuffer(n) // Longest (unknown)
		status := fastlyHTTPCacheGetSurrogateKeys(
			h.h,
			prim.ToPointer(buf.U8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
		if status == FastlyStatusBufLen && buf.NValue() > 0 {
			n = int(buf.NValue())
			continue
		}
		if err := status.toError(); err != nil {
			return "", err
		}
		return string(buf.AsBytes()), nil
	}
}

// witx:
//
//    ;;; Get the vary rule of the found response, returning the `$none` error if there was no found
//    ;;; response.
//    ;;;
//    ;;; The output is a list of header names separated by spaces.
//    ;;;
//    ;;; If the guest-provided output parameter is not long enough to contain the full list of
//    ;;; surrogate keys, the required size is written by the host to `nwritten_out` and the `$buflen`
//    ;;; error is returned.
//    (@interface func (export "get_vary_rule")
//        (param $handle $http_cache_handle)
//        (param $vary_rule_out_ptr (@witx pointer u8))
//        (param $vary_rule_out_len (@witx usize))
//        (param $nwritten_out (@witx pointer (@witx usize)))
//        (result $err (expected (error $fastly_status)))
//    )
//

//go:wasmimport fastly_http_cache get_vary_rule
//go:noescape
func fastlyHTTPCacheGetVaryRule(
	h httpCacheHandle,
	buf prim.Pointer[prim.U8],
	bufLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
) FastlyStatus

func HTTPCacheGetVaryRule(h *HTTPCacheHandle) (string, error) {
	// Longest (unknown)
	value, err := withAdaptiveBuffer(DefaultSmallBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyHTTPCacheGetVaryRule(
			h.h,
			prim.ToPointer(buf.U8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}
