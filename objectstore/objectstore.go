package objectstore

import (
	"errors"
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var ErrKeyNotFound = errors.New("objectstore: key not found")

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

	e.validString = true

	return string(b)

}

// Store represents a Fastly object store
type Store struct {
	objectstore *fastly.ObjectStore
}

// Open returns a handle to the named object store
func Open(name string) (*Store, error) {
	o, err := fastly.OpenObjectStore(name)
	if err != nil {
		return nil, err
	}

	return &Store{objectstore: o}, nil
}

// Lookup fetches a key from the associated object store.  If the key does not
// exist, Lookup returns the sentinel error ErrKeyNotFound.
func (s *Store) Lookup(key string) (*Entry, error) {
	val, err := s.objectstore.Lookup(key)
	if err != nil {

		// turn FastlyStatusNone into NotFound
		if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusNone {
			return nil, ErrKeyNotFound
		}

		return nil, err
	}

	return &Entry{Reader: val}, err
}

// Insert adds a key to the associated object store.
func (s *Store) Insert(key string, value io.Reader) error {
	err := s.objectstore.Insert(key, value)
	return err
}
