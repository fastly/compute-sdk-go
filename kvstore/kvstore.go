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
	generation uint32
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

func (e *Entry) Generation() uint32 {
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
	h, err := s.kvstore.Insert(key, value)
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
