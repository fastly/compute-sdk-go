//go:build (!tinygo.wasm && !wasi) || nofastlyhostcalls
// +build !tinygo.wasm,!wasi nofastlyhostcalls

//revive:disable:exported

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"fmt"
	"io"
	"net"
	"time"
)

func ParseUserAgent(userAgent string) (family, major, minor, patch string, err error) {
	return "", "", "", "", fmt.Errorf("not implemented")
}

type HTTPBody struct{}

func (b *HTTPBody) Append(other *HTTPBody) error {
	return fmt.Errorf("not implemented")
}

func NewHTTPBody() (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *HTTPBody) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (b *HTTPBody) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("not implemented")
}

func (b *HTTPBody) Close() error {
	return fmt.Errorf("not implemented")
}

func (b *HTTPBody) Abandon() error {
	return fmt.Errorf("not implemented")
}

type LogEndpoint struct{}

func GetLogEndpoint(name string) (*LogEndpoint, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *LogEndpoint) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("not implemented")
}

type HTTPRequest struct{}

func BodyDownstreamGet() (*HTTPRequest, *HTTPBody, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetCacheOverride(options CacheOverrideOptions) error {
	return fmt.Errorf("not implemented")
}

func DownstreamClientIPAddr() (net.IP, error) {
	return nil, fmt.Errorf("not implemented")
}

func DownstreamTLSCipherOpenSSLName() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func DownstreamTLSProtocol() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func DownstreamTLSClientHello() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func NewHTTPRequest() (*HTTPRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderNames(maxHeaderNameLen int) *Values {
	return nil
}

func GetOriginalHeaderNames() *Values {
	return nil
}

func GetOriginalHeaderCount() (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
	return nil
}

func (r *HTTPRequest) SetHeaderValues(name string, values []string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) InsertHeader(name, value string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) AppendHeader(name, value string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) RemoveHeader(name string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetMethod(maxMethodLen int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetMethod(method string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetURI(maxURLLen int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetURI(uri string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetVersion() (proto string, major, minor int, err error) {
	return "", 0, 0, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetVersion(v HTTPVersion) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) Send(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

type PendingRequest struct{}

func (r *HTTPRequest) SendAsync(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SendAsyncStreaming(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *PendingRequest) Poll() (done bool, response *HTTPResponse, responseBody *HTTPBody, err error) {
	return false, nil, nil, fmt.Errorf("not implemented")
}

func (r *PendingRequest) Wait() (response *HTTPResponse, responseBody *HTTPBody, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func PendingRequestSelect(reqs ...*PendingRequest) (index int, done *PendingRequest, response *HTTPResponse, responseBody *HTTPBody, err error) {
	return 0, nil, nil, nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetAutoDecompressResponse(options AutoDecompressResponseOptions) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetFramingHeadersMode(manual bool) error {
	return fmt.Errorf("not implemented")
}

func RegisterDynamicBackend(name string, target string, opts *BackendConfigOptions) error {
	return fmt.Errorf("not implemented")
}

func BackendExists(name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func BackendIsHealthy(name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func BackendIsDynamic(name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func BackendGetHost(name string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func BackendGetOverrideHost(name string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func BackendGetPort(name string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func BackendGetConnectTimeout(name string) (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func BackendGetFirstByteTimeout(name string) (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func BackendGetBetweenBytesTimeout(name string) (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}
func BackendIsSSL(name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func BackendGetSSLMinVersion(name string) (TLSVersion, error) {
	return 0, fmt.Errorf("not implemented")
}

func BackendGetSSLMaxVersion(name string) (TLSVersion, error) {
	return 0, fmt.Errorf("not implemented")
}

type HTTPResponse struct{}

func NewHTTPResponse() (*HTTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetHeaderNames(maxHeaderNameLen int) *Values {
	return nil
}

func (r *HTTPResponse) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetHeaderValues(name string, maxHeaderValueLen int) *Values {
	return nil
}

func (r *HTTPResponse) SetHeaderValues(name string, values []string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) InsertHeader(name, value string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) AppendHeader(name, value string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) RemoveHeader(name string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetVersion() (proto string, major, minor int, err error) {
	return "", 0, 0, fmt.Errorf("not implemented")
}

func (r *HTTPResponse) SetVersion(v HTTPVersion) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) SendDownstream(responseBody *HTTPBody, stream bool) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetStatusCode() (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *HTTPResponse) SetStatusCode(code int) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) SetFramingHeadersMode(manual bool) error {
	return fmt.Errorf("not implemented")
}

type Dictionary struct{}

func OpenDictionary(name string) (*Dictionary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Dictionary) Get(key string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func GeoLookup(ip net.IP) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type KVStore struct{}

func OpenKVStore(name string) (*KVStore, error) {
	return nil, fmt.Errorf("not implemented")
}

func (o *KVStore) Lookup(key string) (io.Reader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (o *KVStore) Insert(key string, value io.Reader) error {
	return fmt.Errorf("not implemented")
}

type (
	SecretStore struct{}
	Secret      struct{}
)

func OpenSecretStore(name string) (*SecretStore, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SecretStore) Get(name string) (*Secret, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *Secret) Plaintext() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type (
	CacheEntry          struct{}
	CacheLookupOptions  struct{}
	CacheGetBodyOptions struct{}
	CacheWriteOptions   struct{}
)

func (o *CacheLookupOptions) SetRequest(req *HTTPRequest) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheGetBodyOptions) From(from uint64) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheGetBodyOptions) To(to uint64) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) MaxAge(v time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) SetRequest(req *HTTPRequest) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) Vary(v []string) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) InitialAge(v time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) StaleWhileRevalidate(v time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) SurrogateKeys(v []string) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) ContentLength(v uint64) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) UserMetadata(v []byte) error {
	return fmt.Errorf("not implemented")
}

func (o *CacheWriteOptions) SensitiveData(v bool) error {
	return fmt.Errorf("not implemented")
}

func CacheLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func CacheInsert(key []byte, opts CacheWriteOptions) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

func CacheTransactionLookup(key []byte, opts CacheLookupOptions) (*CacheEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *CacheEntry) Insert(opts CacheWriteOptions) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *CacheEntry) InsertAndStreamBack(opts CacheWriteOptions) (*HTTPBody, *CacheEntry, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (e *CacheEntry) Update(opts CacheWriteOptions) error {
	return fmt.Errorf("not implemented")
}

func (e *CacheEntry) Cancel() error {
	return fmt.Errorf("not implemented")
}

func (c *CacheEntry) Close() error {
	return fmt.Errorf("not implemented")
}

func (c *CacheEntry) State() (CacheLookupState, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *CacheEntry) UserMetadata() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *CacheEntry) Body(opts CacheGetBodyOptions) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *CacheEntry) Length() (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *CacheEntry) MaxAge() (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *CacheEntry) StaleWhileRevalidate() (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *CacheEntry) Age() (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *CacheEntry) Hits() (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

type PurgeOptions struct{}

func (o *PurgeOptions) SoftPurge(v bool) error {
	return fmt.Errorf("not implemented")
}

func PurgeSurrogateKey(surrogateKey string, opts PurgeOptions) error {
	return fmt.Errorf("not implemented")
}
