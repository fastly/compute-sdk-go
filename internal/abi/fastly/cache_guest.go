//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

type CacheLookupOptions struct {
	opts cacheLookupOptions
	mask cacheLookupOptionsMask
}

func (o *CacheLookupOptions) SetRequest(req *HTTPRequest) {
	o.opts.requestHeaders = req.h
	o.mask |= cacheLookupOptionsMaskRequestHeaders
}

func (o *CacheLookupOptions) SetAlwaysUseRequestedRange(alwaysUseRequestedRange bool) {
	if alwaysUseRequestedRange {
		o.mask |= cacheLookupOptionsMaskAlwaysUseRequestedRange
	} else {
		o.mask &= ^cacheLookupOptionsMaskAlwaysUseRequestedRange
	}
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
	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyCacheGetUserMetadata(
			c.h,
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
func fastlyPurgeSurrogateKey(
	surrogateKeyData prim.Pointer[prim.U8], surrogateKeyLen prim.Usize,
	mask purgeOptionsMask,
	opts prim.Pointer[purgeOptions],
) FastlyStatus

func PurgeSurrogateKey(surrogateKey string, opts PurgeOptions) error {
	surrogateKeyBuffer := prim.NewReadBufferFromString(surrogateKey).Wstring()

	return fastlyPurgeSurrogateKey(
		surrogateKeyBuffer.Data, surrogateKeyBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
	).toError()
}
