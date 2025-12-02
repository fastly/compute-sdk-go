//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

// SecretStore represents a Fastly secret store, a collection of
// key/value pairs for storing sensitive data.
type SecretStore struct {
	h secretStoreHandle
}

// Secret represents a secret value.  Data is encrypted at rest, and is
// only decrypted upon the first call to the secret's Plaintext method.
type Secret struct {
	h secretHandle
}

// witx:
//
//   (module $fastly_secret_store
//	   (@interface func (export "open")
//	     (param $name string)
//	     (result $err (expected $secret_store_handle (error $fastly_status)))
//	  )

//go:wasmimport fastly_secret_store open
//go:noescape
func fastlySecretStoreOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[secretStoreHandle],
) FastlyStatus

// OpenSecretStore returns a reference to the named secret store, if it exists.
func OpenSecretStore(name string) (*SecretStore, error) {
	var st SecretStore

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlySecretStoreOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&st.h),
	).toError(); err != nil {
		return nil, err
	}

	return &st, nil
}

// witx:
//
//   (module $fastly_secret_store
//     (@interface func (export "get")
//       (param $store $secret_store_handle)
//       (param $key string)
//       (result $err (expected $secret_handle (error $fastly_status)))
//     )
//   )

//go:wasmimport fastly_secret_store get
//go:noescape
func fastlySecretStoreGet(
	h secretStoreHandle,
	keyData prim.Pointer[prim.U8], keyLen prim.Usize,
	s prim.Pointer[secretHandle],
) FastlyStatus

// Get returns a handle to the secret value for the given name, if it
// exists.
func (st *SecretStore) Get(name string) (*Secret, error) {
	var s Secret

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlySecretStoreGet(
		st.h,
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&s.h),
	).toError(); err != nil {
		return nil, err
	}

	return &s, nil
}

// witx:
//
//   (module $fastly_secret_store
//     (@interface func (export "plaintext")
//       (param $secret $secret_handle)
//       (param $buf (@witx pointer (@witx char8)))
//       (param $buf_len (@witx usize))
//       (param $nwritten_out (@witx pointer (@witx usize)))
//       (result $err (expected (error $fastly_status)))
//     )
//   )

//go:wasmimport fastly_secret_store plaintext
//go:noescape
func fastlySecretPlaintext(
	h secretHandle,
	buf prim.Pointer[prim.Char8], bufLen prim.Usize,
	nwritten prim.Pointer[prim.Usize],
) FastlyStatus

// Plaintext decrypts and returns the secret value as a byte slice.
func (s *Secret) Plaintext() ([]byte, error) {
	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlySecretPlaintext(
			s.h,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}
	return value.AsBytes(), nil
}

func (s *Secret) Handle() secretHandle {
	return s.h
}

// witx:
//
// (@interface func (export "from_bytes")
//     (param $buf (@witx pointer (@witx char8)))
//     (param $buf_len (@witx usize))
//     (result $err (expected $secret_handle (error $fastly_status)))
// )

//go:wasmimport fastly_secret_store from_bytes
//go:noescape
func fastlySecretFromBytes(
	buf prim.Pointer[prim.Char8],
	bufLen prim.Usize,
	h prim.Pointer[secretHandle],
) FastlyStatus

// FromBytes creates a secret handle for the given byte slice.  This is
// for use with APIs that require a secret handle but cannot (for
// whatever reason) use a secret store.
func SecretFromBytes(b []byte) (*Secret, error) {
	var s Secret

	buf := prim.NewReadBufferFromBytes(b)

	if err := fastlySecretFromBytes(
		prim.ToPointer(buf.Char8Pointer()),
		buf.Len(),
		prim.ToPointer(&s.h),
	).toError(); err != nil {
		return nil, err
	}

	return &s, nil
}
