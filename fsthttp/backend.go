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
	err     []error
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

const (
	maxBackendTimeout = (1 << 32) * time.Millisecond
)

type backendValidationError struct {
	field, reason string
}

func (b *backendValidationError) Error() string {
	return "backend config validation error: field " + b.field + ": " + b.reason
}

// HostOverride sets the HTTP Host header on connections to this backend.
func (b *BackendOptions) HostOverride(host string) *BackendOptions {
	b.abiOpts.HostOverride(host)
	return b
}

// ConnectTimeout sets the maximum duration to wait for a connection to this backend to be established.
func (b *BackendOptions) ConnectTimeout(t time.Duration) *BackendOptions {
	if t > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "ConnectTimeout", reason: "too large"})
		return b
	}
	b.abiOpts.ConnectTimeout(t)
	return b
}

// FirstByteTimeout sets the maximum duration to wait for the server response to begin after a TCP connection is established and the request has been sent.
func (b *BackendOptions) FirstByteTimeout(t time.Duration) *BackendOptions {
	if t > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "FirstByteTimeout", reason: "too large"})
		return b
	}
	b.abiOpts.FirstByteTimeout(t)
	return b
}

// BetweenBytesTimeout sets the maximum duration that Fastly will wait while receiving no data on a download from a backend.
func (b *BackendOptions) BetweenBytesTimeout(t time.Duration) *BackendOptions {
	if t > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "BetweenBytesTimeout", reason: "too large"})
		return b
	}
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
	if host == "" {
		b.err = append(b.err, &backendValidationError{"CertHostname", "field cannot be blank"})
		return b
	}
	b.abiOpts.UseSSL(true)
	b.abiOpts.CertHostname(host)
	return b
}

// CACert sets the CA certificate to use when checking the validity of the backend.
// Setting this will enable SSL for the connection as a side effect.
//
// If CACert is not provided (default), the backend's certificate is validated using a set of public root CAs.
func (b *BackendOptions) CACert(cert string) *BackendOptions {
	if cert == "" {
		b.err = append(b.err, &backendValidationError{"CACert", "field cannot be blank"})
		return b
	}
	b.abiOpts.UseSSL(true)
	b.abiOpts.CACert(cert)
	return b
}

// Ciphers sets the list of OpenSSL ciphers to support for connections to this origin.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) Ciphers(ciphers string) *BackendOptions {
	if ciphers == "" {
		b.err = append(b.err, &backendValidationError{"Ciphers", "field cannot be blank"})
		return b
	}
	b.abiOpts.UseSSL(true)
	b.abiOpts.Ciphers(ciphers)
	return b
}

// SNIHostname sets the SNI hostname to use on connections to this backend.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) SNIHostname(host string) *BackendOptions {
	if host == "" {
		b.err = append(b.err, &backendValidationError{"SNIHostname", "field cannot be blank"})
		return b
	}
	b.abiOpts.UseSSL(true)
	b.abiOpts.SNIHostname(host)
	return b
}

// ClientCertificate sets the client certificate to be provided to the server as part of the SSL handshake.
// Setting this will enable SSL for the connection as a side effect.
func (b *BackendOptions) ClientCertificate(certificate string, key secretstore.Secret) *BackendOptions {
	if certificate == "" {
		b.err = append(b.err, &backendValidationError{"ClientCertificate", "field cannot be blank"})
		return b
	}
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
func (b *BackendOptions) HTTPKeepaliveTime(t time.Duration) *BackendOptions {
	if t > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "HTTPKeepaliveTime", reason: "too large"})
		return b
	}
	b.abiOpts.HTTPKeepaliveTime(t)
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
	if interval > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "HTTPKeepaliveTime", reason: "too large"})
		return b
	}
	b.abiOpts.TCPKeepaliveTime(interval)
	return b
}

