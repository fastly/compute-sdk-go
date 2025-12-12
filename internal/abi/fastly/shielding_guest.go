//go:build wasip1 && !nofastlyhostcalls

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
	buf, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyShieldingShieldingInfo(
			n.Data, n.Len,
			prim.ToPointer(buf.Char8Pointer()), buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}

	bufb := buf.AsBytes()

	if len(bufb) == 0 {
		return nil, FastlyStatusBadAlign.toError()
	}

	// block format:
	// 0x1 (running on shield)
	// 0x0 0x0 // no targets, but still have field separators)
	// OR
	// 0x0 (not running on shield)
	// <unencrypted target> 0x0
	// <encrypted target> 0x0

	var info ShieldInfo
	info.me = bufb[0] == 1

	if !info.me {
		// strip first null byte and extract targets
		vals := bytes.Split(bufb[1:], []byte{0})

		// ensure we got what we expected
		if len(vals) != 3 {
			return nil, FastlyStatusBadAlign.toError()
		}

		info.target = string(vals[0])
		info.sslTarget = string(vals[1])
	}

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

func ShieldingBackendForShield(name string, opts *ShieldingBackendOptions) (backend string, err error) {

	n := prim.NewReadBufferFromString(name)
	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyShieldingBackendForShield(
			prim.ToPointer(n.Char8Pointer()), n.Len(),
			opts.mask, prim.ToPointer(&opts.opts),
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return "", err
	}
	return value.ToString(), nil
}
