//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

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
