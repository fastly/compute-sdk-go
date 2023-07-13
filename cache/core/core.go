// Package core provides the Fastly Core Cache API.
//
// This package exposes the primitive operations required to implement
// high-performance cache applications with advanced features such as
// [request collapsing], [streaming miss], [revalidation], and
// [surrogate key purging].
//
// While this API contains affordances for some HTTP caching concepts
// such as Vary headers and stale-while-revalidate, this API is not
// suitable for HTTP caching out-of-the-box.  Future SDK releases will
// add a more customizable HTTP Cache API with support for customizable
// read-through caching, freshness lifetime inference, conditional
// request evaluation, automatic revalidation, and more.
//
// Cached items in this API consist of:
//
//   - A cache key: up to 4096 bytes of arbtirary data that identify a
//     cached object.  The cache key may not uniquely identify an item;
//     headers can be used to augment the key when multiple items are
//     associated with the same key.
//
//   - General metadata, such as expiry data (item age, when to expire,
//     and surrogate keys for purging).
//
//   - User-controlled metadata: arbitrary bytes stored alongside the
//     cached object that can be updated when revalidating the cached
//     object.
//
//   - The object itself: arbitrary bytes read via an [io.ReadCloser] and
//     written via a [WriteCloseAbandoner].
//
// In the simplest cases, the top-level [Insert] and [Lookup] functions
// are used for one-off operations on a cached object, and are
// appropriate when request collapsing and revalidation capabilities are
// not required.
//
// The API also supports more complex uses via [Transaction], which can
// collapse concurrent lookups to the same item, including coordinating
// revalidation.
//
// [request collapsing]: https://developer.fastly.com/learning/concepts/request-collapsing/
// [streaming miss]: https://docs.fastly.com/en/guides/streaming-miss
// [revalidation]: https://developer.fastly.com/learning/concepts/stale/
// [surrogate key purging]: https://docs.fastly.com/en/guides/purging-api-cache-with-surrogate-keys
package core

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrNotFound is returned when a cache lookup fails to find a
	// cached object with the provided key.
	ErrNotFound = errors.New("cache: object not found")

	// ErrInvalidArgument is returned when an argument passed to a
	// function is invalid.
	ErrInvalidArgument = errors.New("cache: invalid argument")

	// ErrInvalidOperation is returned when an operation to be performed
	// is not valid given the state of the cached object.
	ErrInvalidOperation = errors.New("cache: invalid operation")

	// ErrLimitExceeded is returned when a cache operation exceeds the
	// limits allowed for this service.
	ErrLimitExceeded = errors.New("cache: operation limit exceeded")

	// ErrUnsupported is returned when a cache operation is not
	// supported.
	ErrUnsupported = errors.New("cache: operation not supported")
)

// Found represents a cached object found by a cache lookup.
type Found struct {
	abiEntry     *fastly.CacheEntry
	state        fastly.CacheLookupState
	userMetadata []byte

	// Key is the cache key used to find this object.
	Key []byte

	// TTL is the time for which the cached object is considered fresh.
	TTL time.Duration

	// Length is the length of the cached object in bytes, if known.
	// The length of the cached item may be unknown if the item is
	// currently being streamed into the cache without a fixed length.
	Length uint64

	// Age is the age of the cached object at lookup time.
	Age time.Duration

	// StaleWhileRevalidate is the period of time that the cached object
	// can be served stale while revalidation takes place.
	//
	// This provides a signal that the cache should be updated (or its
	// contents otherwise revalidated for freshness) asynchronously,
	// while the stale cached object continues to be used, rather than
	// blocking on updating the cached object.  The default
	// stale-while-revalidate period is zero.
	StaleWhileRevalidate time.Duration

	// Hits is the number of times the cached object has been served.
	// This count only reflects the view of the server that supplied the
	// cached object.  Due to clustering, this count may vary between
	// potentially many servers within the data center where the item is
	// cached.  See the clustering documentation for details:
	// https://developer.fastly.com/learning/vcl/clustering/
	Hits uint64

	// Body is an io.ReadCloser for the cached object.  It must be
	// closed when finished.
	Body io.ReadCloser
}

