package fsthttp

import (
	"fmt"
	"io"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// CandidateResponse is a response from a backend that is a candidate for caching.
type CandidateResponse struct {
	cacheHandle *fastly.HTTPCacheHandle

	abiResp *fastly.HTTPResponse
	abiBody *fastly.HTTPBody

	suggestedCacheWriteOptions *cacheWriteOptions
	suggestedStorageAction     fastly.HTTPCacheStorageAction

	overrideStorageAction fastly.HTTPCacheStorageAction
	useStorageAction      bool

	overridePCI bool
	usePCI      bool

	overrideStaleWhileRevalidate uint32 // seconds
	useSWR                       bool

	extraSurrogateKeys    string
	overrideSurrogateKeys string
	useSurrogate          bool

	overrideTTL uint32 // seconds
	useTTL      bool

	overrideVary string
	useVary      bool

	bodyTransform func(io.ReadCloser) io.ReadCloser
}

type cacheResponse struct {
	cacheWriteOptions cacheWriteOptions
	storageAction     fastly.HTTPCacheStorageAction
	hits              uint64
}

type cacheWriteOptions struct {
	maxAge    uint32 // seconds
	vary      string
	useVary   bool
	age       uint32 // seconds
	stale     uint32 // seconds
	surrogate string
	length    uint64
	useLength bool
	sensitive bool

	abiOpts fastly.HTTPCacheWriteOptions
}

func u64nsTou32s(ns uint64) uint32 { return uint32(time.Duration(ns) / time.Second) }

func u32sTou64ns(s uint32) uint64 { return uint64(time.Duration(s) * time.Second) }

// TODO(dgryski): why no error return here?
func (opts *cacheWriteOptions) flushToABI() {
	opts.abiOpts.SetMaxAgeNs(u32sTou64ns(opts.maxAge))
	opts.abiOpts.SetVaryRule(opts.vary)
	opts.abiOpts.SetInitialAgeNs(u32sTou64ns(opts.age))
	opts.abiOpts.SetStaleWhileRevalidateNs(u32sTou64ns(opts.stale))
	opts.abiOpts.SetSurrogateKeys(opts.surrogate)
	opts.abiOpts.SetSensitiveData(opts.sensitive)
}

func (opts *cacheWriteOptions) loadFromABI() {
	opts.maxAge = u64nsTou32s(opts.abiOpts.MaxAgeNs())
	opts.vary, opts.useVary = opts.abiOpts.VaryRule()
	if ns, ok := opts.abiOpts.InitialAgeNs(); ok {
		opts.age = u64nsTou32s(ns)
	}
	if ns, ok := opts.abiOpts.StaleWhileRevalidateNs(); ok {
		opts.stale = u64nsTou32s(ns)
	}
	opts.surrogate, _ = opts.abiOpts.SurrogateKeys()
	opts.length, opts.useLength = opts.abiOpts.Length()
	opts.sensitive = opts.abiOpts.SensitiveData()
}

func (opts *cacheWriteOptions) loadFromHandle(c *fastly.HTTPCacheHandle) error {

	var err error
	if ns, err := fastly.HTTPCacheGetMaxAgeNs(c); err != nil {
		return fmt.Errorf("get max age: %w", err)
	} else {
		opts.maxAge = u64nsTou32s(uint64(ns))
	}

	opts.vary, err = fastly.HTTPCacheGetVaryRule(c)
	if err != nil {
		return fmt.Errorf("get vary rule: %w", err)
	}

	if ns, err := fastly.HTTPCacheGetAgeNs(c); err != nil {
		return fmt.Errorf("get age: %w", err)
	} else {
		opts.age = u64nsTou32s(uint64(ns))
	}

	if ns, err := fastly.HTTPCacheGetStaleWhileRevalidateNs(c); err != nil {
		return fmt.Errorf("get stale while revalidate: %w", err)
	} else {
		opts.stale = u64nsTou32s(uint64(ns))
	}

	opts.surrogate, err = fastly.HTTPCacheGetSurrogateKeys(c)
	if err != nil {
		return fmt.Errorf("get surrogate keys: %w", err)
	}

	if ln, err := fastly.HTTPCacheGetLength(c); err != nil {
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusNone {
			opts.length = 0
			opts.useLength = false
		} else {
			return fmt.Errorf("get length: %w", err)
		}
	} else {
		opts.useLength = true
		opts.length = uint64(ln)
	}

	opts.sensitive, err = fastly.HTTPCacheGetSensitiveData(c)
	if err != nil {
		return fmt.Errorf("get sensitive data: %w", err)
	}

	return nil
}

const (
	// TODO(dgryski): not sure I like this solution
	cacheStorageActionInvalid = 0xffff
)

func httpCacheWait(c *fastly.HTTPCacheHandle) error {
	_, err := fastly.HTTPCacheGetState(c)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}
	return nil
}

