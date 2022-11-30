package secretstore

import (
	"errors"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrSecretStoreNotFound indicates that the named secret store
	// doesn't exist.
	ErrSecretStoreNotFound = errors.New("secret store not found")

	// ErrSecretNotFound indicates that the named secret doesn't exist
	// within this store.
	ErrSecretNotFound = errors.New("secret not found")
)

// Store represents a Fastly Secret Store
type Store struct {
	st *fastly.SecretStore
}

// Secret represents a secret in a store
type Secret struct {
	s *fastly.Secret
}

// Open returns a handle to the named secret store, if it exists.  It
// will return [ErrSecretStoreNotFound] if it doesn't exist.
func Open(name string) (*Store, error) {
	st, err := fastly.OpenSecretStore(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		if ok && status == fastly.FastlyStatusNone {
			return nil, ErrSecretStoreNotFound
		}
		return nil, err
	}

	return &Store{st: st}, nil
}

// Get returns a handle to the named secret within the store, if it
// exists.  It will return [ErrSecretNotFound] if it doesn't exist.
func (st *Store) Get(name string) (*Secret, error) {
	s, err := st.st.Get(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		if ok && status == fastly.FastlyStatusNone {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}

	return &Secret{s: s}, nil
}

// Plaintext decrypts and returns the secret value as a byte slice.
func (s *Secret) Plaintext() ([]byte, error) {
	return s.s.Plaintext()
}
