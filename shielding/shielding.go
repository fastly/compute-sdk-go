package shielding

import (
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Shield is a shielding site withing Fastly.
type Shield struct {
	name      string
	me        bool
	target    string
	sslTarget string
}

// ShieldFromName returns information about a particular shield site.
func ShieldFromName(n string) (*Shield, error) {
	info, err := fastly.ShieldingShieldInfo(n)
	if err != nil {
		return nil, err
	}

	return &Shield{
		name:      n,
		me:        info.Me(),
		target:    info.Target(),
		sslTarget: info.SSLTarget(),
	}, nil
}

// Name returns the name of the shield site.
func (s *Shield) Name() string { return s.name }

// Me returns whether the Compute node is currently in the shielding site.
func (s *Shield) Me() bool { return s.me }

// Target returns the target for unecrypted data.
func (s *Shield) Target() string { return s.target }

// SSLTarget returns the target for encrypted traffic.
func (s *Shield) SSLTarget() string { return s.sslTarget }

// BackendOptions
type BackendOptions struct {
	cacheKey string
}

// Backend returns a named backend for use with the fsthttp package.
func (s *Shield) Backend(opts *BackendOptions) (string, error) {
	var abiOpts fastly.ShieldingBackendOptions
	if opts.cacheKey != "" {
		abiOpts.CacheKey(opts.cacheKey)
	}
	return fastly.ShieldingBackendForShield(s.name, abiOpts)
}
