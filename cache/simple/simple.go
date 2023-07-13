// Package simple provides the Simple Cache API, a simplified interface
// to inserting and retrieving entries from Fastly's cache.
//
// Cache operations are local to the Fastly POP serving the request.
// Purging can also be performed globally.
//
// For more advanced uses, see the Core Cache API in the
// [github.com/fastly/compute-sdk-go/cache/core] package.
package simple

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/cache/core"
	"github.com/fastly/compute-sdk-go/purge"
)

// Get retrieves the object stored in the cache for the given key.  If
// the key is not cached, [core.ErrNotFound] is returned.  Keys can be
// up to 4096 bytes in length.
//
// The returned [io.ReadCloser] must be closed by the caller when
// finished.
func Get(key []byte) (io.ReadCloser, error) {
	f, err := core.Lookup(key, core.LookupOptions{})
	if err != nil {
		return nil, err
	}
	return f.Body, nil
}

// CacheEntry contains the contents and TTL (time-to-live) for an item
// to be added to the cache via [GetOrSet] or [GetOrSetContents].
type CacheEntry struct {
	// The contents of the cached object.
	Body io.Reader

	// The time-to-live for the cached object.
	TTL time.Duration
}

// GetOrSet retrieves the object stored in the cache for the given key
// if it exists, or inserts and returns the contents by running the
// provided setFn function.
//
// The setFn function is only run when no value is present for the key,
// and no other client is in the process of setting it.  The function
// should return a populated [CacheEntry] or an error.
//
// If the setFn function returns an error, nothing will be saved to the
// cache and the error will be returned from the GetOrSet function.
// Other concurrent readers will also see an error while reading.
//
// The returned [io.ReadCloser] must be closed by the caller when
// finished.
func GetOrSet(key []byte, setFn func() (CacheEntry, error)) (io.ReadCloser, error) {
	tx, err := core.NewTransaction(key, core.LookupOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Close()

	if tx.MustInsertOrUpdate() {
		e, err := setFn()
		if err != nil {
			return nil, err
		}

		w, f, err := tx.InsertAndStreamBack(core.WriteOptions{
			TTL: e.TTL,
			SurrogateKeys: []string{
				SurrogateKeyForCacheKey(key, PurgeScopePOP),
				SurrogateKeyForCacheKey(key, PurgeScopeGlobal),
			},
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		if _, err := io.Copy(w, e.Body); err != nil {
			w.Abandon()
			return nil, err
		}

		if err := w.Close(); err != nil {
			w.Abandon()
			return nil, err
		}

		return f.Body, nil
	}

	f, err := tx.Found()
	if err != nil {
		return nil, err
	}

	return f.Body, nil
}

// GetOrSetEntry retrieves the object stored in the cache for the given
// key if it exists, or inserts and returns the contents provided in the
// [CacheEntry].
//
// The cache entry is only inserted when no value is present for the
// key, and no other client is in the process of setting it.
//
// If the cache entry body content is costly to compute, consider using
// [GetOrSet] instead to avoid creating its [io.Reader] in the case
// where the value is already present.
//
// The returned [io.ReadCloser] must be closed by the caller when
// finished.
func GetOrSetEntry(key []byte, entry CacheEntry) (io.ReadCloser, error) {
	return GetOrSet(key, func() (CacheEntry, error) {
		return entry, nil
	})
}

// PurgeScope controls the scope of a purge operation.  It is used in
// the [PurgeOptions] struct.
type PurgeScope uint32

const (
	// PurgeScopePOP purges the entry only from the local POP cache.
	PurgeScopePOP PurgeScope = iota
	// PurgeScopeGlobal purges the entry from all POP caches.
	PurgeScopeGlobal
)

// PurgeOptions controls the behavior of the [Purge] function.
type PurgeOptions struct {
	Scope PurgeScope
}

// Purge removes the entry associated with the given cache key, if one
// exists.
//
// The scope of the purge can be controlled with the PurgeOptions.
//
// Purges are handled asynchronously, and the cached object may persist
// in cache for a short time (~150ms or less) after this function
// returns.
func Purge(key []byte, opts PurgeOptions) error {
	sk := SurrogateKeyForCacheKey(key, opts.Scope)
	return purge.PurgeSurrogateKey(sk, purge.PurgeOptions{})
}

// SurrogateKeyForCacheKey creates a surrogate key for the given cache
// key and purge scope that is compatible with the Simple Cache API.
// Each cache entry for the Simple Cache API is configured with
// surrogate keys from this function.
//
// This function is provided as a convenience for implementors wishing
// to add a Simple Cache-compatible surrogate key manually via the Core
// Cache API ([github.com/fastly/compute-sdk-go/cache/core]) for
// interoperability with [Purge].
func SurrogateKeyForCacheKey(cacheKey []byte, scope PurgeScope) string {
	// The values are SHA-256 digests of the cache key (plus the local POP
	// for the local surrogate key), converted to uppercase hexadecimal.
	// This scheme must be kept consistent across all compute SDKs.
	h := sha256.New()
	h.Write(cacheKey)
	if scope == PurgeScopePOP {
		h.Write([]byte(os.Getenv("FASTLY_POP")))
	}

	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}