func newFound(key []byte, e *fastly.CacheEntry, state fastly.CacheLookupState, opts LookupOptions) (*Found, error) {
	ttl, err := e.MaxAge()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	length, err := e.Length()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	age, err := e.Age()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	staleWhileRevalidate, err := e.StaleWhileRevalidate()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	hits, err := e.Hits()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	var bopts fastly.CacheGetBodyOptions
	if opts.From > 0 {
		bopts.From(opts.From)
	}
	if opts.To > 0 {
		bopts.To(opts.To)
	}
	body, err := e.Body(bopts)
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	return &Found{
		abiEntry:             e,
		state:                state,
		Key:                  key,
		TTL:                  ttl,
		Length:               length,
		Age:                  age,
		StaleWhileRevalidate: staleWhileRevalidate,
		Hits:                 hits,
		Body:                 &bodyWrapper{body},
	}, nil
}

// Stale returns true if the cached object is stale.
func (f *Found) Stale() bool {
	return f.state&fastly.CacheLookupStateStale != 0
}

// Usable returns true if the cached object is usable.
func (f *Found) Usable() bool {
	return f.state&fastly.CacheLookupStateUsable != 0
}

// UserMetadata returns user-provided metadata associated with the
// cached object.  It will return an empty slice if no metadata was
// provided when the object was inserted.
func (f *Found) UserMetadata() ([]byte, error) {
	if f.userMetadata != nil {
		return f.userMetadata, nil
	}

	userMetadata, err := f.abiEntry.UserMetadata()
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	f.userMetadata = userMetadata
	return userMetadata, nil
}

// GetRange returns an [io.ReadCloser] for the provided range of bytes.
// The Found's Body must be closed before calling this function, or it
// will return [ErrInvalidOperation].
func (f *Found) GetRange(from, to uint64) (io.ReadCloser, error) {
	var bopts fastly.CacheGetBodyOptions
	if from > 0 {
		bopts.From(from)
	}
	if to > 0 {
		bopts.To(to)
	}

	body, err := f.abiEntry.Body(bopts)
	if err := ignoreNoneError(err); err != nil {
		return nil, mapFastlyError(err)
	}

	return &bodyWrapper{body}, nil
}

func requestHeadersBody(h fsthttp.Header) (*fastly.HTTPRequest, error) {
	req, err := fastly.NewHTTPRequest()
	if err != nil {
		return nil, err
	}

	for _, key := range h.Keys() {
		vals := h.Values(key)
		if err := req.SetHeaderValues(key, vals); err != nil {
			return nil, fmt.Errorf("set headers: %w", err)
		}
	}

	return req, nil
}

// LookupOptions control the behavior of cache lookups and the objects
// returned.
type LookupOptions struct {
	// RequestHeaders are a set of HTTP headers that influence cache
	// lookups when Vary rules are set.
	//
	// A lookup will succeed when there is at least one cached object
	// that matches the lookup's cache key, and all of the headers
	// included in the cached object's Vary list match the corresponding
	// headers in that cached object.
	RequestHeaders fsthttp.Header

	// From indicates the starting offset to read from the cached
	// object.
	From uint64

	// To indicates the ending offset to read from the cached object.  A
	// value of 0 means to read to the end of the object.
	To uint64
}

func abiLookupOptions(opts LookupOptions) (fastly.CacheLookupOptions, error) {
	var abiOpts fastly.CacheLookupOptions

	if opts.RequestHeaders != nil {
		req, err := requestHeadersBody(opts.RequestHeaders)
		if err != nil {
			return abiOpts, err
		}

		abiOpts.SetRequest(req)
	}

	return abiOpts, nil
}

