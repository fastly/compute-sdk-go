package objectstore

import (
	"github.com/fastly/compute-sdk-go/kvstore"
)

var ErrKeyNotFound = kvstore.ErrKeyNotFound

// Deprecated: Use the kvstore package instead.
type Entry = kvstore.Entry

// Store represents a Fastly object store
//
// Deprecated: Use the kvstore package instead.
type Store = kvstore.Store

// Open returns a handle to the named object store
//
// Deprecated: Use kvstore.Open() instead.
func Open(name string) (*Store, error) {
	return kvstore.Open(name)
}
