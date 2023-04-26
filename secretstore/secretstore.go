// Package secretstore provides a read-only interface to Fastly
// Compute@Edge Secret Stores.
//
// Secret stores are persistent, globally distributed stores for
// secrets.  Secrets are decrypted as-needed at the edge.
//
// See the [Fastly Secret Store documentation] for details.
//
// [Fastly Secret Store documentation]: https://developer.fastly.com/learning/concepts/dynamic-config/#secret-stores
package secretstore

import (
	"errors"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrSecretStoreNotFound indicates that the named secret store
	// doesn't exist.
	ErrSecretStoreNotFound = errors.New("secret store not found")

	// ErrInvalidSecretStoreName indicates that the given secret store
	// name is invalid.
	ErrInvalidSecretStoreName = errors.New("invalid secret store name")

	// ErrSecretNotFound indicates that the named secret doesn't exist
	// within this store.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrInvalidSecretName indicates that the given secret name is
	// invalid.
	ErrInvalidSecretName = errors.New("invalid secret name")

	// ErrUnexpected indicates than an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
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
		switch {
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrSecretStoreNotFound
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrInvalidSecretStoreName
		case ok:
			return nil, ErrUnexpected
		default:
			return nil, err
		}
	}

	return &Store{st: st}, nil
}

// Get returns a handle to the named secret within the store, if it
// exists.  It will return [ErrSecretNotFound] if it doesn't exist.
func (st *Store) Get(name string) (*Secret, error) {
	s, err := st.st.Get(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrSecretNotFound
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrInvalidSecretName
		case ok:
			return nil, ErrUnexpected
		default:
			return nil, err
		}
	}

	return &Secret{s: s}, nil
}

// Plaintext decrypts and returns the secret value as a byte slice.
func (s *Secret) Plaintext() ([]byte, error) {
	plaintext, err := s.s.Plaintext()
	if err != nil {
		_, ok := fastly.IsFastlyError(err)
		if ok {
			return nil, ErrUnexpected
		}
		return nil, err
	}
	return plaintext, nil
}