// Lookup performs a simple, non-transactional lookup for the given key.
// If the key is not cached, [ErrNotFound] is returned.  Keys can be up
// to 4096 bytes in length.
//
// In contrast to lookups using [NewTransaction], this will not
// coordinate with any concurrent cache lookups.  No request collapsing
// is done.
func Lookup(key []byte, opts LookupOptions) (*Found, error) {
	abiOpts, err := abiLookupOptions(opts)
	if err != nil {
		return nil, err
	}

	e, err := fastly.CacheLookup(key, abiOpts)
	if err != nil {
		return nil, mapFastlyError(err)
	}

	state, err := e.State()
	if err != nil {
		e.Close()
		return nil, mapFastlyError(err)
	}

	if state&fastly.CacheLookupStateFound == 0 {
		e.Close()
		return nil, ErrNotFound
	}

	f, err := newFound(key, e, state, opts)
	if err != nil {
		e.Close()
		return nil, err
	}

	// Note that for Found objects created by this Lookup function, we
	// never close the underlying abiEntry.  This isn't ideal but it
	// makes for a nicer API.  Otherwise we would need to add a Close
	// method for Founds created from this function, whereas Found
	// objects from a transaction wouldn't need to be closed.

	return f, nil
}

// WriteOptions control the behavior of cache inserts and updates.  TTL
// is required, but all other fields are optional.
type WriteOptions struct {
	// TTL is the maximum time the cached object will be considered
	// fresh.  Required.
	TTL time.Duration

	// RequestHeaders are a set of HTTP headers that influence cache
	// lookups when Vary rules are set.
	//
	// This field is only valid for Insert.  If provided for Update,
	// ErrInvalidArgument will be returned.
	RequestHeaders fsthttp.Header

	// Vary is a list of HTTP header names (provided in RequestHeaders)
	// that must match when looking up this key.
	Vary []string

	// InitialAge is the initial age of the cached object.
	InitialAge time.Duration

	// StaleWhileRevalidate is the period of time in which the cached
	// object can be served stale while revalidation is taking place.
	//
	// This provides a signal that the cache should be updated (or its
	// contents otherwise revalidated for freshness) asynchronously,
	// while the stale cached object continues to be used, rather than
	// blocking on updating the cached object.  The methods Usable and
	// Stale can be used to determine the current state of a found item.
	StaleWhileRevalidate time.Duration

	// SurrogateKeys is a list of surrogate keys which can be used to
	// purge this object.
	//
	// Surrogate key purges are the only means to purge specific items
	// from the cache.  At least one surrogate key must be set in order
	// to remove an item without performaing a purge-all, waiting for
	// the item's TTL to elapse, or overwriting the item with Insert.
	//
	// Surrogate keys must contain only printable ASCII characters
	// (those between 0x21 and 0x7E, inclusive).  Any invalid keys will
	// be ignored.
	//
	// See the Fastly surrogate keys guide for details:
	// https://docs.fastly.com/en/guides/purging-api-cache-with-surrogate-keys
	SurrogateKeys []string

	// Length sets the size of the cached object, in bytes, when known
	// prior to actually providing the bytes.
	//
	// It is preferable to provide a length, if possible.  Clients that
	// begin streaming the object's contents before it is completely
	// provided will see the promised length which allows them to, for
	// example, use Content-Length instead of chunked Transfer-Encoding
	// if the item is used as the body of an HTTP request or response.
	Length uint64

	// UserMetadata is abitrary user-provided metadata that will be
	// associated with the cached object.
	UserMetadata []byte

	// SensitiveData indiciates whether to enable PCI/HIPAA-compliant
	// non-volatile caching.
	//
	// See the Fastly PCI-Compliant Caching and Delivery documentation
	// for details:
	// https://docs.fastly.com/products/pci-compliant-caching-and-delivery
	SensitiveData bool
}

