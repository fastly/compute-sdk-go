//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"sync"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//	(module $fastly_dictionary
//	   (@interface func (export "open")
//	      (param $name string)
//	      (result $err $fastly_status)
//	      (result $h $dictionary_handle)
//	   )
//
//go:wasmimport fastly_dictionary open
//go:noescape
func fastlyDictionaryOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[dictionaryHandle],
) FastlyStatus

// Dictionary represents a Fastly edge dictionary, a collection of read-only
// key/value pairs. For convenience, keys are modeled as Go strings, and values
// as byte slices.
//
// NOTE: wasm, by definition, is a single-threaded execution environment. This
// allows us to use valueBuf scratch space between the guest and host to avoid
// allocations any larger than necessary, without locking.
type Dictionary struct {
	h dictionaryHandle

	mu       sync.Mutex // protects valueBuf
	valueBuf [dictionaryMaxValueLen]byte
}

// Dictionaries are subject to very specific limitations: 255 character keys and 8000 character values, utf-8 encoded.
// The current storage collation limits utf-8 representations to 3 bytes in length.
// https://docs.fastly.com/en/guides/about-edge-dictionaries#limitations-and-considerations
// https://dev.mysql.com/doc/refman/8.4/en/charset-unicode-utf8mb3.html
// https://en.wikipedia.org/wiki/UTF-8#Encoding
const (
	dictionaryMaxKeyLen   = 255 * 3  // known maximum size for config store keys: 755 bytes, for 255 3-byte utf-8 encoded characters
	dictionaryMaxValueLen = 8000 * 3 // known maximum size for config store values: 24,000 bytes, for 8000 3-byte utf-8 encoded characters
)

// OpenDictionary returns a reference to the named dictionary, if it exists.
func OpenDictionary(name string) (*Dictionary, error) {
	var d Dictionary

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyDictionaryOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&d.h),
	).toError(); err != nil {
		return nil, err
	}
	return &d, nil
}

// witx:
//
//	(@interface func (export "get")
//	   (param $h $dictionary_handle)
//	   (param $key string)
//	   (param $value (@witx pointer char8))
//	   (param $value_max_len (@witx usize))
//	   (result $err $fastly_status)
//	   (result $nwritten (@witx usize))
//	)
//
//go:wasmimport fastly_dictionary get
//go:noescape
func fastlyDictionaryGet(
	h dictionaryHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	value prim.Pointer[prim.Char8],
	valueMaxLen prim.Usize,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

// Get the value for key, if it exists. The returned slice's backing array is
// shared between multiple calls to getBytesUnlocked.
func (d *Dictionary) getBytesUnlocked(key string) ([]byte, error) {
	keyBuffer := prim.NewReadBufferFromString(key)
	if keyBuffer.Len() > dictionaryMaxKeyLen {
		return nil, FastlyStatusInval.toError()
	}
	buf := prim.NewWriteBufferFromBytes(d.valueBuf[:]) // fresh slice of backing array
	keyStr := keyBuffer.Wstring()
	status := fastlyDictionaryGet(
		d.h,
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
func (d *Dictionary) GetBytes(key string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, err := d.getBytesUnlocked(key)
	if err != nil {
		return nil, err
	}
	p := make([]byte, len(v))
	copy(p, v)
	return p, nil
}

// Has returns true if key is found.
func (d *Dictionary) Has(key string) (bool, error) {
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()
	var npointer prim.Usize = 0

	status := fastlyDictionaryGet(
		d.h,
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
