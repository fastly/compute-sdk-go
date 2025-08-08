//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package fastly

import (
	"bytes"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// (module $fastly_shielding

// witx:
//
//	(@interface func (export "shield_info")
//	  (param $name string)
//	  (param $info_block (@witx pointer (@witx char8)))
//	  (param $info_block_max_len (@witx usize))
//	  (result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_shielding shield_info
//go:noescape
func fastlyShieldingShieldingInfo(
	name prim.Pointer[prim.U8], nameLen prim.Usize,
	bufPtr prim.Pointer[prim.Char8], bufLen prim.Usize,
	bufLenOut prim.Pointer[prim.Usize],
) FastlyStatus

func ShieldingShieldInfo(name string) (*ShieldInfo, error) {

	n := prim.NewReadBufferFromString(name).Wstring()
	buf := prim.NewWriteBuffer(DefaultMediumBufLen)

	if err := fastlyShieldingShieldingInfo(
		n.Data, n.Len,
		prim.ToPointer(buf.Char8Pointer()), buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return nil, err
	}

	// split on null bytes
	vals := bytes.Split(buf.AsBytes(), []byte{0})

	if len(vals) == 0 {
		return nil, FastlyStatusBadAlign.toError()
	}

	var info ShieldInfo

	if len(vals[0]) == 0 {
		info.me = true
	}

	info.target = string(vals[1])
	info.sslTarget = string(vals[2])

	return &info, nil
}

// witx:
//
// (@interface func (export "backend_for_shield")
//
//	  (param $shield_name string)
//	  (param $backend_config_mask $shield_backend_options)
//	  (param $backend_configuration (@witx pointer $shield_backend_config))
//	  (param $backend_name_out (@witx pointer (@witx char8)))
//	  (param $backend_name_max_len (@witx usize))
//	  (result $err (expected $num_bytes (error $fastly_status)))
//	)
//
//go:wasmimport fastly_shielding backend_for_shield
//go:noescape
func fastlyShieldingBackendForShield(
	name prim.Pointer[prim.Char8], nameLen prim.Usize,
	mask shieldingBackendOptionsMask,
	opts prim.Pointer[shieldingBackendOptions],
	bufPtr prim.Pointer[prim.Char8], bufLen prim.Usize,
	bufLenOut prim.Pointer[prim.Usize],
) FastlyStatus

func ShieldingBackendForShield(name string, opts ShieldingBackendOptions) (backend string, err error) {

	n := prim.NewReadBufferFromString(name)

	buf := prim.NewWriteBuffer(DefaultMediumBufLen)

	if err := fastlyShieldingBackendForShield(
		prim.ToPointer(n.Char8Pointer()), n.Len(),
		opts.mask, prim.ToPointer(&opts.opts),
		prim.ToPointer(buf.Char8Pointer()),
		buf.Cap(),
		prim.ToPointer(buf.NPointer()),
	).toError(); err != nil {
		return "", err

	}

	return buf.ToString(), nil
}
