//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
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