func httpCacheMustInsertOrUpdate(c *fastly.HTTPCacheHandle) (bool, error) {
	state, err := fastly.HTTPCacheGetState(c)
	if err != nil {
		return false, fmt.Errorf("get state: %w", err)

	}
	return state&fastly.CacheLookupStateMustInsertOrUpdate == fastly.CacheLookupStateMustInsertOrUpdate, nil
}

func httpCacheGetFoundResponse(c *fastly.HTTPCacheHandle, req *Request, backend string, transformForClient bool) (*Response, error) {
	abiResp, abiBody, err := fastly.HTTPCacheGetFoundResponse(c, transformForClient)
	if err != nil {
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusNone {
			return nil, nil
		}
		return nil, fmt.Errorf("get found response: %w", err)
	}

	hits, err := fastly.HTTPCacheGetHits(c)
	if err != nil {
		return nil, fmt.Errorf("get hits: %w", err)
	}

	var opts cacheWriteOptions
	if err := opts.loadFromHandle(c); err != nil {
		return nil, fmt.Errorf("load cache options from handle: %w", err)
	}
	opts.flushToABI()

	resp, err := newResponse(req, backend, abiResp, abiBody)
	if err != nil {
		return nil, fmt.Errorf("new response: %w", err)
	}

	resp.cacheResponse = cacheResponse{
		cacheWriteOptions: opts,
		storageAction:     cacheStorageActionInvalid,
		hits:              uint64(hits),
	}
	return resp, nil
}

func newCandidateFromPendingBackendCaching(pending *pendingBackendRequestForCaching) (*CandidateResponse, error) {
	// pending backend request for caching -> into_candidate
	abiResp, abiBody, err := pending.pending.Wait()
	if err != nil {
		return nil, err
	}

	candidate, err := newCandidate(pending.cacheHandle, &pending.cacheOptions, abiResp, abiBody)
	if err != nil {
		return nil, fmt.Errorf("new candidate: %w", err)
	}

	if fn := pending.afterSend; fn != nil {
		if err := fn(candidate); err != nil {
			return nil, fmt.Errorf("after send: %w", err)
		}
		// Don't need to flush config here because that will happen in finalizeOptions()
		// which is called from applyAndStreamBack
	}

	return candidate, nil
}

