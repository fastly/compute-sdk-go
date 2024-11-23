//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"io"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

//
//          (@interface func (export "list")
//              (param $store $kv_store_handle)
//              (param $list_config_mask $kv_list_config_options)
//              (param $list_configuration (@witx pointer $kv_list_config))
//              (param $handle_out (@witx pointer $kv_store_list_handle))
//              (result $err (expected (error $fastly_status)))
//          )
//
//          (@interface func (export "list_wait")
//              (param $handle $kv_store_list_handle)
//              (param $body_handle_out (@witx pointer $body_handle))
//              (param $kv_error_out (@witx pointer $kv_error))
//              (result $err (expected (error $fastly_status)))
//          )
//      )

//   (module $fastly_object_store
//	   (@interface func (export "open")
//	     (param $name string)
//	     (result $err (expected $object_store_handle (error $fastly_status)))
//	  )

// witx:
//
//      (module $fastly_kv_store
//          (@interface func (export "open")
//              (param $name string)
//              (result $err (expected $kv_store_handle (error $fastly_status)))
//          )
//

//go:wasmimport fastly_kv_store open
//go:noescape
func fastlyKVStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[kvstoreHandle],
) FastlyStatus

// KVStore represents a Fastly kv store, a collection of key/value pairs.
// For convenience, keys and values are both modelled as Go strings.
type KVStore struct {
	h kvstoreHandle
}

// KVStoreOpen returns a reference to the named kv store, if it exists.
func OpenKVStore(name string) (*KVStore, error) {
	var kv KVStore = KVStore{h: invalidKVStoreHandle}

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyKVStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&kv.h),
	).toError(); err != nil {
		return nil, err
	}

	return &kv, nil
}

// witx:
//
//          (@interface func (export "lookup")
//              (param $store $kv_store_handle)
//              (param $key string)
//              (param $lookup_config_mask $kv_lookup_config_options)
//              (param $lookup_configuration (@witx pointer $kv_lookup_config))
//              (param $handle_out (@witx pointer $kv_store_lookup_handle))
//              (result $err (expected (error $fastly_status)))
//          )
//

//go:wasmimport fastly_kv_store lookup
//go:noescape
func fastlyKVStoreLookup(
	h kvstoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	mask kvLookupConfigMask,
	config prim.Pointer[kvLookupConfig],
	lookupHandle prim.Pointer[kvstoreLookupHandle],
) FastlyStatus

// Lookup returns a handle to a pending lookup operation
func (kv *KVStore) Lookup(key string) (kvstoreLookupHandle, error) {
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	// empty
	var mask kvLookupConfigMask
	var conf kvLookupConfig

	var lookupHandle kvstoreLookupHandle = invalidKVLookupHandle

	if err := fastlyKVStoreLookup(
		kv.h,
		keyBuffer.Data, keyBuffer.Len,
		mask,
		prim.ToPointer(&conf),
		prim.ToPointer(&lookupHandle),
	).toError(); err != nil {
		return 0, err
	}

	if lookupHandle == invalidKVLookupHandle {
		return 0, FastlyStatusError.toError()
	}

	return lookupHandle, nil
}

// witx:
//
//          (@interface func (export "lookup_wait")
//              (param $handle $kv_store_lookup_handle)
//              (param $body_handle_out (@witx pointer $body_handle))
//              (param $metadata_buf (@witx pointer (@witx char8)))
//              (param $metadata_buf_len (@witx usize))
//              (param $nwritten_out (@witx pointer (@witx usize)))
//              (param $generation_out (@witx pointer u32))
//              (param $kv_error_out (@witx pointer $kv_error))
//              (result $err (expected (error $fastly_status)))
//          )

//go:wasmimport fastly_kv_store lookup_wait
//go:noescape
func fastlyKVStoreLookupWait(
	h kvstoreLookupHandle,
	b prim.Pointer[bodyHandle],
	metaData prim.Pointer[prim.U8], metaLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
	generation prim.Pointer[prim.U32],
	kvErr prim.Pointer[KVError],
) FastlyStatus

// LookupWait returns a lookup response for a pending lookup handle
func (kv *KVStore) LookupWait(lookupH kvstoreLookupHandle) (KVLookupResult, error) {
	body := HTTPBody{h: invalidBodyHandle}

	meta := prim.NewWriteBuffer(kvstoreMetadataMaxBufLen)
	var generation prim.U32

	var kvErr KVError = KVErrorUninitialized

	if err := fastlyKVStoreLookupWait(
		lookupH,
		prim.ToPointer(&body.h),
		prim.ToPointer(meta.U8Pointer()), meta.Cap(),
		prim.ToPointer(meta.NPointer()),
		prim.ToPointer(&generation),
		prim.ToPointer(&kvErr),
	).toError(); err != nil {
		return KVLookupResult{}, err
	}

	if kvErr != KVErrorOK {
		return KVLookupResult{}, kvErr
	}

	result := KVLookupResult{
		Body:       &body,
		Meta:       meta.AsBytes(),
		Generation: uint32(generation),
	}

	return result, nil
}

