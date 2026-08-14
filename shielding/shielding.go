package shielding

import (
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Shield is a shielding site within Fastly.
type Shield struct {
	name      string
	runningOn bool
}

// ShieldFromName returns information about a particular shield site.
func ShieldFromName(n string) (*Shield, error) {
	info, err := fastly.ShieldingShieldInfo(n)
	if err != nil {
		return nil, err
	}

	return &Shield{
		name:      n,
		runningOn: info.RunningOn(),
	}, nil
}

// Name returns the name of the shield site.
func (s *Shield) Name() string { return s.name }

// IsRunningOn returns whether the Compute node is currently in the shielding site.
func (s *Shield) IsRunningOn() bool { return s.runningOn }

// BackendOptions controls the options for a backend created for a shield.
type BackendOptions struct {
	abiOpts fastly.ShieldingBackendOptions
}

// FirstByteTimeout sets the first-byte timeout for a derived backend.
func (b *BackendOptions) FirstByteTimeout(t time.Duration) *BackendOptions {
	b.abiOpts.FirstByteTimeout(t)
	return b
}

// Backend returns a named backend for use with the fsthttp package.
func (s *Shield) Backend(opts *BackendOptions) (string, error) {
	return fastly.ShieldingBackendForShield(s.name, &opts.abiOpts)
}
