//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//   (module $fastly_object_store
//	   (@interface func (export "open")
//	     (param $name string)
//	     (result $err (expected $object_store_handle (error $fastly_status)))
//	  )

//go:wasmimport fastly_object_store open
//go:noescape
func fastlyObjectStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[objectStoreHandle],
) FastlyStatus

// objectStore represents a Fastly kv store, a collection of key/value pairs.
// For convenience, keys and values are both modelled as Go strings.
type KVStore struct {
	h objectStoreHandle
}

// KVStoreOpen returns a reference to the named kv store, if it exists.
func OpenKVStore(name string) (*KVStore, error) {
	var o KVStore

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyObjectStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&o.h),
	).toError(); err != nil {
		return nil, err
	}

	return &o, nil
}

// witx:
//
//   (@interface func (export "lookup")
//	   (param $store $object_store_handle)
//	   (param $key string)
//	   (param $body_handle_out (@witx pointer $body_handle))
//	   (result $err (expected (error $fastly_status)))
//	)

//go:wasmimport fastly_object_store lookup
//go:noescape
func fastlyObjectStoreLookup(
	h objectStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	b prim.Pointer[bodyHandle],
) FastlyStatus

// Lookup returns the value for key, if it exists.
func (o *KVStore) Lookup(key string) (io.Reader, error) {
	body := HTTPBody{h: invalidBodyHandle}

	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	if err := fastlyObjectStoreLookup(
		o.h,
		keyBuffer.Data, keyBuffer.Len,
		prim.ToPointer(&body.h),
	).toError(); err != nil {
		return nil, err
	}

	// Didn't get a valid handle back.  This means there was no key
	// with that name.  Report this to the caller by returning `None`.
	if body.h == invalidBodyHandle {
		return nil, FastlyError{Status: FastlyStatusNone}
	}

	return &body, nil
}

// witx:
//
//  (@interface func (export "insert")
//	  (param $store $object_store_handle)
//	  (param $key string)
//	  (param $body_handle $body_handle)
//	  (result $err (expected (error $fastly_status)))
//	)

//go:wasmimport fastly_object_store insert
//go:noescape
func fastlyObjectStoreInsert(
	h objectStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	b bodyHandle,
) FastlyStatus

// Insert adds a key/value pair to the kv store.
func (o *KVStore) Insert(key string, value io.Reader) error {
	body, err := NewHTTPBody()
	if err != nil {
		return err
	}

	if _, err := io.Copy(body, value); err != nil {
		return err
	}

	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	if err := fastlyObjectStoreInsert(
		o.h,
		keyBuffer.Data, keyBuffer.Len,
		body.h,
	).toError(); err != nil {
		return err
	}

	return nil
}

// witx:
//
//  (@interface func (export "delete_async")
//      (param $store $object_store_handle)
//      (param $key string)
//      (param $pending_handle_out (@witx pointer $pending_kv_delete_handle))
//      (result $err (expected (error $fastly_status)))
//  )

//go:wasmimport fastly_object_store delete_async
//go:noescape
func fastlyObjectStoreDeleteAsync(
	h objectStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	pendingReq prim.Pointer[pendingRequestHandle],
) FastlyStatus

// witx:
//
//  (@interface func (export "pending_delete_wait")
//      (param $pending_handle $pending_kv_delete_handle)
//      (result $err (expected (error $fastly_status)))
//  )

//go:wasmimport fastly_object_store pending_delete_wait
//go:noescape
func fastlyObjectStorePendingDeleteWait(
	pendingReq pendingRequestHandle,
) FastlyStatus

// Delete removes a key from the kv store.
func (o *KVStore) Delete(key string) error {
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	var handle pendingRequestHandle
	if err := fastlyObjectStoreDeleteAsync(
		o.h,
		keyBuffer.Data, keyBuffer.Len,
		prim.ToPointer(&handle),
	).toError(); err != nil {
		return err
	}

	return fastlyObjectStorePendingDeleteWait(handle).toError()
}
