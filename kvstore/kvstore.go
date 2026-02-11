// Package kvstore provides access to Fastly KV stores.
//
// KV stores provide durable storage of key/value data that is readable
// and writable at the edge and synchronized globally.
//
// See the [Fastly KV store documentation] for details.
//
// [Fastly KV store documentation]: https://developer.fastly.com/learning/concepts/data-stores/#kv-stores
package kvstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrStoreNotFound indicates that the named store doesn't exist.
	ErrStoreNotFound = errors.New("kvstore: store not found")

	// ErrKeyNotFound indicates that the named key doesn't exist in this
	// KV store.
	ErrKeyNotFound = errors.New("kvstore: key not found")

	// ErrInvalidKey indicates that the given key is invalid.
	ErrInvalidKey = errors.New("kvstore: invalid key")

	// ErrTooManyRequests is returned when inserting a value exceeds the
	// rate limit.
	ErrTooManyRequests = errors.New("kvstore: too many requests")

	// ErrInvalidOptions indicates the options provided for this operation were invalid.
	ErrInvalidOptions = errors.New("kvstore: invalid options")

	// ErrBadRequest indicates the KV Store request was bad.
	ErrBadRequest = errors.New("kvstore: bad request")

	// ErrPreconditionFailed indicates a precondition for the kvstore operation failed.
	ErrPreconditionFailed = errors.New("kvstore: precondition failed")

	// ErrPayloadTooLarge indicates the item exceeded the payload limit.
	ErrPayloadTooLarge = errors.New("kvstore: payload too large")

	// ErrUnexpected indicates than an unexpected error occurred.
	ErrUnexpected = errors.New("kvstore: unexpected error")
)

// Entry represents a KV store value.
//
// It embeds an [io.Reader] which holds the contents of the value, and
// can be passed to functions that accept an [io.Reader].
//
// For smaller values, an [Entry.String] method is provided to consume the
// contents of the underlying reader and return a string.
//
// Do not mix-and-match these approaches: use either the [io.Reader] or
// the [Entry.String] method, not both.
type Entry struct {
	io.Reader

	validString bool
	s           string

	meta       []byte
	generation uint64
}

// String consumes the entire contents of the Entry and returns it as a
// string.
//
// Take care when using this method, as large values might exceed the
// per-request memory limit.
func (e *Entry) String() string {
	if e.validString {
		return e.s
	}

	// TODO(dgryski): replace with StringBuilder + io.Copy ?
	b, err := io.ReadAll(e)
	if err != nil {
		return ""
	}

	e.s = string(b)
	e.validString = true
	return e.s
}

func (e *Entry) Meta() []byte {
	return e.meta
}

func (e *Entry) Generation() uint64 {
	return e.generation
}

// Store represents a Fastly KV store
type Store struct {
	kvstore *fastly.KVStore
}

// Open returns a handle to the named kv store
func Open(name string) (*Store, error) {
	kv, err := fastly.OpenKVStore(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrStoreNotFound
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}

	return &Store{kvstore: kv}, nil
}

// Lookup fetches a key from the associated KV store.  If the key does not
// exist, Lookup returns the sentinel error [ErrKeyNotFound].
func (s *Store) Lookup(key string) (*Entry, error) {
	h, err := s.kvstore.Lookup(key)
	if err != nil {
		return nil, mapFastlyErr(err)
	}

	result, err := s.kvstore.LookupWait(h)
	if err != nil {
		return nil, mapFastlyErr(err)
	}

	return &Entry{Reader: result.Body, meta: result.Meta, generation: result.Generation}, nil
}

// Insert adds a key to the associated KV store.
func (s *Store) Insert(key string, value io.Reader) error {
	return s.InsertWithConfig(key, value, nil)
}

type InsertMode = fastly.KVInsertMode

const (
	InsertModeOverwrite = fastly.KVInsertModeOverwrite
	InsertModeAdd       = fastly.KVInsertModeAdd
	InsertModeAppend    = fastly.KVInsertModeAppend
	InsertModePrepend   = fastly.KVInsertModePrepend
)

type InsertConfig struct {
	Mode            InsertMode
	BackgroundFetch bool
	Metadata        []byte
	TTLSec          uint32
}

// Insert adds a key to the associated KV store.
func (s *Store) InsertWithConfig(key string, value io.Reader, config *InsertConfig) error {
	var abiConf fastly.KVInsertConfig
	if config != nil {
		abiConf.Mode(config.Mode)
		if config.BackgroundFetch {
			abiConf.BackgroundFetch()
		}
		if config.Metadata != nil {
			abiConf.Metadata(config.Metadata)
		}
		if config.TTLSec != 0 {
			abiConf.TTLSec(config.TTLSec)
		}
	}

	h, err := s.kvstore.Insert(key, value, &abiConf)
	if err != nil {
		return mapFastlyErr(err)
	}

	err = s.kvstore.InsertWait(h)
	if err != nil {
		return mapFastlyErr(err)
	}
	return nil
}

// Delete removes a key from the associated KV store.
func (s *Store) Delete(key string) error {
	h, err := s.kvstore.Delete(key)
	if err != nil {
		return mapFastlyErr(err)
	}

	err = s.kvstore.DeleteWait(h)
	if err != nil {
		return mapFastlyErr(err)
	}
	return nil
}