func newCandidate(c *fastly.HTTPCacheHandle, opts *CacheOptions, abiResp *fastly.HTTPResponse, abiBody *fastly.HTTPBody) (*CandidateResponse, error) {
	storageAction, abiResp, err := fastly.HTTPCachePrepareResponseForStorage(c, abiResp)
	if err != nil {
		return nil, fmt.Errorf("prepare response for storage: %w", err)
	}

	// Fastly-specific heuristic: by default, we do not cache responses that set cookies
	if v, err := abiResp.GetHeaderValue("set-cookie", ResponseLimits.maxHeaderValueLen); err == nil && v != "" && storageAction != fastly.HTTPCacheStorageActionDoNotStore {
		storageAction = fastly.HTTPCacheStorageActionRecordUncacheable
	}

	candidate := CandidateResponse{
		cacheHandle:                c,
		abiResp:                    abiResp,
		abiBody:                    abiBody,
		suggestedCacheWriteOptions: nil,
		bodyTransform:              nil,
		suggestedStorageAction:     storageAction,

		overrideStorageAction:        0,
		overridePCI:                  opts.PCI,
		overrideStaleWhileRevalidate: opts.StaleWhileRevalidate,
		extraSurrogateKeys:           opts.SurrogateKey,
		overrideSurrogateKeys:        "",
		overrideTTL:                  opts.TTL,
		overrideVary:                 "",
	}

	if candidate.overrideTTL != 0 {
		candidate.useTTL = true
	}

	if candidate.overrideStaleWhileRevalidate != 0 {
		candidate.useSWR = true
	}

	if candidate.overridePCI {
		candidate.usePCI = true
	}

	return &candidate, nil
}

// SetHeader sets a header on the candidate response.
func (candidateResponse *CandidateResponse) SetHeader(key string, value string) error {
	if err := candidateResponse.abiResp.SetHeaderValues(key, []string{value}); err != nil {
		return fmt.Errorf("set header value: %w", err)
	}
	return nil
}

// DelHeader deletes a header from the candidate response.
func (candidateResponse *CandidateResponse) DelHeader(key string) error {
	if err := candidateResponse.abiResp.RemoveHeader(key); err != nil {
		return fmt.Errorf("remove header: %w", err)
	}
	return nil
}

// Header gets a header from the candidate response.
func (candidateResponse *CandidateResponse) Header(key string) (string, error) {
	v, err := candidateResponse.abiResp.GetHeaderValue(key, ResponseLimits.maxHeaderValueLen)
	if err != nil {
		return "", fmt.Errorf("get header: %w", err)
	}
	return v, nil
}

// SetCacheable marks the response as cacheable.
//
// Forces this response to be stored in the cache, even if its headers or
// status would normally prevent that.
func (candidateResponse *CandidateResponse) SetCacheable() {
	if !candidateResponse.useStorageAction {
		candidateResponse.useStorageAction = true
		candidateResponse.overrideStorageAction = candidateResponse.suggestedStorageAction
	}

	if candidateResponse.overrideStorageAction != fastly.HTTPCacheStorageActionUpdate {
		candidateResponse.overrideStorageAction = fastly.HTTPCacheStorageActionInsert
	}
}

// SetUncacheable marks the response as not to be stored in the cache.
//
// See the [Fastly request collapsing guide] for more details on the mechanism
// that `recordUncacheable` disables.
//
// [Fastly request collapsing guide]: https://www.fastly.com/documentation/guides/concepts/edge-state/cache/request-collapsing/
func (candidateResponse *CandidateResponse) SetUncacheable() {
	candidateResponse.useStorageAction = true
	candidateResponse.overrideStorageAction = fastly.HTTPCacheStorageActionDoNotStore
}

// SetUncacheableDisableCollapsing marks the response as not to be stored in the cache
// due to being an uncacheable response.
//
// Future cache lookups will result in immediately going to the backend, rather
// than attempting to coordinate concurrent requests to reduce backend traffic.
//
// See the [Fastly request collapsing guide] for more details on the mechanism
// that `recordUncacheable` disables.
//
// [Fastly request collapsing guide]: https://www.fastly.com/documentation/guides/concepts/edge-state/cache/request-collapsing/
func (candidateResponse *CandidateResponse) SetUncacheableDisableCollapsing() {
	candidateResponse.useStorageAction = true
	candidateResponse.overrideStorageAction = fastly.HTTPCacheStorageActionRecordUncacheable
}

// SetStatus sets the HTTP Status of the candidate response.
func (candidateResponse *CandidateResponse) SetStatus(status int) error {
	if err := candidateResponse.abiResp.SetStatusCode(status); err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	return nil
}

