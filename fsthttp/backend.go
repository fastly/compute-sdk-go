package fsthttp

import (
	"errors"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrDynamicBackendDisallowed indicates the service is not allowed to
	// create dynamic backends.
	ErrDynamicBackendDisallowed = errors.New("dynamic backends not supported for this service")

	// ErrBackendNameInUse indicates the backend name is already in use.
	ErrBackendNameInUse = errors.New("backend name already in use")
)

type TLSVersion uint32

// Constants for dynamic backend TLS configuration
const (
	TLSVersion1_0 TLSVersion = 0
	TLSVersion1_1 TLSVersion = 1
	TLSVersion1_2 TLSVersion = 2
	TLSVersion1_3 TLSVersion = 3
)

// BackendOption is a builder for the configuration of a dynamic backend.
type BackendOptions struct {
	abiOpts fastly.BackendConfigOptions
}

// Backend is a fastly backend
type Backend struct {
	name   string
	target string
}

// Name returns the name associated with this backend.
func (b *Backend) Name() string {
	return b.name
}

// Target returns the target associated with this backend.
func (b *Backend) Target() string {
	return b.target
}

// HostOverride sets the HTTP Host header on connections to this backend.
func (b *BackendOptions) HostOverride(host string) *BackendOptions {
	b.abiOpts.HostOverride(host)
	return b
}

// ConnectTimeout sets the maximum duration to wait for a connection to this backend to be established.
func (b *BackendOptions) ConnectTimeout(t time.Duration) *BackendOptions {
	b.abiOpts.ConnectTimeout(t)
	return b
}

// FirstByteTimeout sets the maximum duration to wait for the server response to begin after a TCP connection is established and the request has been sent.
func (b *BackendOptions) FirstByteTimeout(t time.Duration) *BackendOptions {
	b.abiOpts.FirstByteTimeout(t)
	return b
}

// BetweenBytesTimeout sets the maximum duration that Fastly will wait while receiving no data on a download from a backend.
func (b *BackendOptions) BetweenBytesTimeout(t time.Duration) *BackendOptions {
	b.abiOpts.BetweenBytesTimeout(t)
	return b
}

// UseSSL sets whether or not to require TLS for connections to this backend.
func (b *BackendOptions) UseSSL(v bool) *BackendOptions {
	b.abiOpts.UseSSL(v)
	return b
}

// SSLMinVersion sets the minimum allowed TLS version on SSL connections to this backend.
func (b *BackendOptions) SSLMinVersion(min TLSVersion) *BackendOptions {
	b.abiOpts.SSLMinVersion(fastly.TLSVersion(min))
	return b
}

// SSLMaxVersion sets the maximum allowed TLS version on SSL connections to this backend.
func (b *BackendOptions) SSLMaxVersion(max TLSVersion) *BackendOptions {
	b.abiOpts.SSLMaxVersion(fastly.TLSVersion(max))
	return b
}

// CertHostname sets the hostname that the server certificate should declare.
func (b *BackendOptions) CertHostname(host string) *BackendOptions {
	b.abiOpts.CertHostname(host)
	return b
}

// CACert sets the CA certificate to use when checking the validity of the backend.
func (b *BackendOptions) CACert(cert string) *BackendOptions {
	b.abiOpts.CACert(cert)
	return b
}

// Ciphers sets the list of OpenSSL ciphers to support for connections to this origin.
func (b *BackendOptions) Ciphers(ciphers string) *BackendOptions {
	b.abiOpts.Ciphers(ciphers)
	return b
}

// SNIHostname sets the SNI hostname to use on connections to this backend.
func (b *BackendOptions) SNIHostname(host string) *BackendOptions {
	b.abiOpts.SNIHostname(host)
	return b
}

// Register a new dynamic backend.
func RegisterDynamicBackend(name string, target string, options *BackendOptions) (*Backend, error) {
	var abiOpts *fastly.BackendConfigOptions
	if options != nil {
		abiOpts = &options.abiOpts
	} else {
		abiOpts = &fastly.BackendConfigOptions{}
	}
	err := fastly.RegisterDynamicBackend(name, target, abiOpts)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusUnsupported:
			return nil, ErrDynamicBackendDisallowed
		case ok && status == fastly.FastlyStatusError:
			return nil, ErrBackendNameInUse
		default:
			return nil, err
		}
	}
	b := Backend{
		name:   name,
		target: target,
	}
	return &b, nil
}