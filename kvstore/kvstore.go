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
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var ErrKeyNotFound = errors.New("kvstore: key not found")

// Entry represents a KV store value.
//
// It embeds an [io.Reader] which holds the contents of the value, and
// can be passed to other functions.
//
// For smaller values, a [String] method is provided to consume the
// contents of the underlying reader and return a string.
//
// Do not mix-and-match these approaches: use either the [io.Reader] or
// the [String] method, not both.
type Entry struct {
	io.Reader

	validString bool
	s           string
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

// Store represents a Fastly KV store
type Store struct {
	kvstore *fastly.KVStore
}

// Open returns a handle to the named kv store
func Open(name string) (*Store, error) {
	o, err := fastly.OpenKVStore(name)
	if err != nil {
		return nil, err
	}

	return &Store{kvstore: o}, nil
}

// Lookup fetches a key from the associated KV store.  If the key does not
// exist, Lookup returns the sentinel error [ErrKeyNotFound].
func (s *Store) Lookup(key string) (*Entry, error) {
	val, err := s.kvstore.Lookup(key)
	if err != nil {

		// turn FastlyStatusNone into NotFound
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusNone {
			return nil, ErrKeyNotFound
		}

		return nil, err
	}

	return &Entry{Reader: val}, err
}

// Insert adds a key to the associated KV store.
func (s *Store) Insert(key string, value io.Reader) error {
	err := s.kvstore.Insert(key, value)
	return err
}
