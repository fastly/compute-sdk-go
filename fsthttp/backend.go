package fsthttp

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
	"github.com/fastly/compute-sdk-go/secretstore"
)

var (
	// ErrDynamicBackendDisallowed indicates the service is not allowed to
	// create dynamic backends.
	ErrDynamicBackendDisallowed = errors.New("dynamic backends not supported for this service")

	// ErrBackendNameInUse indicates the backend name is already in use.
	ErrBackendNameInUse = errors.New("backend name already in use")

	// ErrBackendNotFound indicates the provided backend was not found.
	ErrBackendNotFound = errors.New("backend not found")

	// ErrUnexpected indicates an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
)

type BackendHealth uint32

// Constants for dynamic backend health status
const (
	BackendHealthUnknown   BackendHealth = 0
	BackendHealthHealthy   BackendHealth = 1
	BackendHealthUnhealthy BackendHealth = 2
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

	// has the config been populated
	dynamic bool

	hostOverride        string
	connectTimeout      time.Duration
	firstByteTimeout    time.Duration
	betweenBytesTimeout time.Duration
	isSSL               bool
	sslMinVersion       TLSVersion
	sslMaxVersion       TLSVersion
}

func BackendFromName(name string) (*Backend, error) {
	var err error

	exists, err := fastly.BackendExists(name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrBackendNotFound
	}

	b := &Backend{
		name: name,
	}

	if err := b.populateConfig(); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Backend) populateConfig() error {
	var err error

	b.dynamic, err = fastly.BackendIsDynamic(b.name)
	if err := ignoreNoneError(err); err != nil {
		return err
	}

	host, err := fastly.BackendGetHost(b.name)
	if err := ignoreNoneError(err); err != nil {
		return err
	}

	port, err := fastly.BackendGetPort(b.name)
	if err := ignoreNoneError(err); err != nil {
		return err
	}

	b.target = host + ":" + strconv.Itoa(port)

	b.hostOverride, err = fastly.BackendGetOverrideHost(b.name)
	if err := ignoreNoneError(err); err != nil {
		return err
	}

	// Timing-related calls return FastlyStatusUnsupported under
	// Viceroy, so filter that out for these hostcalls too.

	b.connectTimeout, err = fastly.BackendGetConnectTimeout(b.name)
	if err := ignoreUnsupportedError(ignoreNoneError(err)); err != nil {
		return err
	}

	b.firstByteTimeout, err = fastly.BackendGetFirstByteTimeout(b.name)
	if err := ignoreUnsupportedError(ignoreNoneError(err)); err != nil {
		return err
	}

	b.betweenBytesTimeout, err = fastly.BackendGetBetweenBytesTimeout(b.name)
	if err := ignoreUnsupportedError(ignoreNoneError(err)); err != nil {
		return err
	}

	b.isSSL, err = fastly.BackendIsSSL(b.name)
	if err := ignoreNoneError(err); err != nil {
		return err
	}

	if b.isSSL {
		// SSL version calls also return FastlyStatusUnsupported under
		// Viceroy.

		var v fastly.TLSVersion
		v, err = fastly.BackendGetSSLMaxVersion(b.name)
		if err := ignoreUnsupportedError(ignoreNoneError(err)); err != nil {
			return err
		}
		b.sslMaxVersion = TLSVersion(v)

		v, err = fastly.BackendGetSSLMinVersion(b.name)
		if err := ignoreUnsupportedError(ignoreNoneError(err)); err != nil {
			return err
		}
		b.sslMinVersion = TLSVersion(v)
	}

	return nil
}

// Name returns the name associated with this backend.
func (b *Backend) Name() string {
	return b.name
}

// Target returns the target associated with this backend.
func (b *Backend) Target() string {
	return b.target
}

// Health dynamically checks the backend's health status.
func (b *Backend) Health() (BackendHealth, error) {
	v, err := fastly.BackendIsHealthy(b.name)
	if err != nil {
		return BackendHealthUnknown, err
	}
	return BackendHealth(v), nil
}

// IsDynamic returns whether the backend is dynamic.
func (b *Backend) IsDynamic() bool {
	return b.dynamic
}

func (b *Backend) HostOverride() string {
	return b.hostOverride
}

func (b *Backend) ConnectTimeout() time.Duration {
	return b.connectTimeout
}

func (b *Backend) FirstByteTimeout() time.Duration {
	return b.firstByteTimeout
}

func (b *Backend) BetweenBytesTimeout() time.Duration {
	return b.betweenBytesTimeout
}

func (b *Backend) IsSSL() bool {
	return b.isSSL
}

func (b *Backend) SSLMaxVersion() TLSVersion {
	return b.sslMaxVersion
}

func (b *Backend) SSLMinVersion() TLSVersion {
	return b.sslMinVersion
}

func NewBackendOptions() *BackendOptions {
	return &BackendOptions{}
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
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) SSLMinVersion(min TLSVersion) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.SSLMinVersion(fastly.TLSVersion(min))
	return b
}

// SSLMaxVersion sets the maximum allowed TLS version on SSL connections to this backend.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) SSLMaxVersion(max TLSVersion) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.SSLMaxVersion(fastly.TLSVersion(max))
	return b
}

// CertHostname sets the hostname that the server certificate should declare.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) CertHostname(host string) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.CertHostname(host)
	return b
}

// CACert sets the CA certificate to use when checking the validity of the backend.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) CACert(cert string) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.CACert(cert)
	return b
}

// Ciphers sets the list of OpenSSL ciphers to support for connections to this origin.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) Ciphers(ciphers string) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.Ciphers(ciphers)
	return b
}

// SNIHostname sets the SNI hostname to use on connections to this backend.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) SNIHostname(host string) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.SNIHostname(host)
	return b
}

// ClientCertificate sets the client certificate to be provided to the server as part of the SSL handshake.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) ClientCertificate(certificate string, key secretstore.Secret) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.ClientCert(certificate, key.Handle())
	return b
}

// PoolConnections allows users to turn connection pooling on or off for the
// backend. Pooling allows the Compute platform to reuse connections across
// multiple sessions, resulting in lower resource use at the server (because it
// does not need to reperform the TCP handhsake and TLS authentication when the
// connection is reused). The default is to pool connections. Set this to false
// to create a new connection to the backend for every incoming session.
func (b *BackendOptions) PoolConnections(poolingOn bool) *BackendOptions {
	b.abiOpts.PoolConnections(poolingOn)
	return b
}

// UseGRPC sets whether or not to connect to the backend via gRPC
func (b *BackendOptions) UseGRPC(v bool) *BackendOptions {
	b.abiOpts.UseGRPC(v)
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
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}
	b := Backend{
		name:   name,
		target: target,
	}

	if err := b.populateConfig(); err != nil {
		return nil, err
	}

	return &b, nil
}

func ignoreNoneError(err error) error {
	status, ok := fastly.IsFastlyError(err)
	if ok && status == fastly.FastlyStatusNone {
		return nil
	}
	return err
}

func ignoreUnsupportedError(err error) error {
	status, ok := fastly.IsFastlyError(err)
	if ok && status == fastly.FastlyStatusUnsupported {
		return nil
	}
	return err
}
