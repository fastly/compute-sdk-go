// Copyright 2022 Fastly, Inc.

package edgedict

import (
	"errors"

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
)

// Dictionary is a read-only representation of an edge dictionary.
type Dictionary struct {
	abiDict *fastly.Dictionary
}

// Open returns an edge dictionary with the given name. Names are case
// sensitive.
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

// Get returns the item in the dictionary with the given key.
func (d *Dictionary) Get(key string) (string, error) {
	if d == nil {
		return "", ErrKeyNotFound
	}

	s, err := d.abiDict.Get(key)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusBadf:
			return "", ErrDictionaryNotFound
		case ok && status == fastly.FastlyStatusNone:
			return "", ErrKeyNotFound
		default:
			return "", err
		}
	}

	return s, nil
}