func abiWriteOptions(opts WriteOptions) (fastly.CacheWriteOptions, error) {
	var wopts fastly.CacheWriteOptions
	wopts.MaxAge(opts.TTL)

	if len(opts.RequestHeaders) > 0 {
		req, err := requestHeadersBody(opts.RequestHeaders)
		if err != nil {
			return wopts, err
		}

		wopts.SetRequest(req)
	}

	if len(opts.Vary) > 0 {
		wopts.Vary(opts.Vary)
	}

	if opts.InitialAge > 0 {
		wopts.InitialAge(opts.InitialAge)
	}

	if opts.StaleWhileRevalidate > 0 {
		wopts.StaleWhileRevalidate(opts.StaleWhileRevalidate)
	}

	if len(opts.SurrogateKeys) > 0 {
		wopts.SurrogateKeys(opts.SurrogateKeys)
	}

	if opts.Length > 0 {
		wopts.ContentLength(opts.Length)
	}

	if len(opts.UserMetadata) > 0 {
		wopts.UserMetadata(opts.UserMetadata)
	}

	wopts.SensitiveData(opts.SensitiveData)

	return wopts, nil
}

// WriteCloseAbandoner is an interface returned by cache insert
// operations.  It is an [io.WriteCloser] that also supports abandoning
// the stream.  When Abandon is called, the insert operation is canceled
// and content written is not saved into the cache.
type WriteCloseAbandoner interface {
	io.WriteCloser
	Abandon() error
}

// Insert creates a [WriteCloseAbandoner] used for inserting an object
// into the cache for the given key.  For the insertion to complete
// successfully, the object must be fully written to the writer and its
// Close method called.
//
// If Close is not called, or the writer's Abandon method is called, the
// insertion is incomplete and any concurrent lookups that may be
// reading from the object as it is streamed into the cache may
// encounter an error while reading.
//
// Unlike [Transaction.Insert], this does not coordinate with any other
// lookups or inserts for this key.  Concurrent inserts may race with
// concurrent lookups or insertions, and will unconditionally overwrite
// existing cached items rather than allowing for revalidation of an
// existing object.
func Insert(key []byte, opts WriteOptions) (WriteCloseAbandoner, error) {
	wopts, err := abiWriteOptions(opts)
	if err != nil {
		return nil, err
	}

	wca, err := fastly.CacheInsert(key, wopts)
	if err != nil {
		return nil, mapFastlyError(err)
	}

	return wca, nil
}

// Transaction represents a construct to coordinate concurrent actions
// for the same cache key.
//
// Transactions incorporate concepts of [request collapsing] and
// [revalidation], though at a lower level that does not automatically
// interpret HTTP semantics.
//
// # Request collapsing
//
// If there are multiple concurrent calls to [NewTransaction] for
// the same object and that object is not present, only one of the
// callers will be instructed to insert the item into the cache as part
// of the transaction.  The other callers will block until the metadata
// for the object has been inserted, and can then begin streaming its
// contents out of the cache at the same time that the inserting caller
// streams them into the cache.
//
// # Revalidation
//
// Similarly, if an item is usable but stale, and multiple callers
// attempt a lookup concurrently, they will all be given access to the
// stale item, but only one will be designated to perform an update (or
// insertion) to freshen the item in the cache.
//
// [request collapsing]: https://developer.fastly.com/learning/concepts/request-collapsing/
// [revalidation]: https://developer.fastly.com/learning/concepts/stale/
type Transaction struct {
	abiEntry *fastly.CacheEntry
	key      []byte
	state    fastly.CacheLookupState
	opts     LookupOptions
	found    *Found
	ended    bool
}