// Status gets the status of the candidate response.
func (candidateResponse *CandidateResponse) Status() (int, error) {
	status, err := candidateResponse.abiResp.GetStatusCode()
	if err != nil {
		return 0, fmt.Errorf("get status: %w", err)
	}
	return status, nil
}

// IsStale returns whether the cached response is stale.
//
// A cached response is stale if it has been in the cache beyond its TTL period.
func (candidateResponse *CandidateResponse) IsStale() (bool, error) {
	state, err := fastly.HTTPCacheGetState(candidateResponse.cacheHandle)
	if err != nil {
		return false, fmt.Errorf("get state: %w", err)
	}
	return state&fastly.CacheLookupStateStale == fastly.CacheLookupStateStale, nil
}

// Age returns current age in seconds of the cached item, relative to the originating backend.
func (candidateResponse *CandidateResponse) Age() (uint32, error) {
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return 0, err
	}
	return opts.age, nil
}

// TTL returns the Time to Live (TTL) in seconds in the cache for this response.
//
// The TTL determines the duration of "freshness" for the cached response
// after it is inserted into the cache.
func (candidateResponse *CandidateResponse) TTL() (uint32, error) {
	if candidateResponse.useTTL {
		return candidateResponse.overrideTTL, nil
	}
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return 0, err
	}

	return opts.maxAge - opts.age, nil
}

// SetTTL sets the Time to Live (TTL) in seconds in the cache for this response.
//
// The TTL determines the duration of "freshness" for the cached response
// after it is inserted into the cache.
func (candidateResponse *CandidateResponse) SetTTL(ttl uint32) {
	candidateResponse.overrideTTL = ttl
	candidateResponse.useTTL = true
}

// SetStaleWhileRevalidate sets the time in seconds for which a cached item can safely be used despite being considered stale.
func (candidateResponse *CandidateResponse) SetStaleWhileRevalidate(swr uint32) {
	candidateResponse.overrideStaleWhileRevalidate = swr
	candidateResponse.useSWR = true
}

// StaleWhileRevalidate returns the time in seconds for which a cached item can safely be used despite being considered stale.
func (candidateResponse *CandidateResponse) StaleWhileRevalidate() (uint32, error) {
	if candidateResponse.useSWR {
		return candidateResponse.overrideStaleWhileRevalidate, nil
	}
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return 0, err
	}
	return opts.stale, nil
}

// SetSensitive sets the caching behavior of this response to enable or disable PCI/HIPAA-compliant
// non-volatile caching.
//
// By default, this is `false`, which means the response may not be PCI/HIPAA-compliant. Set it
// to `true` to enable compliant caching.
//
// See the [Fastly PCI-Compliant Caching and Delivery documentation] for details.
//
// [Fastly PCI-Compliant Caching and Delivery documentation]: https://docs.fastly.com/products/pci-compliant-caching-and-delivery)
func (candidateResponse *CandidateResponse) SetSensitive(sensitive bool) {
	candidateResponse.overridePCI = sensitive
	candidateResponse.usePCI = true
}

// Sensitive returns whether this response should only be stored via PCI/HIPAA-compliant non-volatile caching.
//
// See the [Fastly PCI-Compliant Caching and Delivery documentation] for details.
//
// [Fastly PCI-Compliant Caching and Delivery documentation]: https://docs.fastly.com/products/pci-compliant-caching-and-delivery
func (candidateResponse *CandidateResponse) Sensitive() (bool, error) {
	if candidateResponse.usePCI {
		return candidateResponse.overridePCI, nil
	}
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return false, err
	}
	return opts.sensitive, nil
}

// SetVary sets the set of request headers for which the response may vary.
func (candidateResponse *CandidateResponse) SetVary(vary string) {
	candidateResponse.overrideVary = vary
	candidateResponse.useVary = true
}

