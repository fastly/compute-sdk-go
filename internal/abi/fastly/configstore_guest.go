//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
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
type ConfigStore struct {
	h configstoreHandle
}

// Config Stores are limited to keys of length 255 character. By default, values are limited to 8000 character values,
// but this can be adjust on a per-customer basis.
// https://docs.fastly.com/en/guides/about-edge-dictionaries#limitations-and-considerations
const configstoreMaxKeyLen = 255 * 3 // known maximum size for config store keys: 755 bytes, for 255 3-byte utf-8 encoded characters

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

// GetBytes returns a slice of newly-allocated memory for the value
// corresponding to key.
func (c *ConfigStore) GetBytes(key string) ([]byte, error) {
	keyBuffer := prim.NewReadBufferFromString(key)
	if keyBuffer.Len() > configstoreMaxKeyLen {
		return nil, FastlyStatusInval.toError()
	}
	keyStr := keyBuffer.Wstring()

	value, err := withAdaptiveBuffer(DefaultSmallBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyConfigStoreGet(
			c.h,
			keyStr.Data, keyStr.Len,
			prim.ToPointer(buf.Char8Pointer()), buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}
	return value.AsBytes(), nil
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
