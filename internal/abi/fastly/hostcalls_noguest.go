//go:build !wasip1 || nofastlyhostcalls

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

func (b *HTTPBody) Length() (uint64, error) {
	return 0, fmt.Errorf("not implemented")
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

func (r *HTTPRequest) DownstreamClientIPAddr() (net.IP, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamServerIPAddr() (net.IP, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSCipherOpenSSLName() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSProtocol() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSClientHello() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSJA3MD5() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamH2Fingerprint() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamRequestID() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamOHFingerprint() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamDDOSDetected() (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSRawClientCertificate() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSClientCertVerifyResult() (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamTLSJA4() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamComplianceRegion() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) DownstreamFastlyKeyIsValid() (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func NewHTTPRequest() (*HTTPRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderNames() *Values {
	return nil
}

func (r *HTTPRequest) DownstreamOriginalHeaderNames() *Values {
	return nil
}

func (r *HTTPRequest) DownstreamOriginalHeaderCount() (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderValue(name string, maxHeaderValueLen int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetHeaderValues(name string) *Values {
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

func (r *HTTPRequest) GetMethod() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetMethod(method string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) GetURI() (string, error) {
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

func (r *HTTPRequest) SendV3(requestBody *HTTPBody, backend string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

type PendingRequest struct{}

func (r *HTTPRequest) SendAsync(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SendAsyncV2(requestBody *HTTPBody, backend string, streaming bool) (*PendingRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SendAsyncStreaming(requestBody *HTTPBody, backend string) (*PendingRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SendToImageOpto(requestBody *HTTPBody, backend, query string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (r *PendingRequest) Poll() (done bool, response *HTTPResponse, responseBody *HTTPBody, err error) {
	return false, nil, nil, fmt.Errorf("not implemented")
}

func (r *PendingRequest) Wait() (response *HTTPResponse, responseBody *HTTPBody, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetAutoDecompressResponse(options AutoDecompressResponseOptions) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) SetFramingHeadersMode(manual bool) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) HandoffWebsocket(backend string) error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPRequest) HandoffFanout(backend string) error {
	return fmt.Errorf("not implemented")
}

type HTTPRequestPromise struct{}

func DownstreamNextRequest(opts *NextRequestOptions) (*HTTPRequestPromise, error) {
	return nil, fmt.Errorf("not implemented")
}

func (HTTPRequestPromise) Wait() (*HTTPRequest, *HTTPBody, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (HTTPRequestPromise) Abandon() error {
	return fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetAddrDestIP() (net.IP, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetAddrDestPort() (uint16, error) {
	return 0, fmt.Errorf("not implemented")
}

func HandoffWebsocket(backend string) error {
	return fmt.Errorf("not implemented")
}

func HandoffFanout(backend string) error {
	return fmt.Errorf("not implemented")
}

func RegisterDynamicBackend(name string, target string, opts *BackendConfigOptions) error {
	return fmt.Errorf("not implemented")
}

func BackendExists(name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func BackendIsHealthy(name string) (BackendHealth, error) {
	return BackendHealthUnknown, fmt.Errorf("not implemented")
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

func (r *HTTPResponse) GetHeaderNames() *Values {
	return nil
}

func (r *HTTPResponse) GetHeaderValue(name string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (r *HTTPResponse) GetHeaderValues(name string) *Values {
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

func (d *Dictionary) GetBytes(key string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Dictionary) Get(key string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (d *Dictionary) Has(key string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

type ConfigStore struct{}

func OpenConfigStore(name string) (*ConfigStore, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *ConfigStore) GetBytes(key string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *ConfigStore) Get(key string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (d *ConfigStore) Has(key string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func GeoLookup(ip net.IP) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type KVStore struct{}

func OpenKVStore(name string) (*KVStore, error) {
	return nil, fmt.Errorf("not implemented")
}

func (o *KVStore) Lookup(key string) (kvstoreLookupHandle, error) {
	return 0, fmt.Errorf("not implemented")
}

func (o *KVStore) LookupWait(h kvstoreLookupHandle) (KVLookupResult, error) {
	return KVLookupResult{}, fmt.Errorf("not implemented")
}

func (o *KVStore) Insert(key string, value io.Reader, config *KVInsertConfig) (kvstoreInsertHandle, error) {
	return 0, fmt.Errorf("not implemented")
}

func (o *KVStore) InsertWait(h kvstoreInsertHandle) error {
	return fmt.Errorf("not implemented")
}

func (o *KVStore) Delete(key string) (kvstoreDeleteHandle, error) {
	return 0, fmt.Errorf("not implemented")
}

func (o *KVStore) DeleteWait(h kvstoreDeleteHandle) error {
	return fmt.Errorf("not implemented")
}

func (kv *KVStore) List(config *KVListConfig) (kvstoreListHandle, error) {
	return 0, fmt.Errorf("not implemented")
}

func (kv *KVStore) ListWait(listH kvstoreListHandle) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
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

func (s *Secret) Handle() secretHandle {
	return 0
}

func SecretFromBytes(b []byte) (*Secret, error) {
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

func (o *CacheLookupOptions) SetAlwaysUseRequestedRange(alwaysUseRequestedRange bool) error {
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

func DeviceLookup(userAgent string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func ERLCheckRate(rateCounter, entry string, delta uint32, window RateWindow, limit uint32, penaltyBox string, ttl time.Duration) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func RateCounterIncrement(rateCounter, entry string, delta uint32) error {
	return fmt.Errorf("not implemented")
}

func RateCounterLookupRate(rateCounter, entry string, window RateWindow) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

func RateCounterLookupCount(rateCounter, entry string, duration CounterDuration) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

func PenaltyBoxAdd(penaltyBox, entry string, ttl time.Duration) error {
	return fmt.Errorf("not implemented")
}

func PenaltyBoxHas(penaltyBox, entry string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func GetVCPUMilliseconds() (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func GetHeapMiB() (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

type ACLHandle struct{}

func OpenACL(name string) (*ACLHandle, error) {
	return nil, fmt.Errorf("not implemented")
}

func (acl *ACLHandle) Lookup(ip net.IP) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

type HTTPCacheLookupOptions struct{}

func (HTTPCacheLookupOptions) OverrideKey(key string) {
}

func HTTPCacheIsRequestCacheable(req *HTTPRequest) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func HTTPCacheGetSuggestedCacheKey(req *HTTPRequest) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type HTTPCacheHandle struct{}

func HTTPCacheLookup(req *HTTPRequest, opts *HTTPCacheLookupOptions) (*HTTPCacheHandle, error) {
	return nil, fmt.Errorf("not implemented")
}

func HTTPCacheTransactionLookup(req *HTTPRequest, opts *HTTPCacheLookupOptions) (*HTTPCacheHandle, error) {
	return nil, fmt.Errorf("not implemented")
}

type HTTPCacheWriteOptions struct{}

func (o *HTTPCacheWriteOptions) FillConfigMask() {}

func (o *HTTPCacheWriteOptions) SetMaxAgeNs(maxAge uint64) {}

func (o *HTTPCacheWriteOptions) MaxAgeNs() uint64 { return 0 }

func (o *HTTPCacheWriteOptions) SetVaryRule(rule string) {}

func (o *HTTPCacheWriteOptions) VaryRule() (string, bool) { return "", false }

func (o *HTTPCacheWriteOptions) SetInitialAgeNs(initialAge uint64) {}

func (o *HTTPCacheWriteOptions) InitialAgeNs() (uint64, bool) {
	return 0, false
}

func (o *HTTPCacheWriteOptions) SetStaleWhileRevalidateNs(staleWhileRevalidateNs uint64) {
}

func (o *HTTPCacheWriteOptions) StaleWhileRevalidateNs() (uint64, bool) {
	return 0, false
}

func (o *HTTPCacheWriteOptions) SetSurrogateKeys(keys string) {}

func (o *HTTPCacheWriteOptions) SurrogateKeys() (string, bool) {
	return "", false
}

func (o *HTTPCacheWriteOptions) SetLength(length uint64) {}

func (o *HTTPCacheWriteOptions) Length() (uint64, bool) { return 0, false }

func (o *HTTPCacheWriteOptions) SetSensitiveData(sensitive bool) {}

func (o *HTTPCacheWriteOptions) SensitiveData() bool { return false }

func HTTPCacheTransactionInsert(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, error) {
	return nil, fmt.Errorf("not implemented")
}

func HTTPCacheTransactionInsertAndStreamback(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPBody, *HTTPCacheHandle, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func HTTPCacheTransactionUpdate(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) error {
	return fmt.Errorf("not implemented")
}

func HTTPCacheTransactionUpdateAndReturnFresh(h *HTTPCacheHandle, resp *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPCacheHandle, error) {
	return nil, fmt.Errorf("not implemented")
}

func HTTPCacheTransactionRecordNotCacheable(h *HTTPCacheHandle, opts *HTTPCacheWriteOptions) error {
	return fmt.Errorf("not implemented")
}

func HTTPCacheTransactionAbandon(h *HTTPCacheHandle) error {
	return fmt.Errorf("not implemented")
}

func HTTPCacheTransactionClose(h *HTTPCacheHandle) error {
	return fmt.Errorf("not implemented")
}

func HTTPCacheGetSuggestedBackendRequest(h *HTTPCacheHandle) (*HTTPRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

func HTTPCacheGetSuggestedCacheOptions(h *HTTPCacheHandle, r *HTTPResponse, opts *HTTPCacheWriteOptions) (*HTTPCacheWriteOptions, error) {
	return nil, fmt.Errorf("not implemented")
}

func HTTPCachePrepareResponseForStorage(h *HTTPCacheHandle, r *HTTPResponse) (HTTPCacheStorageAction, *HTTPResponse, error) {
	return 0, nil, fmt.Errorf("not implemented")
}

func HTTPCacheGetFoundResponse(h *HTTPCacheHandle, transform bool) (*HTTPResponse, *HTTPBody, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func HTTPCacheGetState(h *HTTPCacheHandle) (CacheLookupState, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetLength(h *HTTPCacheHandle) (httpCacheObjectLength, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetMaxAgeNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetStaleWhileRevalidateNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetAgeNs(h *HTTPCacheHandle) (httpCacheDurationNs, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetHits(h *HTTPCacheHandle) (httpCacheHitCount, error) {
	return 0, fmt.Errorf("not implemented")
}

func HTTPCacheGetSensitiveData(h *HTTPCacheHandle) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func HTTPCacheGetSurrogateKeys(h *HTTPCacheHandle) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func HTTPCacheGetVaryRule(h *HTTPCacheHandle) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func ShieldingShieldInfo(name string) (*ShieldInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func ShieldingBackendForShield(name string, opts *ShieldingBackendOptions) (string, error) {
	return "", fmt.Errorf("not implemented")
}
