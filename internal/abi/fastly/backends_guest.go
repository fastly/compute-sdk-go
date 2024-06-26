//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//	(@interface func (export "register_dynamic_backend")
//		(param $name_prefix string)
//		(param $target string)
//		(param $backend_config_mask $backend_config_options)
//		(param $backend_configuration (@witx pointer $dynamic_backend_config))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_http_req register_dynamic_backend
//go:noescape
func fastlyRegisterDynamicBackend(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	targetData prim.Pointer[prim.U8], targetLen prim.Usize,
	mask backendConfigOptionsMask,
	opts prim.Pointer[backendConfigOptions],
) FastlyStatus

func RegisterDynamicBackend(name string, target string, opts *BackendConfigOptions) error {
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()
	targetBuffer := prim.NewReadBufferFromString(target).Wstring()

	if err := fastlyRegisterDynamicBackend(
		nameBuffer.Data, nameBuffer.Len,
		targetBuffer.Data, targetBuffer.Len,
		opts.mask,
		prim.ToPointer(&opts.opts),
	).toError(); err != nil {
		return err
	}
	return nil
}

// witx:
//
//	(module $fastly_backend
//		(@interface func (export "exists")
//			(param $backend string)
//			(result $err (expected
//			$backend_exists
//			(error $fastly_status)))
//		)
//
//go:wasmimport fastly_backend exists
//go:noescape
func fastlyBackendExists(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	exists prim.Pointer[prim.U32],
) FastlyStatus

func BackendExists(name string) (bool, error) {
	var exists prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendExists(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&exists),
	).toError(); err != nil {
		return false, err
	}
	return exists != 0, nil
}

// witx:
//
//	(@interface func (export "is_healthy")
//		(param $backend string)
//		(result $err (expected
//		$backend_health
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend is_healthy
//go:noescape
func fastlyBackendIsHealthy(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	healthy prim.Pointer[prim.U32],
) FastlyStatus

func BackendIsHealthy(name string) (BackendHealth, error) {
	var health prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsHealthy(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&health),
	).toError(); err != nil {
		return BackendHealthUnknown, err
	}
	return BackendHealth(health), nil
}

// witx:
//
//	(@interface func (export "is_dynamic")
//		(param $backend string)
//		(result $err (expected
//		is_dyanmic
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend is_dynamic
//go:noescape
func fastlyBackendIsDynamic(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	dynamic prim.Pointer[prim.U32],
) FastlyStatus

func BackendIsDynamic(name string) (bool, error) {
	var dynamic prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsDynamic(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&dynamic),
	).toError(); err != nil {
		return false, err
	}
	return dynamic != 0, nil
}

// witx:
//
//	(@interface func (export "get_host")
//		(param $backend string)
//		(param $value (@witx pointer (@witx char8)))
//		(param $value_max_len (@witx usize))
//		(param $nwritten_out (@witx pointer (@witx usize)))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_host
//go:noescape
func fastlyBackendGetHost(nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	host prim.Pointer[prim.Char8],
	hostLen prim.Usize,
	hostWritten prim.Pointer[prim.Usize],
) FastlyStatus

func BackendGetHost(name string) (string, error) {
	buf := prim.NewWriteBuffer(dnsBufLen) // Longest (255) by DNS RFCs https://datatracker.ietf.org/doc/html/rfc2181#section-11

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetHost(
		nameBuffer.Data, nameBuffer.Len,

		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "get_override_host")
//		(param $backend string)
//		(param $value (@witx pointer (@witx char8)))
//		(param $value_max_len (@witx usize))
//		(param $nwritten_out (@witx pointer (@witx usize)))
//		(result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_override_host
//go:noescape
func fastlyBackendGetOverrideHost(nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	host prim.Pointer[prim.Char8],
	hostLen prim.Usize,
	hostWritten prim.Pointer[prim.Usize],
) FastlyStatus

func BackendGetOverrideHost(name string) (string, error) {
	buf := prim.NewWriteBuffer(dnsBufLen) // Longest (255) by DNS RFCs https://datatracker.ietf.org/doc/html/rfc2181#section-11

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetOverrideHost(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return "", err
	}

	return buf.ToString(), nil
}

// witx:
//
//	(@interface func (export "get_port")
//		(param $backend string)
//		(result $err (expected
//		$port
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_port
//go:noescape
func fastlyBackendGetPort(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	port prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetPort(name string) (int, error) {
	var port prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetPort(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&port),
	).toError(); err != nil {
		return 0, err
	}
	return int(port), nil
}

// witx:
//
//	(@interface func (export "get_connect_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_connect_timeout_ms
//go:noescape
func fastlyBackendGetConnectTimeoutMs(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	timeout prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetConnectTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetConnectTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "get_first_byte_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_first_byte_timeout_ms
//go:noescape
func fastlyBackendGetFirstByteTimeoutMs(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	timeout prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetFirstByteTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetFirstByteTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "get_between_bytes_timeout_ms")
//		(param $backend string)
//		(result $err (expected
//		$timeout_ms
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_between_bytes_timeout_ms
//go:noescape
func fastlyBackendGetBetweenBytesTimeoutMs(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	timeout prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetBetweenBytesTimeout(name string) (time.Duration, error) {
	var timeout prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetBetweenBytesTimeoutMs(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&timeout),
	).toError(); err != nil {
		return 0, err
	}
	return time.Duration(time.Duration(timeout) * time.Millisecond), nil
}

// witx:
//
//	(@interface func (export "is_ssl")
//		(param $backend string)
//		(result $err (expected
//		is_ssl
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend is_ssl
//go:noescape
func fastlyBackendIsSSL(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	ssl prim.Pointer[prim.U32],
) FastlyStatus

func BackendIsSSL(name string) (bool, error) {
	var ssl prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendIsSSL(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&ssl),
	).toError(); err != nil {
		return false, err
	}
	return ssl != 0, nil
}

// witx:
//
//	(@interface func (export "get_ssl_min_version")
//		(param $backend string)
//		(result $err (expected
//		$tls_version
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_ssl_min_version
//go:noescape
func fastlyBackendGetSSLMinVersion(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	version prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetSSLMinVersion(name string) (TLSVersion, error) {
	var version prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetSSLMinVersion(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&version),
	).toError(); err != nil {
		return 0, err
	}
	return TLSVersion(version), nil
}

// witx:
//
//	(@interface func (export "get_ssl_max_version")
//		(param $backend string)
//		(result $err (expected
//		$tls_version
//		(error $fastly_status)))
//	)
//
//go:wasmimport fastly_backend get_ssl_max_version
//go:noescape
func fastlyBackendGetSSLMaxVersion(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	version prim.Pointer[prim.U32],
) FastlyStatus

func BackendGetSSLMaxVersion(name string) (TLSVersion, error) {
	var version prim.U32
	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyBackendGetSSLMaxVersion(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&version),
	).toError(); err != nil {
		return 0, err
	}
	return TLSVersion(version), nil
}
