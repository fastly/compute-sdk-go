package core_test

import (
	"fmt"
	"io"
	"time"

	"github.com/fastly/compute-sdk-go/cache/core"
)

func ExampleLookup() {
	// f is a core.Found value, representing a found cache item.
	// core.ErrNotFound is returned if the item is not cached.
	f, err := core.Lookup([]byte("my_key"), core.LookupOptions{})
	if err != nil {
		panic(err)
	}
	defer f.Body.Close()

	// The contents of the cached item are in the Found's Body field.
	cachedStr, err := io.ReadAll(f.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("The cached value was: %s", cachedStr)
}

func ExampleInsert() {
	const (
		key      = "my_key"
		contents = "my cached object"
	)

	// w is a core.WriteCloseAbandoner, a superset of io.WriteCloser.
	// Data written to this handle is streamed into the Fastly cache.
	w, err := core.Insert([]byte(key), core.WriteOptions{
		TTL:           time.Hour,
		SurrogateKeys: []string{key},
		Length:        uint64(len(contents)),
	})
	if err != nil {
		panic(err)
	}

	if _, err := io.WriteString(w, contents); err != nil {
		panic(err)
	}

	// The writer must be closed to complete the cache operation.
	// Always check for errors from Close.
	if err := w.Close(); err != nil {
		panic(err)
	}
}

func ExampleTransaction() {
	// Users of the transactional API should at a minimum anticipate
	// lookups that are obligated to insert an object into the cache,
	// and lookups which are not.  If the stale-while-revalidate
	// parameter is set for a cached object, the user should also
	// distinguish between the insertion and revalidation cases.

	useFoundItem := func(f *core.Found) {
		// Do something with the found item
	}

	buildContents := func() []byte {
		// Build the contents of the cached item
		return []byte("hello world!")
	}

	shouldReplace := func(f *core.Found, contents []byte) bool {
		// Determine whether the cached item should be replaced with
		// the new contents
		return true
	}

	tx, err := core.NewTransaction([]byte("my_key"), core.LookupOptions{})
	if err != nil {
		panic(err)
	}
	defer tx.Close()

	// f is a core.Found value, representing a found cache item.
	// core.ErrNotFound is returned if the item is not cached.
	f, err := tx.Found()
	switch err {
	case nil:
		// A cached item was found, though it might be stale.
		useFoundItem(f)

		// Perform revalidation, if necessary.
		if tx.MustInsertOrUpdate() {
			contents := buildContents()
			if shouldReplace(f, contents) {
				// Use Insert to replace the previous object
				w, err := tx.Insert(core.WriteOptions{
					TTL:           time.Hour,
					SurrogateKeys: []string{"my_key"},
					Length:        uint64(len(contents)),
				})
				if err != nil {
					panic(err)
				}

				if _, err := w.Write(contents); err != nil {
					panic(err)
				}

				if err := w.Close(); err != nil {
					panic(err)
				}
			} else {
				// Otherwise update the stale object's metadata
				if err := tx.Update(core.WriteOptions{
					TTL:           time.Hour,
					SurrogateKeys: []string{"my_key"},
				}); err != nil {
					panic(err)
				}
			}
		}

	case core.ErrNotFound:
		// The item was not found.
		if tx.MustInsert() {
			// We've been chosen to insert the object.
			contents := buildContents()
			w, f, err := tx.InsertAndStreamBack(core.WriteOptions{
				TTL:           time.Hour,
				SurrogateKeys: []string{"my_key"},
				Length:        uint64(len(contents)),
			})
			if err != nil {
				panic(err)
			}

			if _, err := w.Write(contents); err != nil {
				panic(err)
			}

			if err := w.Close(); err != nil {
				panic(err)
			}

			useFoundItem(f)
		} else {
			panic(err)
		}

	default:
		// An unexpected error
		panic(err)
	}
}

func ExampleFound_GetRange() {
	const (
		key      = "my_key"
		contents = "my cached object"
	)

	// Start by filling the cache...
	w, err := core.Insert([]byte(key), core.WriteOptions{
		TTL:           time.Hour,
		SurrogateKeys: []string{key},
		Length:        uint64(len(contents)),
	})
	if err != nil {
		panic(err)
	}
	if _, err := io.WriteString(w, contents); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	// We get a response...
	f, err := core.Lookup([]byte("my_key"), core.LookupOptions{})
	if err != nil {
		panic(err)
	}
	// ...then discard the body, so we can re-open a new body reading a subset of the bytes.
	if err := f.Body.Close(); err != nil {
		panic(err)
	}

	// If we try to read an invalid range (from > to), we get an error:
	_, err = f.GetRange(3, 1)
	if err == nil {
		panic("accepted invalid range")
	}

	// We can use "0" as a signal value to say "everything to the end":
	body, err := f.GetRange(3, 0)
	if err == nil {
		panic("accepted invalid range")
	}
	cachedStr, err := io.ReadAll(body)
	if err != nil {
		panic(err)
	}
	if string(cachedStr) != "cached object" {
		panic(fmt.Sprintf("got: %q, want: %q", cachedStr, "cached object"))
	}

	fmt.Printf("The cached value was: %s", cachedStr)
}

func ExampleLookupOptions_AlwaysUseRequestedRange() {
	const (
		key      = "my_key"
		contents = "my cached object"
	)

	// Start an insert; this will be concurrent with the read.
	// Data written to this handle is streamed into the Fastly cache.
	w, err := core.Insert([]byte(key), core.WriteOptions{
		TTL:           time.Hour,
		SurrogateKeys: []string{key},
		Length:        uint64(len(contents)),
	})
	if err != nil {
		panic(err)
	}

	// With the write still outstanding, start a lookup for a specific range.
	f, err := core.Lookup([]byte("my_key"), core.LookupOptions{
		From:                    3,
		To:                      8,
		AlwaysUseRequestedRange: true,
	})
	if err != nil {
		panic(err)
	}

	// Write and flush:
	if _, err := io.WriteString(w, contents); err != nil {
		panic(err)
	}

	cachedStr, err := io.ReadAll(f.Body)
	if err != nil {
		panic(err)
	}
	if string(cachedStr) != "cached" {
		panic(fmt.Sprintf("got: %q, want: %q", cachedStr, "cached"))
	}

	t.Logf("The cached value was: %s", cachedStr)
}