// UseGRPC sets whether or not to connect to the backend via gRPC
func (b *BackendOptions) UseGRPC(v bool) *BackendOptions {
	b.abiOpts.UseGRPC(v)
	return b
}

// MaxConnections sets how many connections to allow in the connection pool for this backend.
//
// `0` is treated as unlimited. The default is `200`.
//
// Note that this limit is best determined experimentally, since the total number of
// connections to the backend will depend on POP sizes, HTTP keepalive limits, and the traffic
// patterns for individual POPs.
func (b *BackendOptions) MaxConnections(count uint32) *BackendOptions {
	b.abiOpts.MaxConnections(count)
	return b
}

// MaxUse sets how many times an HTTP keepalive connection can be reused in a connection pool.
//
// `0` is treated as unlimited. The default is `0`.
func (b *BackendOptions) MaxUse(count uint32) *BackendOptions {
	b.abiOpts.MaxUse(count)
	return b
}

// MaxLifetime sets an upper bound for how long a pooled HTTP keepalive connection is allowed to have
// been open before we stop trying to reuse it.
//
// 0ms is treated as unlimited. The default is 0ms.
func (b *BackendOptions) MaxLifetime(t time.Duration) *BackendOptions {
	if t > maxBackendTimeout {
		b.err = append(b.err, &backendValidationError{field: "MaxLifetime", reason: "too large"})
		return b
	}
	b.abiOpts.MaxLifetime(t)
	return b
}

// PreferIPV6 sets whether to prefer trying IPv6 connections first before IPv4 when a hostname
// has both A and AAAA records.
//
// This defaults to `true`.
func (b *BackendOptions) PreferIPV6(v bool) *BackendOptions {
	b.abiOpts.PreferIPV4(!v)
	return b
}

// Healthcheck sets the backend health check configuration
//
// NOTE: Support for this feature is experimental and may be removed at any time.
func (b *BackendOptions) Healthcheck(h *BackendHealthcheckOptions) *BackendOptions {
	if h == nil {
		b.err = append(b.err, &backendValidationError{field: "Healthcheck", reason: "field cannot be nil"})
		return b
	}
	if len(h.err) > 0 {
		b.err = append(b.err, h.err...)
		return b
	}
	b.abiOpts.Healthcheck(h.abiOpts)
	return b
}

const (
	healthcheckURLSizeLimit    = 8192
	healthcheckMethodSizeLimit = 8192
	healthcheckWindowLimit     = 15
	healthcheckIntervalMin     = time.Second
	healthcheckIntervalMax     = time.Hour
	healthcheckTimeoutMin      = time.Second
	healthcheckTimeoutMax      = time.Hour
)

type BackendHealthcheckOptions struct {
	abiOpts *fastly.BackendHealthcheckConfig
	err     []error

	host      string
	path      string
	window    uint32
	threshold uint32
	initial   uint32
}

func NewBackendHealthcheck(host string) *BackendHealthcheckOptions {
	h := &BackendHealthcheckOptions{
		abiOpts:   fastly.NewBackendHealthConfig(host),
		host:      host,
		path:      "/",
		window:    5,
		threshold: 3,
		initial:   4,
	}
	if host == "" {
		h.err = append(h.err, &backendValidationError{field: "Host", reason: "field cannot be blank"})
		return h
	}
	if len(host)+len(h.path) > healthcheckURLSizeLimit {
		h.err = append(h.err, &backendValidationError{field: "Host", reason: "host and path are too large"})
	}
	return h
}

// Interval sets the interval where a health check should be performed.
// Times must be 1 second to 1 hour inclusive. Must be less than the
// current timeout.
//
// Defaults to 15 seconds.
func (h *BackendHealthcheckOptions) Interval(t time.Duration) *BackendHealthcheckOptions {
	if t < healthcheckIntervalMin || t >= healthcheckIntervalMax {
		h.err = append(h.err, &backendValidationError{field: "Interval", reason: "not within 1 second and 1 hour"})
		return h
	}
	h.abiOpts.Interval(t)
	return h
}