// Vary returns the set of request headers for which the response may vary.
func (candidateResponse *CandidateResponse) Vary() (string, error) {
	if candidateResponse.useVary {
		return candidateResponse.overrideVary, nil
	}
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return "", err
	}
	return opts.vary, nil
}

// SetSurrogateKeys sets the surrogate keys for the cached response.
//
// Surrogate keys must contain only printable ASCII characters (those between `0x21` and
// `0x7E`, inclusive). Any invalid keys will be ignored.
//
// See the [Fastly surrogate keys guide] for details.
//
// [Fastly surrogate keys guide]: https://docs.fastly.com/en/guides/purging-api-cache-with-surrogate-keys
func (candidateResponse *CandidateResponse) SetSurrogateKeys(keys string) {
	candidateResponse.overrideSurrogateKeys = keys
	candidateResponse.useSurrogate = true
}

// SurrogateKeys returns the surrogate keys for the cached response.
func (candidateResponse *CandidateResponse) SurrogateKeys() (string, error) {
	if candidateResponse.useSurrogate {
		return candidateResponse.overrideSurrogateKeys, nil
	}
	opts, err := candidateResponse.getSuggestedCacheWriteOptions()
	if err != nil {
		return "", err
	}
	return opts.surrogate, nil
}

// SetBodyTransform sets a callback to be used for transforming the response body prior to
// caching.
//
// Body transformations are performed via a callback, rather than by
// directly working with the body, because not every response contains a
// fresh body: 304 Not Modified responses, which are used to revalidate a
// stale cached response, are valuable precisely because they do not
// retransmit the body.
//
// For any other response status, the backend response will contain a relevant
// body, and the `transform` callback will be invoked to return a new
// `io.ReadCloser` for the transformed body.
func (candidateResponse *CandidateResponse) SetBodyTransform(fn func(io.ReadCloser) io.ReadCloser) {
	candidateResponse.bodyTransform = fn
}

// BodyTransform returns the current body transformation function.
func (candidateResponse *CandidateResponse) BodyTransform() func(io.ReadCloser) io.ReadCloser {
	return candidateResponse.bodyTransform
}

func (candidateResponse *CandidateResponse) getSuggestedCacheWriteOptions() (*cacheWriteOptions, error) {
	if candidateResponse.suggestedCacheWriteOptions == nil {
		opts, err := candidateResponse.buildFreshSuggestedCacheWriteOptions()
		if err != nil {
			return nil, err
		}
		candidateResponse.suggestedCacheWriteOptions = opts
	}

	return candidateResponse.suggestedCacheWriteOptions, nil
}

func (candidateResponse *CandidateResponse) buildFreshSuggestedCacheWriteOptions() (*cacheWriteOptions, error) {
	opts, err := httpCacheGetSuggestedCacheWriteOptions(candidateResponse.cacheHandle, candidateResponse.abiResp)
	if err != nil {
		return nil, err
	}

	keys := opts.surrogate
	if extra := candidateResponse.extraSurrogateKeys; extra != "" {
		if keys != "" {
			keys += " "
		}
		keys += extra
	}

	vals := candidateResponse.abiResp.GetHeaderValues(surrogateKey, RequestLimits.maxHeaderValueLen)
	for vals.Next() {
		if keys != "" {
			keys += " "
		}
		keys += string(vals.Bytes())
	}
	if err := vals.Err(); err != nil {
		return nil, fmt.Errorf("get header values: %w", err)
	}
	opts.surrogate = keys

	return opts, nil
}

func httpCacheGetSuggestedCacheWriteOptions(cacheHandle *fastly.HTTPCacheHandle, resp *fastly.HTTPResponse) (*cacheWriteOptions, error) {
	var opts cacheWriteOptions
	opts.abiOpts.FillConfigMask()

	newABIOpts, err := fastly.HTTPCacheGetSuggestedCacheOptions(cacheHandle, resp, &opts.abiOpts)
	if err != nil {
		return nil, fmt.Errorf("get suggested cache options: %w", err)
	}

	newOpts := &cacheWriteOptions{abiOpts: *newABIOpts}
	newOpts.loadFromABI()

	return newOpts, nil
}

