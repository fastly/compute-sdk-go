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

// String returns a string representation of the backend health.
func (h BackendHealth) String() string {
	switch h {
	case BackendHealthHealthy:
		return "healthy"
	case BackendHealthUnhealthy:
		return "unhealthy"
	case BackendHealthUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

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
//
// When using TLS, Fastly checks the validity of the backend's certificate, and fails the connection if the certificate is invalid.
// This check is not optional: an invalid certificate will cause the backend connection to fail (but read on).
//
// By default, the validity check does not require that the certificate hostname matches the hostname of your request.
// You can use [BackendOptions.CertHostname] to request a check of the certificate hostname.
//
// By default, certificate validity uses a set of public certificate authorities.
// You can specify an alternative CA using [BackendOptions.CACert].
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
//
// If CertHostname is not provided (default), the server certificate's hostname can have any value.
func (b *BackendOptions) CertHostname(host string) *BackendOptions {
	b.abiOpts.UseSSL(true)
	b.abiOpts.CertHostname(host)
	return b
}

// CACert sets the CA certificate to use when checking the validity of the backend.
// Setting this will enable SSL for the connection as a side effect.
//
// If CACert is not provided (default), the backend's certificate is validated using a set of public root CAs.
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
// multiple executions, resulting in lower resource use at the server (because it
// does not need to repeat the TCP handhsake and TLS authentication when the
// connection is reused). The default is to pool connections. Set this to false
// to create a new connection to the backend for every incoming request.
func (b *BackendOptions) PoolConnections(poolingOn bool) *BackendOptions {
	b.abiOpts.PoolConnections(poolingOn)
	return b
}

// HTTPKeepaliveTime configures how long to allow HTTP connections to remain
// idle in a connection pool before it should be considered closed.
func (b *BackendOptions) HTTPKeepaliveTime(time time.Duration) *BackendOptions {
	b.abiOpts.HTTPKeepaliveTime(time)
	return b
}

// TCPKeepaliveEnable sets whether or not to use TCP keepalives to try to
// maintain the connetion to the backend.
func (b *BackendOptions) TCPKeepaliveEnable(enable bool) *BackendOptions {
	b.abiOpts.TCPKeepaliveEnable(enable)
	return b
}

// TCPKeepaliveInterval sets the interval to use when sending TCP keepalive
// probes. Intervals of less than 1 second will be rounded up to 1 second.
//
// Setting this value implicitly enables TCP keepalives. If you are calling both
// this method and `TCPKeepAliveEnable` with dynamically loaded or generated
// values, make sure to call `TCPKeepAliveEnable` last.
func (b *BackendOptions) TCPKeepaliveInterval(interval time.Duration) *BackendOptions {
	if interval < time.Second {
		interval = time.Second
	}

	b.abiOpts.TCPKeepaliveInterval(interval)

	return b
}

// TCPKeepaliveProbes sets how many unanswered TCP probes we should send to the
// backend before we consider the connection dead. Setting this value
// implicitly enables TCP keepalives.
func (b *BackendOptions) TCPKeepaliveProbes(count uint32) *BackendOptions {
	b.abiOpts.TCPKeepaliveProbes(count)
	return b
}

// TCPKeepaliveTime sets how long to wait after the last data was sent before
// starting to send keepalive probes. Setting this value implicitly enables
// TCP keepalives.
func (b *BackendOptions) TCPKeepaliveTime(interval time.Duration) *BackendOptions {
	b.abiOpts.TCPKeepaliveTime(interval)
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