// Timeout sets the time in which to perform the health check.
// Note that querying the health check renews the timer. Must not be less
// than the current interval.
//
// Defaults to 5 seconds.
func (h *BackendHealthcheckOptions) Timeout(t time.Duration) *BackendHealthcheckOptions {
	if t < healthcheckTimeoutMin || t >= healthcheckTimeoutMax {
		h.err = append(h.err, &backendValidationError{field: "Timeout", reason: "not within 1 second and 1 hour"})
		return h
	}
	h.abiOpts.Timeout(t)
	return h
}

// Method sets an HTTP verb (i.e., HEAD, GET, or POST) to use when performing the health check.
//
// Defaults to "GET".
func (h *BackendHealthcheckOptions) Method(m string) *BackendHealthcheckOptions {
	if len(m) > healthcheckMethodSizeLimit {
		h.err = append(h.err, &backendValidationError{field: "Method", reason: "too large"})
		return h
	}
	h.abiOpts.Method(m)
	return h
}

// Path sets a path to visit on your origins when performing the check. Use a unique path.
// For example, use /website-healthcheck.txt, not / or /healthcheck.
//
// Defaults to "/".
func (h *BackendHealthcheckOptions) Path(p string) *BackendHealthcheckOptions {
	if len(h.host)+len(p) > healthcheckURLSizeLimit {
		h.err = append(h.err, &backendValidationError{field: "Path", reason: "host and path are too large"})
		return h
	}
	h.path = p
	h.abiOpts.Path(p)
	return h
}

// Status sets the HTTP status code that signifies a healthy state.
//
// Defaults to 200.
func (h *BackendHealthcheckOptions) Status(status uint32) *BackendHealthcheckOptions {
	h.abiOpts.Status(status)
	return h
}

// Window sets the number of most recent health check queries to keep.
// Must not be greater than 15.
//
// Defaults to 5.
func (h *BackendHealthcheckOptions) Window(w uint32) *BackendHealthcheckOptions {
	if w > healthcheckWindowLimit {
		h.err = append(h.err, &backendValidationError{field: "Window", reason: "too large"})
		return h
	}
	if w < h.threshold {
		h.err = append(h.err, &backendValidationError{field: "Window", reason: "threshold is greater than window"})
		return h
	}
	if w < h.initial {
		h.err = append(h.err, &backendValidationError{field: "Window", reason: "initial is greater than window"})
		return h
	}
	h.window = w
	h.abiOpts.Window(w)
	return h
}

// Threshold sets the number of health checks that must be successful within the window
// to be considered healthy. Must not be greater than the current window.
//
// Defaults to 3.
func (h *BackendHealthcheckOptions) Threshold(threshold uint32) *BackendHealthcheckOptions {
	if threshold > h.window {
		h.err = append(h.err, &backendValidationError{field: "Threshold", reason: "greater than window"})
		return h
	}
	h.threshold = threshold
	h.abiOpts.Threshold(threshold)
	return h
}

// Initial sets the number of successes to assume are successful when beginning a health check.
// Must not be greater than the current window.
//
// Defaults to 4.
func (h *BackendHealthcheckOptions) Initial(initial uint32) *BackendHealthcheckOptions {
	if initial > h.window {
		h.err = append(h.err, &backendValidationError{field: "Initial", reason: "greater than window"})
		return h
	}
	h.initial = initial
	h.abiOpts.Initial(initial)
	return h
}

// RegisterDynamicBackend registers a new dynamic backend.
func RegisterDynamicBackend(name string, target string, options *BackendOptions) (*Backend, error) {
	var abiOpts *fastly.BackendConfigOptions
	if options != nil {
		abiOpts = &options.abiOpts
	} else {
		abiOpts = &fastly.BackendConfigOptions{}
	}

	if options != nil && len(options.err) > 0 {
		return nil, errors.Join(options.err...)
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