type ListConsistency = fastly.KVListMode

const (
	ListModeStrong   = fastly.KVListModeStrong
	ListModeEventual = fastly.KVListModeEventual
)

var consistencyStrings = [...]string{
	ListModeStrong:   "strong",
	ListModeEventual: "eventual",
}

func consistencyString(m ListConsistency) string {
	if int(m) < len(consistencyStrings) {
		return consistencyStrings[m]
	}
	return "unknown"
}

func consistencyMode(m string) ListConsistency {
	switch m {
	case "strong":
		return ListModeStrong
	case "eventual":
		return ListModeEventual
	}
	return ListModeStrong
}

// ListConfig holds the option for the List operation.
type ListConfig struct {
	// Mode is the consistency for the list operation

	Mode ListConsistency

	// Limit is the number of results per page.
	Limit uint32

	// Prefix is the key prefix to list.
	Prefix string

	// Cursor is the internal list operation cursor.
	Cursor string
}

// ListIter is an iterator over pages of List results.
type ListIter struct {
	kvstore *fastly.KVStore
	page    ListPage
	err     error
}

// Err returns any error encountered during the iteration.
func (it *ListIter) Err() error {
	return it.err
}

// Next advances the list iterator to the next page of results.  Returns false when the iteration is complete.
func (it *ListIter) Next() bool {
	if it.err != nil {
		return false
	}

	if it.page.Meta.NextCursor == "" && len(it.page.Data) != 0 {
		// end of iteration
		return false
	}

	var abiConf fastly.KVListConfig

	if it.page.Meta.Mode != "" {
		abiConf.Mode(consistencyMode(it.page.Meta.Mode))
	}
	if it.page.Meta.Limit != 0 {
		abiConf.Limit(it.page.Meta.Limit)
	}
	if it.page.Meta.Prefix != "" {
		abiConf.Prefix(it.page.Meta.Prefix)
	}
	if it.page.Meta.NextCursor != "" {
		abiConf.Cursor(it.page.Meta.NextCursor)
	}

	h, err := it.kvstore.List(&abiConf)
	if err != nil {
		it.err = mapFastlyErr(err)
		return false
	}

	body, err := it.kvstore.ListWait(h)
	if err != nil {
		it.err = mapFastlyErr(err)
		return false
	}

	buf, err := io.ReadAll(body)
	if err != nil {
		it.err = err
		return false
	}

	var p ListPage
	if err := json.Unmarshal(buf, &p); err != nil {
		it.err = err
		return false
	}

	if len(p.Data) == 0 {
		return false
	}

	it.page = p

	return true
}

type ListPage struct {
	// Data is the list of keys returned for this page.
	Data []string `json:"data"`

	// Meta is the metadata assocaited with this page of results.
	Meta ListMetadata `json:"meta"`
}

// Page returns the current page of results.
func (it *ListIter) Page() ListPage {
	return it.page
}

// ListMetadata is the metadata for a particular page of list results.[
type ListMetadata struct {
	Limit      uint32 `json:"limit"`
	NextCursor string `json:"next_cursor"`
	Prefix     string `json:"prefix"`
	Mode       string `json:"mode"`
}

// List returns an iterator over pages of keys matching
func (s *Store) List(config *ListConfig) *ListIter {
	if config == nil {
		config = &ListConfig{}
	}

	return &ListIter{
		kvstore: s.kvstore,
		page: ListPage{
			Meta: ListMetadata{
				Limit:      config.Limit,
				NextCursor: config.Cursor,
				Prefix:     config.Prefix,
				Mode:       consistencyString(config.Mode),
			},
		},
	}
}

var kvErrToErr = [...]error{
	// We really shouldn't be returning these
	fastly.KVErrorUninitialized: ErrUnexpected,
	fastly.KVErrorOK:            ErrUnexpected,

	// Mapping internal KVErrors to kvstore package-level errors.
	fastly.KVErrorBadRequest:         ErrBadRequest,
	fastly.KVErrorNotFound:           ErrKeyNotFound,
	fastly.KVErrorPreconditionFailed: ErrPreconditionFailed,
	fastly.KVErrorPayloadTooLarge:    ErrPayloadTooLarge,
	fastly.KVErrorInternalError:      ErrUnexpected,
	fastly.KVErrorTooManyRequests:    ErrTooManyRequests,
}

func mapFastlyErr(err error) error {

	// Is it a kvstore-specific error?
	if kvErr, ok := err.(fastly.KVError); ok {
		if kvErr <= fastly.KVErrorTooManyRequests {
			return kvErrToErr[kvErr]
		}
		return fmt.Errorf("%w (%s)", ErrUnexpected, err)
	}

	// Maybe it was a fastly error?
	status, ok := fastly.IsFastlyError(err)
	switch {
	case ok && status == fastly.FastlyStatusBadf:
		return ErrStoreNotFound
	case ok && status == fastly.FastlyStatusInval:
		return ErrInvalidKey
	case ok:
		return fmt.Errorf("%w (%s)", ErrUnexpected, status)
	}

	// No idea; just return what we have.

	return err
}
