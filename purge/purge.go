// Package purge provides cache purging operations for Fastly
// Compute.
//
// See the [Fastly purge documentation] for details.
//
// [Fastly purge documentation]: https://developer.fastly.com/learning/concepts/purging/
package purge

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// PurgeOptions control the behavior of purge operations.
type PurgeOptions struct {
	// Whether to soft purge the item.  A soft purge marks a cached
	// object as stale, rather than invalidating it.
	Soft bool
}

// PurgeSurrogateKey purges all cached objects with the provided
// surrogate key.
func PurgeSurrogateKey(surrogateKey string, opts PurgeOptions) error {
	var abiOpts fastly.PurgeOptions
	abiOpts.SoftPurge(opts.Soft)

	return fastly.PurgeSurrogateKey(surrogateKey, abiOpts)
}