// NewTransaction creates a new cache transaction for the given key.
//
// Transactions must be closed by calling the [Transaction.Close] method
// when finished.
//
// Keys can be up to 4096 bytes in length.
func NewTransaction(key []byte, opts LookupOptions) (*Transaction, error) {
	abiOpts, err := abiLookupOptions(opts)
	if err != nil {
		return nil, err
	}

	e, err := fastly.CacheTransactionLookup(key, abiOpts)
	if err != nil {
		return nil, mapFastlyError(err)
	}

	// Concurrent lookups for the same key will block on this call.
	state, err := e.State()
	if err != nil {
		e.Close()
		return nil, mapFastlyError(err)
	}

	return &Transaction{
		abiEntry: e,
		key:      key,
		state:    state,
		opts:     opts,
	}, nil
}

// Found returns information about the found cached object, if one is
// available.  If there is no cached object, [ErrNotFound] is returned.
//
// Even if an object is found, the cache item might be stale and require
// updating.  Use [Transaction.MustInsertOrUpdate] to determine whether
// this transaction client is expected to update the cached object.
func (t *Transaction) Found() (*Found, error) {
	if t.state&fastly.CacheLookupStateFound == 0 {
		return nil, ErrNotFound
	}

	if t.found == nil {
		f, err := newFound(t.key, t.abiEntry, t.state, t.opts)
		if err != nil {
			return nil, mapFastlyError(err)
		}
		t.found = f
	}

	return t.found, nil
}

// MustInsert returns true if a usable cached item was not found, and
// this transaction client is expected to insert one.
//
// This function will return false if any cached item was found, even if
// stale.  Use [Transaction.MustInsertOrUpdate] instead to handle stale
// items.
//
// Use [Transaction.Insert] to insert the object, or
// [Transaction.Cancel] to exit the transaction without providing an
// object.
func (t *Transaction) MustInsert() bool {
	return t.state&fastly.CacheLookupStateFound == 0 && t.state&fastly.CacheLookupStateMustInsertOrUpdate != 0
}

// MustInsertOrUpdate returns true if a fresh cached item was not found,
// and this transaction client is expected to insert a new item or
// update a stale item.
//
// A fresh cached item not being found could mean one of two things:
//   - No cached item was found, or
//   - A stale cached item was found.
//
// Use [Transaction.MustInsert] or [Transaction.Found] to determine
// whether a new cached item must be inserted.
//
// Use:
//   - [Transaction.Update] to freshen a found item by updating its metadata,
//   - [Transaction.Insert] to insert a new item including object data,
//   - [Transaction.Cancel] to exit the transaction without providing an object.
func (t *Transaction) MustInsertOrUpdate() bool {
	return t.state&fastly.CacheLookupStateMustInsertOrUpdate != 0
}

// Insert creates a [WriteCloseAbandoner] used for inserting an object
// into the cache for this transaction's key.  For the insertion to
// complete successfully, the object must be fully written to the writer
// and its Close method called.
//
// If Close is not called, or the writer's Abandon method is called,
// the insertion is incomplete and any concurrent lookups that may be
// reading from the object as it is streamed into the cache may
// encounter an error while reading.
//
// Inserting an object into the cache will unblock other transactions
// waiting on this object, streaming the contents to clients as the data
// is written.
func (t *Transaction) Insert(opts WriteOptions) (WriteCloseAbandoner, error) {
	wopts, err := abiWriteOptions(opts)
	if err != nil {
		return nil, err
	}

	wca, err := t.abiEntry.Insert(wopts)
	if err != nil {
		return nil, mapFastlyError(err)
	}

	return wca, nil
}

