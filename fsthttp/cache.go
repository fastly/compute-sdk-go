package fsthttp

import (
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Cache is a type to namespace the HTTPCache methods
type Cache struct{}

func (Cache) IsRequestCacheable(r *Request) (bool, error) {
	if err := r.constructABIRequest(); err != nil {
		return false, err
	}

	ok, err := fastly.HTTPCacheIsRequestCacheable(r.abi.req)
	return ok, err
}

func (Cache) GetSuggestedCacheKey(r *Request) ([]byte, error) {
	if err := r.constructABIRequest(); err != nil {
		return nil, err
	}

	b, err := fastly.HTTPCacheGetSuggestedCacheKey(r.abi.req)
	return b, err
}

type CacheLookupOptions struct {
	abiOpts fastly.HTTPCacheLookupOptions
}

func (c *CacheLookupOptions) OverrideKey(key []byte) {
	c.abiOpts.OverrideKey(key)
}

type CacheHandle struct {
	h *fastly.HTTPCacheHandle
}

func (Cache) Lookup(r *Request, opts *CacheLookupOptions) (*CacheHandle, error) {
	if err := r.constructABIRequest(); err != nil {
		return nil, err
	}

	h, err := fastly.HTTPCacheLookup(r.abi.req, &opts.abiOpts)
	if err != nil {
		return nil, err
	}

	return &CacheHandle{h: h}, nil
}

func (Cache) TransactionLookup(r *Request, opts *CacheLookupOptions) (*CacheHandle, error) {
	if err := r.constructABIRequest(); err != nil {
		return nil, err
	}

	h, err := fastly.HTTPCacheTransactionLookup(r.abi.req, &opts.abiOpts)
	if err != nil {
		return nil, err
	}

	return &CacheHandle{h: h}, nil
}

type CacheWriteOptions struct {
	abiOpts *fastly.HTTPCacheWriteOptions
}

func (Cache) TransactionInsert(h *CacheHandle, r *Response, opts *CacheWriteOptions) (io.ReadCloser, error) {
	body, err := fastly.HTTPCacheTransactionInsert(h.h, r.abi.resp, opts.abiOpts)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (Cache) TransactionInsertAndStreamback(h *CacheHandle, r *Response, opts *CacheWriteOptions) (io.ReadCloser, *CacheHandle, error) {
	body, handle, err := fastly.HTTPCacheTransactionInsertAndStreamback(h.h, r.abi.resp, opts.abiOpts)
	if err != nil {
		return nil, nil, err
	}

	return body, &CacheHandle{h: handle}, nil
}

func (Cache) TransactionUpdate(h *CacheHandle, r *Response, opts *CacheWriteOptions) error {
	err := fastly.HTTPCacheTransactionUpdate(h.h, r.abi.resp, opts.abiOpts)
	if err != nil {
		return err
	}

	return nil
}

func (Cache) TransactionUpdateAndReturnFresh(h *CacheHandle, r *Response, opts *CacheWriteOptions) (*CacheHandle, error) {
	newh, err := fastly.HTTPCacheTransactionUpdateAndReturnFresh(h.h, r.abi.resp, opts.abiOpts)
	if err != nil {
		return nil, err
	}

	return &CacheHandle{h: newh}, nil
}

func (Cache) TransactionRecordNotCacheable(h *CacheHandle, opts *CacheWriteOptions) error {
	err := fastly.HTTPCacheTransactionRecordNotCacheable(h.h, opts.abiOpts)
	if err != nil {
		return err
	}

	return nil
}

func (Cache) TransactionAbandon(h *CacheHandle) error {
	err := fastly.HTTPCacheTransactionAbandon(h.h)
	if err != nil {
		return err
	}

	return nil
}

func (Cache) TransactionClose(h *CacheHandle) error {
	err := fastly.HTTPCacheTransactionClose(h.h)
	if err != nil {
		return err
	}

	return nil
}

type RequestHandle struct {
	h *fastly.HTTPRequest
}

type ResponseHandle struct {
	h *fastly.HTTPResponse
}

func (Cache) GetSuggestedBackendRequest(h *CacheHandle) (*RequestHandle, error) {
	r, err := fastly.HTTPCacheGetSuggestedBackendRequest(h.h)
	if err != nil {
		return nil, err
	}

	return &RequestHandle{h: r}, nil
}

type StorageAction uint32

const (
	// Insert the response into cache (`transaction_insert*`).
	StorageActionInsert StorageAction = 0

	// Update the stale response in cache (`transaction_update*`).
	StorageActionUpdate StorageAction = 1

	// Do not store this response.
	StorageActionDoNotStore StorageAction = 2

	// Do not store this response, and furthermore record its non-cacheability for other pending
	// requests (`transaction_record_not_cacheable`).
	StorageActionRecordUnreachable StorageAction = 3
)

func (Cache) GetSuggestedCacheOptions(h *CacheHandle, r *Response, opts *CacheWriteOptions) error {
	err := fastly.HTTPCacheGetSuggestedCacheOptions(h.h, r.abi.resp, opts.abiOpts)
	if err != nil {
		return err
	}

	return nil
}

func (Cache) PrepareResponseForStorage(h *CacheHandle, r *Response) (StorageAction, *ResponseHandle, error) {
	action, rh, err := fastly.HTTPCachePrepareResponseForStorage(h.h, r.abi.resp)
	if err != nil {
		return 0, nil, err
	}

	return StorageAction(action), &ResponseHandle{h: rh}, nil
}

func (Cache) GetFoundResponse(h *CacheHandle, transformForClient bool) (*ResponseHandle, io.ReadCloser, error) {
	r, b, err := fastly.HTTPCacheGetFoundResponse(h.h, transformForClient)
	if err != nil {
		return nil, nil, err
	}

	return &ResponseHandle{h: r}, b, nil
}

type CacheLookupState uint32

const (

	//              $found ;; a cached object was found
	CacheLookupStateFound CacheLookupState = 0b0000_0001 // $found

	//              $usable ;; the cached object is valid to use (implies $found)
	CacheLookupStateUsable CacheLookupState = 0b0000_0010 // $usable

	//              $stale ;; the cached object is stale (but may or may not be valid to use)
	CacheLookupStateStale CacheLookupState = 0b0000_0100 // $stale

	//              $must_insert_or_update ;; this client is requested to insert or revalidate an object
	CacheLookupStateMustInsertOrUpdate CacheLookupState = 0b0000_1000 // $must_insert_or_update
)

func (Cache) GetState(h *CacheHandle) (CacheLookupState, error) {
	s, err := fastly.HTTPCacheGetState(h.h)
	if err != nil {
		return 0, err
	}
	return CacheLookupState(s), nil
}

func (Cache) GetLength(h *CacheHandle) (uint64, error) {
	l, err := fastly.HTTPCacheGetLength(h.h)
	if err != nil {
		return 0, err
	}
	return uint64(l), nil
}

func (Cache) GetMaxAgeNs(h *CacheHandle) (uint64, error) {
	ns, err := fastly.HTTPCacheGetMaxAgeNs(h.h)
	if err != nil {
		return 0, err
	}
	return uint64(ns), nil
}

func (Cache) GetStaleWhileRevalidate(h *CacheHandle) (uint64, error) {
	ns, err := fastly.HTTPCacheGetStaleWhieRevalidate(h.h)
	if err != nil {
		return 0, err
	}
	return uint64(ns), nil
}

func (Cache) GetAgeNs(h *CacheHandle) (uint64, error) {
	ns, err := fastly.HTTPCacheGetAgeNs(h.h)
	if err != nil {
		return 0, err
	}
	return uint64(ns), nil
}

func (Cache) GetHits(h *CacheHandle) (uint64, error) {
	hits, err := fastly.HTTPCacheGetHits(h.h)
	if err != nil {
		return 0, err
	}
	return uint64(hits), nil
}

func (Cache) GetSensitiveData(h *CacheHandle) (bool, error) {
	b, err := fastly.HTTPCacheGetSensitiveData(h.h)
	if err != nil {
		return false, err
	}
	return b, nil
}

func (Cache) GetSurrogateKeys(h *CacheHandle) (string, error) {
	s, err := fastly.HTTPCacheGetSurrogateKeys(h.h)
	if err != nil {
		return "", err
	}
	return s, nil
}

func (Cache) GetVaryRule(h *CacheHandle) (string, error) {
	s, err := fastly.HTTPCacheGetVaryRule(h.h)
	if err != nil {
		return "", err
	}
	return s, nil
}
