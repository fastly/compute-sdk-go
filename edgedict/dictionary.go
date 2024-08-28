// Copyright 2022 Fastly, Inc.

package edgedict

import (
	"errors"
	"fmt"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrDictionaryNotFound indicates the named dictionary doesn't exist.
	ErrDictionaryNotFound = errors.New("dictionary not found")

	// ErrDictionaryNameEmpty indicates the given dictionary name
	// was empty.
	ErrDictionaryNameEmpty = errors.New("dictionary name was empty")

	// ErrDictionaryNameInvalid indicates the given dictionary name
	// was invalid.
	ErrDictionaryNameInvalid = errors.New("dictionary name contained invalid characters")

	// ErrDictionaryNameTooLong indicates the given dictionary name
	// was too long.
	ErrDictionaryNameTooLong = errors.New("dictionary name too long")

	// ErrKeyNotFound indicates a key isn't in a dictionary.
	ErrKeyNotFound = errors.New("key not found")

	// ErrUnexpected indicates an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
)

// Dictionary is a read-only representation of an edge dictionary.
//
// Deprecated: Use the configstore package instead.
type Dictionary struct {
	abiDict *fastly.Dictionary
}

// Open returns an edge dictionary with the given name. Names are case
// sensitive.
//
// Deprecated: Use configstore.Open() instead.
func Open(name string) (*Dictionary, error) {
	d, err := fastly.OpenDictionary(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return nil, ErrDictionaryNotFound
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrDictionaryNameEmpty
		case ok && status == fastly.FastlyStatusUnsupported:
			return nil, ErrDictionaryNameTooLong
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrDictionaryNameInvalid
		default:
			return nil, err
		}
	}
	return &Dictionary{d}, nil
}

// GetBytes returns the value in the dictionary for the given key, if it exists, as a byte slice.
func (d *Dictionary) GetBytes(key string) ([]byte, error) {
	if d == nil {
		return nil, ErrKeyNotFound
	}

	v, err := d.abiDict.GetBytes(key)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return nil, ErrDictionaryNotFound
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

// Get returns the value in the dictionary with the given key, if it exists.
func (d *Dictionary) Get(key string) (string, error) {
	buf, err := d.GetBytes(key)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Has returns true if the key exists in the dictionary, without allocating
// space to read a value.
func (d *Dictionary) Has(key string) (bool, error) {
	if d == nil {
		return false, ErrKeyNotFound
	}

	v, err := d.abiDict.Has(key)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return false, ErrDictionaryNotFound
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
