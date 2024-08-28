// Copyright 2022 Fastly, Inc.

package configstore

import (
	"errors"
	"fmt"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrStoreNotFound indicates the named config store doesn't exist.
	ErrStoreNotFound = errors.New("config store not found")

	// ErrStoreNameEmpty indicates the given config store name
	// was empty.
	ErrStoreNameEmpty = errors.New("config store name was empty")

	// ErrStoreNameInvalid indicates the given config store name
	// was invalid.
	ErrStoreNameInvalid = errors.New("config store name contained invalid characters")

	// ErrStoreNameTooLong indicates the given config store name
	// was too long.
	ErrStoreNameTooLong = errors.New("config store name too long")

	// ErrKeyNotFound indicates a key isn't in a config store.
	ErrKeyNotFound = errors.New("key not found")

	// ErrUnexpected indicates an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
)

// Store is a read-only representation of a config store.
type Store struct {
	abiDict *fastly.ConfigStore
}

// Open returns a config store with the given name. Names are case
// sensitive.
func Open(name string) (*Store, error) {
	d, err := fastly.OpenConfigStore(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return nil, ErrStoreNotFound
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrStoreNameEmpty
		case ok && status == fastly.FastlyStatusUnsupported:
			return nil, ErrStoreNameTooLong
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrStoreNameInvalid
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}
	return &Store{d}, nil
}

// Has returns true if the key exists in the config store, without allocating
// space to read a value.
func (s *Store) Has(key string) (bool, error) {
	if s == nil {
		return false, ErrKeyNotFound
	}

	v, err := s.abiDict.Has(key)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return false, ErrStoreNotFound
		case ok && status == fastly.FastlyStatusNone:
			return false, ErrKeyNotFound
		case ok:
			return false, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return false, err
		}
	}

	return v, nil
}

// GetBytes returns the value in the config store for the given key, if it exists, as a byte slice.
func (s *Store) GetBytes(key string) ([]byte, error) {
	if s == nil {
		return nil, ErrKeyNotFound
	}

	v, err := s.abiDict.GetBytes(key)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return nil, ErrStoreNotFound
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrKeyNotFound
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}
	return v, nil
}

// Get returns the value in the config store with the given key, if it exists.
func (s *Store) Get(key string) (string, error) {
	buf, err := s.GetBytes(key)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
