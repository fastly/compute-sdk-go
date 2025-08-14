// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"net/textproto"
)

// Header represents the key-value pairs in a set of HTTP headers. Unlike
// net/http, keys are canonicalized to their lowercase form.
type Header map[string][]string

// NewHeader returns an initialized and empty set of headers.
func NewHeader() Header {
	return map[string][]string{}
}

// Add adds the key, value pair to the headers. It appends to any existing
// values associated with key. The key is case insensitive; it is canonicalized
// by CanonicalHeaderKey.
func (h Header) Add(key, value string) {
	key = CanonicalHeaderKey(key)
	h[key] = append(h[key], value)
}

// Del deletes the values associated with key. The key is case insensitive; it
// is canonicalized by CanonicalHeaderKey.
func (h Header) Del(key string) {
	key = CanonicalHeaderKey(key)
	delete(h, key)
}

// Get gets the first value associated with the given key. It is case
// insensitive; CanonicalHeaderKey is used to canonicalize the provided key. If
// there are no values associated with the key, Get returns "".
func (h Header) Get(key string) string {
	key = CanonicalHeaderKey(key)
	if values := h[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Set sets the header entries associated with key to the single element value.
// It replaces any existing values associated with key. The key is case
// insensitive; it is canonicalized by CanonicalHeaderKey.
func (h Header) Set(key, value string) {
	key = CanonicalHeaderKey(key)
	h[key] = []string{value}
}

// Keys returns all keys in the header collection.
func (h Header) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}

// Values returns all values associated with the given key. It is case
// insensitive; CanonicalHeaderKey is used to canonicalize the provided key. The
// returned slice is not a copy.
func (h Header) Values(key string) []string {
	key = CanonicalHeaderKey(key)
	return h[key]
}

// Clone returns a copy of the headers.
func (h Header) Clone() Header {
	clone := NewHeader()
	clone.Apply(h)
	return clone
}

// Reset deletes all existing headers, and adds all of the headers in hs.
func (h Header) Reset(hs Header) {
	for key := range h {
		h.Del(key)
	}
	h.Apply(hs)
}

// Apply adds all of the headers in hs. In the case of key conflict,
// values from hs totally overwrite existing values in h.
func (h Header) Apply(hs Header) {
	for _, key := range hs.Keys() {
		h.Del(key)
		for _, value := range hs.Values(key) {
			h.Add(key, value)
		}
	}
}

// CanonicalHeaderKey returns the canonical format of the header key s. The
// canonicalization converts the first letter and any letter following a hyphen
// to upper case; the rest are converted to lowercase. For example, the
// canonical key for "accept-encoding" is "Accept-Encoding". If s contains a
// space or invalid header field bytes, it is returned without modifications.
func CanonicalHeaderKey(s string) string {
	return textproto.CanonicalMIMEHeaderKey(s)
}
