// Copyright 2022 Fastly, Inc.

package edgedict

import (
	"github.com/fastly/compute-sdk-go/configstore"
)

var (
	// ErrDictionaryNotFound indicates the named dictionary doesn't exist.
	ErrDictionaryNotFound = configstore.ErrStoreNotFound

	// ErrDictionaryNameEmpty indicates the given dictionary name
	// was empty.
	ErrDictionaryNameEmpty = configstore.ErrStoreNameEmpty

	// ErrDictionaryNameInvalid indicates the given dictionary name
	// was invalid.
	ErrDictionaryNameInvalid = configstore.ErrStoreNameInvalid

	// ErrDictionaryNameTooLong indicates the given dictionary name
	// was too long.
	ErrDictionaryNameTooLong = configstore.ErrStoreNameTooLong

	// ErrKeyNotFound indicates a key isn't in a dictionary.
	ErrKeyNotFound = configstore.ErrKeyNotFound
)

// Dictionary is a read-only representation of an edge dictionary.
//
// Deprecated: Use the configstore package instead.
type Dictionary = configstore.Store

// Open returns an edge dictionary with the given name. Names are case
// sensitive.
//
// Deprecated: Use configstore.Open() instead.
func Open(name string) (*Dictionary, error) {
	return configstore.Open(name)
}
