// Package kvstore provides access to Fastly KV stores.
//
// KV stores provide durable storage of key/value data that is readable
// and writable at the edge and synchronized globally.
//
// See the [Fastly KV store documetnation] for details.
//
// [Fastly KV store documentation]: https://developer.fastly.com/learning/concepts/data-stores/#kv-stores
package kvstore

import (
	"errors"
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var ErrKeyNotFound = errors.New("kvstore: key not found")

type Entry struct {
	io.Reader

	validString bool
	s           string
}

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

// Store represents a Fastly kv store
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

// Lookup fetches a key from the associated kv store.  If the key does not
// exist, Lookup returns the sentinel error ErrKeyNotFound.
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

// Insert adds a key to the associated kv store.
func (s *Store) Insert(key string, value io.Reader) error {
	err := s.kvstore.Insert(key, value)
	return err
}