// witx:
//          (@interface func (export "insert")
//              (param $store $kv_store_handle)
//              (param $key string)
//              (param $body_handle $body_handle)
//              (param $insert_config_mask $kv_insert_config_options)
//              (param $insert_configuration (@witx pointer $kv_insert_config))
//              (param $handle_out (@witx pointer $kv_store_insert_handle))
//              (result $err (expected (error $fastly_status)))
//          )
//

//go:wasmimport fastly_kv_store insert
//go:noescape
func fastlyKVStoreInsert(
	h kvstoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	b bodyHandle,
	mask kvInsertConfigMask,
	config prim.Pointer[kvInsertConfig],
	insertHandle prim.Pointer[kvstoreInsertHandle],
) FastlyStatus

// Insert returns a handle to a pending key/value pair insertion.
func (k *KVStore) Insert(key string, value io.Reader) (kvstoreInsertHandle, error) {
	body, err := NewHTTPBody()
	if err != nil {
		return 0, err
	}

	if _, err := io.Copy(body, value); err != nil {
		return 0, err
	}

	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	var mask kvInsertConfigMask
	var config kvInsertConfig

	var insertHandle kvstoreInsertHandle = invalidKVInsertHandle

	if err := fastlyKVStoreInsert(
		k.h,
		keyBuffer.Data, keyBuffer.Len,
		body.h,
		mask,
		prim.ToPointer(&config),
		prim.ToPointer(&insertHandle),
	).toError(); err != nil {
		return 0, err
	}

	if insertHandle == invalidKVInsertHandle {
		return 0, FastlyStatusError.toError()
	}

	return insertHandle, nil
}

// witx:
//
//          (@interface func (export "insert_wait")
//              (param $handle $kv_store_insert_handle)
//              (param $kv_error_out (@witx pointer $kv_error))
//              (result $err (expected (error $fastly_status)))
//          )

//go:wasmimport fastly_kv_store insert_wait
//go:noescape
func fastlyKVStoreInsertWait(
	h kvstoreInsertHandle,
	kvErr prim.Pointer[KVError],
) FastlyStatus

// InsertWait returns the status of the given pending insertion handle.
func (kv *KVStore) InsertWait(insertH kvstoreInsertHandle) error {
	var kvErr KVError = KVErrorUninitialized

	if err := fastlyKVStoreInsertWait(
		insertH,
		prim.ToPointer(&kvErr),
	).toError(); err != nil {
		return err
	}

	if kvErr != KVErrorOK {
		return kvErr
	}

	return nil
}

// witx:
//
//          (@interface func (export "delete")
//              (param $store $kv_store_handle)
//              (param $key string)
//              (param $delete_config_mask $kv_delete_config_options)
//              (param $delete_configuration (@witx pointer $kv_delete_config))
//              (param $handle_out (@witx pointer $kv_store_delete_handle))
//              (result $err (expected (error $fastly_status)))
//          )

//go:wasmimport fastly_kv_store delete
//go:noescape
func fastlyKVStoreDelete(
	h kvstoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	mask kvDeleteConfigMask,
	config prim.Pointer[kvDeleteConfig],
	deleteHandle prim.Pointer[kvstoreDeleteHandle],
) FastlyStatus

// Delete returns a handle to a pending key/value removal.
func (kv *KVStore) Delete(key string) (kvstoreDeleteHandle, error) {
	keyBuffer := prim.NewReadBufferFromString(key).Wstring()

	var mask kvDeleteConfigMask
	var config kvDeleteConfig

	var deleteHandle kvstoreDeleteHandle = invalidKVDeleteHandle

	if err := fastlyKVStoreDelete(
		kv.h,
		keyBuffer.Data, keyBuffer.Len,
		mask,
		prim.ToPointer(&config),
		prim.ToPointer(&deleteHandle),
	).toError(); err != nil {
		return 0, err
	}

	if deleteHandle == invalidKVDeleteHandle {
		return 0, FastlyStatusError.toError()
	}

	return deleteHandle, nil
}

// witx:
//
//          (@interface func (export "delete_wait")
//              (param $handle $kv_store_delete_handle)
//              (param $kv_error_out (@witx pointer $kv_error))
//              (result $err (expected (error $fastly_status)))
//          )
//

//go:wasmimport fastly_kv_store delete_wait
//go:noescape
func fastlyKVStoreDeleteWait(
	h kvstoreDeleteHandle,
	kvErr prim.Pointer[KVError],
) FastlyStatus

// DeleteWait completes the pending deletion for the given handle.
func (kv *KVStore) DeleteWait(deleteH kvstoreDeleteHandle) error {
	var kvErr KVError = KVErrorUninitialized

	if err := fastlyKVStoreDeleteWait(
		deleteH,
		prim.ToPointer(&kvErr),
	).toError(); err != nil {
		return err
	}

	if kvErr != KVErrorOK {
		return kvErr
	}

	return nil
}