func (candidateResponse *CandidateResponse) finalizeOptions() (fastly.HTTPCacheStorageAction, *cacheWriteOptions, error) {
	var storageAction = candidateResponse.suggestedStorageAction
	if candidateResponse.useStorageAction {
		storageAction = candidateResponse.overrideStorageAction
	}

	suggestedCacheWriteOptions := candidateResponse.suggestedCacheWriteOptions
	if suggestedCacheWriteOptions == nil {
		var err error
		suggestedCacheWriteOptions, err = candidateResponse.buildFreshSuggestedCacheWriteOptions()
		if err != nil {
			return 0, nil, err
		}
	}

	var opts cacheWriteOptions

	if candidateResponse.useTTL {
		opts.maxAge = candidateResponse.overrideTTL - suggestedCacheWriteOptions.age
	} else {
		opts.maxAge = suggestedCacheWriteOptions.maxAge
	}

	opts.age = suggestedCacheWriteOptions.age

	if candidateResponse.useSWR {
		opts.stale = candidateResponse.overrideStaleWhileRevalidate
	} else {
		opts.stale = suggestedCacheWriteOptions.stale
	}

	if candidateResponse.useVary {
		opts.vary = candidateResponse.overrideVary
	} else {
		opts.vary = suggestedCacheWriteOptions.vary
	}

	if candidateResponse.useSurrogate {
		opts.surrogate = candidateResponse.overrideSurrogateKeys
	} else {
		opts.surrogate = suggestedCacheWriteOptions.surrogate
	}

	if candidateResponse.usePCI {
		opts.sensitive = candidateResponse.overridePCI
	} else {
		opts.sensitive = suggestedCacheWriteOptions.sensitive
	}

	if candidateResponse.bodyTransform == nil {
		if len, ok := bodyHasKnownLength(candidateResponse.abiBody); ok {
			opts.length = len
			opts.useLength = true
		}
	}

	opts.flushToABI()

	return storageAction, &opts, nil
}

func bodyHasKnownLength(body io.ReadCloser) (uint64, bool) {
	if abiBody, ok := body.(*fastly.HTTPBody); ok {
		l, err := abiBody.Length()
		if err != nil {
			return 0, false
		}
		return l, true
	}

	if lenner, ok := body.(interface{ Len() int }); ok {
		return uint64(lenner.Len()), true
	}

	return 0, false
}

