//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"sync"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//	(module $fastly_config_store
//	   (@interface func (export "open")
//	      (param $name string)
//	      (result $err (expected $config_store_handle (error $fastly_status)))
//	)
//
//go:wasmimport fastly_config_store open
//go:noescape
func fastlyConfigStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[configstoreHandle],
) FastlyStatus

// ConfigStore represents a Fastly config store a collection of read-only
// key/value pairs. For convenience, keys are modeled as Go strings, and values
// as byte slices.
//
// NOTE: wasm, by definition, is a single-threaded execution environment. This
// allows us to use valueBuf scratch space between the guest and host to avoid
// allocations any larger than necessary, without locking.
type ConfigStore struct {
	h configstoreHandle

	mu       sync.Mutex // protects valueBuf
	valueBuf [configstoreMaxValueLen]byte
}

// Dictionaries are subject to very specific limitations: 255 character keys and 8000 character values, utf-8 encoded.
// The current storage collation limits utf-8 representations to 3 bytes in length.
// https://docs.fastly.com/en/guides/about-edge-dictionaries#limitations-and-considerations
// https://dev.mysql.com/doc/refman/8.4/en/charset-unicode-utf8mb3.html
// https://en.wikipedia.org/wiki/UTF-8#Encoding
const (
	configstoreMaxKeyLen   = 255 * 3  // known maximum size for config store keys: 755 bytes, for 255 3-byte utf-8 encoded characters
	configstoreMaxValueLen = 8000 * 3 // known maximum size for config store values: 24,000 bytes, for 8000 3-byte utf-8 encoded characters
)

// OpenConfigStore returns a reference to the named config store, if it exists.
func OpenConfigStore(name string) (*ConfigStore, error) {
	var c ConfigStore

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyConfigStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&c.h),
	).toError(); err != nil {
		return nil, err
	}
	return &c, nil
}

// witx:
//
//		(@interface func (export "get")
//	       (param $h $config_store_handle)
//	       (param $key string)
//	       (param $value (@witx pointer (@witx char8)))
//	       (param $value_max_len (@witx usize))
//	       (param $nwritten_out (@witx pointer (@witx usize)))
//	       (result $err (expected (error $fastly_status)))
//	   )
//
//go:wasmimport fastly_config_store get
//go:noescape
func fastlyConfigStoreGet(
	h configstoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	value prim.Pointer[prim.Char8],
	valueMaxLen prim.Usize,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

// Get the value for key, if it exists. The returned slice's backing array is
// shared between multiple calls to getBytesUnlocked.
func (c *ConfigStore) getBytesUnlocked(key string) ([]byte, error) {
	keyBuffer := prim.NewReadBufferFromString(key)
	if keyBuffer.Len() > configstoreMaxKeyLen {
		return nil, FastlyStatusInval.toError()
	}
	buf := prim.NewWriteBufferFromBytes(c.valueBuf[:]) // fresh slice of backing array
	keyStr := keyBuffer.Wstring()
	status := fastlyConfigStoreGet(
		c.h,
		keyStr.Data, keyStr.Len,
		prim.ToPointer(buf.Char8Pointer()), buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	)
	if err := status.toError(); err != nil {
		return nil, err
	}
	return buf.AsBytes(), nil
}

// GetBytes returns a slice of newly-allocated memory for the value
// corresponding to key.
func (c *ConfigStore) GetBytes(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, err := c.getBytesUnlocked(key)
	if err != nil {
		return nil, err
	}
	p := make([]byte, len(v))
	copy(p, v)
	return p, nil
}

// Has returns true if key is found.
func (c *ConfigStore) Has(key string) (bool, error) {
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()
	var npointer prim.Usize = 0

	status := fastlyConfigStoreGet(
		c.h,
		keyBuffer.Data, keyBuffer.Len,
		prim.NullChar8Pointer(), 0,
		prim.ToPointer(&npointer),
	)
	switch status {
	case FastlyStatusOK, FastlyStatusBufLen:
		return true, nil
	case FastlyStatusNone:
		return false, nil
	default:
		return false, status.toError()
	}
}