// InsertAndStreamBack creates a [WriteCloseAbandoner] used for
// inserting an object into the cache, as well as a new [Found] that can
// be used to stream the object back to the caller.
//
// Inserting an object into the cache will unblock other transactions
// waiting on this object, streaming the contents to clients as the data
// is written.
//
// The returned [Found] allows the client inserting a cache object to
// efficiently read back the contents of that item, avoiding the need to
// buffer contents for copying to multiple destinations.  This pattern
// is commonly required when caching an item that also must be provided
// to, for example, the client response.
func (t *Transaction) InsertAndStreamBack(opts WriteOptions) (WriteCloseAbandoner, *Found, error) {
	wopts, err := abiWriteOptions(opts)
	if err != nil {
		return nil, nil, err
	}

	writeBody, abiEntry, err := t.abiEntry.InsertAndStreamBack(wopts)
	if err != nil {
		return nil, nil, mapFastlyError(err)
	}

	newState, err := abiEntry.State()
	if err != nil {
		writeBody.Abandon()
		abiEntry.Close()
		return nil, nil, mapFastlyError(err)
	}

	f, err := newFound(t.key, abiEntry, newState, LookupOptions{})
	if err != nil {
		writeBody.Abandon()
		abiEntry.Close()
		return nil, nil, err
	}

	return writeBody, f, nil
}

// Update is used to update a stale object in the cache.
//
// Updating an object freshens it by updating its metadata without
// changing the object itself.
//
// This method should only be called when
// [Transaction.MustInsertOrUpdate] is true and the item is found.
// Otherwise, an [ErrInvalidOperation] will be returned.
//
// NOTE: Updating a cached item will replace ALL of the configuration in
// the underlying cache object.  If something is not set in the provided
// [WriteOptions], it will revert to the default value.  This behavior
// is likely to change in the future to use the existing configuration.
// This change will be noted in a future changelog.
//
// The provided write options must not include request headers, and this
// will return [ErrInvalidArgument] if they do.
//
// This method will return [ErrInvalidOperation] if the object is not
// stale.
func (t *Transaction) Update(opts WriteOptions) error {
	wopts, err := abiWriteOptions(opts)
	if err != nil {
		return err
	}

	err = t.abiEntry.Update(wopts)
	return mapFastlyError(err)
}

// Cancel terminates the obligation to provide an object to the cache.
//
// If there are concurrent transactional lookups that were blocked
// waiting on this client to provide the item, one of them will be
// chosen to be unblocked and given the [Transaction.MustInsertOrUpdate]
// obligation.
//
// Cancel does not close the transaction.  Callers must still call Close
// to end the transaction and cleanup resources associated with it.
func (t *Transaction) Cancel() error {
	return mapFastlyError(t.abiEntry.Cancel())
}

// Close ends the transaction, commits any inserts or updates, and
// cleans up resources associated with it.
//
// If a Found is associated with this transaction, its Body will be
// closed if it hasn't been already.
func (t *Transaction) Close() error {
	if t.ended {
		return nil
	}
	t.ended = true

	if t.found != nil {
		t.found.Body.Close()
	}
	return t.abiEntry.Close()
}

type bodyWrapper struct {
	abiBody *fastly.HTTPBody
}

func (b *bodyWrapper) Read(p []byte) (int, error) {
	if b.abiBody == nil {
		return 0, io.EOF
	}

	return b.abiBody.Read(p)
}

func (b *bodyWrapper) Close() error {
	if b.abiBody == nil {
		return nil
	}

	// TODO(joeshaw): currently bodies must be fully drained before they're closed
	io.Copy(io.Discard, b.abiBody)

	err := b.abiBody.Close()
	b.abiBody = nil
	return err
}

func mapFastlyError(err error) error {
	status, ok := fastly.IsFastlyError(err)
	if !ok {
		return err
	}

	switch status {
	case fastly.FastlyStatusNone:
		return ErrNotFound
	case fastly.FastlyStatusInval:
		return ErrInvalidArgument
	case fastly.FastlyStatusLimitExceeded:
		return ErrLimitExceeded
	case fastly.FastlyStatusBadf:
		return ErrInvalidOperation
	case fastly.FastlyStatusUnsupported:
		return ErrUnsupported
	default:
		return err
	}
}

func ignoreNoneError(err error) error {
	status, ok := fastly.IsFastlyError(err)
	if ok && status == fastly.FastlyStatusNone {
		return nil
	}
	return err
}