func (candidateResponse *CandidateResponse) applyAndStreamBack(req *Request) (*Response, error) {
	var resp *Response

	action, opts, err := candidateResponse.finalizeOptions()
	if err != nil {
		return nil, fmt.Errorf("finalize options: %w", err)
	}
	switch action {
	case fastly.HTTPCacheStorageActionInsert:
		body, readback, err := fastly.HTTPCacheTransactionInsertAndStreamback(candidateResponse.cacheHandle, candidateResponse.abiResp, &opts.abiOpts)
		if err != nil {
			return nil, fmt.Errorf("cache transaction insert and stream back: %w", err)
		}
		defer fastly.HTTPCacheTransactionClose(readback)

		if fn, respBody := candidateResponse.bodyTransform, candidateResponse.abiBody; fn != nil {
			if _, err := io.Copy(body, fn(respBody)); err != nil {
				return nil, fmt.Errorf("bodyTransform: io.Copy: %w", err)
			}
		} else if err := body.Append(respBody); err != nil {
			return nil, fmt.Errorf("body.Append: %w", err)
		}
		body.Close()

		resp, err = httpCacheGetFoundResponse(readback, req, "", false)
		if err != nil {
			return nil, fmt.Errorf("cache get found response: %w", err)
		}

	case fastly.HTTPCacheStorageActionUpdate:
		newch, err := fastly.HTTPCacheTransactionUpdateAndReturnFresh(candidateResponse.cacheHandle, candidateResponse.abiResp, &opts.abiOpts)
		if err != nil {
			return nil, fmt.Errorf("cache transaction update and return fresh: %w", err)
		}
		defer fastly.HTTPCacheTransactionClose(newch)

		resp, err = httpCacheGetFoundResponse(newch, req, "", true)
		if err != nil {
			return nil, fmt.Errorf("cache get found response: %w", err)
		}

	case fastly.HTTPCacheStorageActionDoNotStore:
		// Use `abandon` to only wake a single waiter in the
		// non-hit-for-pass case, so concurrent requests remain
		// serialized.

		if err := fastly.HTTPCacheTransactionAbandon(candidateResponse.cacheHandle); err != nil {
			return nil, fmt.Errorf("cache transaction abandon: %w", err)
		}

		resp, err = newResponseFromCandidate(candidateResponse, req, opts)
		if err != nil {
			return nil, err
		}

	case fastly.HTTPCacheStorageActionRecordUncacheable:
		err := fastly.HTTPCacheTransactionRecordNotCacheable(candidateResponse.cacheHandle, &opts.abiOpts)
		if err != nil {
			return nil, fmt.Errorf("cache transaction record not cacheable: %w", err)
		}

		resp, err = newResponseFromCandidate(candidateResponse, req, opts)
		if err != nil {
			return nil, err
		}
	}

	resp.cacheResponse.storageAction = action
	return resp, nil
}

func (candidateResponse *CandidateResponse) applyInBackground() error {
	action, opts, err := candidateResponse.finalizeOptions()
	if err != nil {
		return err
	}
	switch action {
	case fastly.HTTPCacheStorageActionInsert:
		body, err := fastly.HTTPCacheTransactionInsert(candidateResponse.cacheHandle, candidateResponse.abiResp, &opts.abiOpts)
		if err != nil {
			return fmt.Errorf("cache transaction insert: %w", err)
		}

		if fn, respBody := candidateResponse.bodyTransform, candidateResponse.abiBody; fn != nil {
			if _, err := io.Copy(body, fn(respBody)); err != nil {
				return fmt.Errorf("bodyTransform: io.Copy: %w", err)
			}
		} else if err := body.Append(respBody); err != nil {
			return fmt.Errorf("body.Append(): %w", err)
		}
		body.Close()

	case fastly.HTTPCacheStorageActionUpdate:
		err := fastly.HTTPCacheTransactionUpdate(candidateResponse.cacheHandle, candidateResponse.abiResp, &opts.abiOpts)
		if err != nil {
			return fmt.Errorf("cache transaction update: %w", err)
		}

	case fastly.HTTPCacheStorageActionDoNotStore:
		// Use `abandon` to only wake a single waiter in the
		// non-hit-for-pass case, so concurrent requests remain
		// serialized.
		if err := fastly.HTTPCacheTransactionAbandon(candidateResponse.cacheHandle); err != nil {
			return fmt.Errorf("cache transaction abandon: %w", err)
		}

	case fastly.HTTPCacheStorageActionRecordUncacheable:
		err := fastly.HTTPCacheTransactionRecordNotCacheable(candidateResponse.cacheHandle, &opts.abiOpts)
		if err != nil {
			return fmt.Errorf("cache transaction record not cacheable: %w", err)
		}
	}

	return nil
}

func newResponseFromCandidate(candidate *CandidateResponse, req *Request, opts *cacheWriteOptions) (*Response, error) {
	resp, err := newResponse(req, "", candidate.abiResp, candidate.abiBody)
	if err != nil {
		return nil, fmt.Errorf("newResponse: %w", err)
	}

	if fn := candidate.bodyTransform; fn != nil {
		resp.Body = candidate.bodyTransform(resp.Body)
	}
	resp.cacheResponse.cacheWriteOptions = *opts
	return resp, nil
}
