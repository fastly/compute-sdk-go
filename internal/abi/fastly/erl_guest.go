//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
// (@interface func (export "check_rate")
//
//	(param $rc string)
//	(param $entry string)
//	(param $delta u32)
//	(param $window u32)
//	(param $limit u32)
//	(param $pb string)
//	(param $ttl u32)
//
//	(result $err (expected $blocked (error $fastly_status)))
//
// )
//
//go:wasmimport fastly_erl check_rate
//go:noescape
func fastlyERLCheckRate(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	delta prim.U32,
	window prim.U32,
	limit prim.U32,
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	ttl prim.U32,
	blocked prim.Pointer[prim.U32],
) FastlyStatus

func ERLCheckRate(rateCounter, entry string, delta uint32, window RateWindow, limit uint32, penaltyBox string, ttl time.Duration) (bool, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var blocked prim.U32

	if err := fastlyERLCheckRate(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(delta),
		prim.U32(window.value),
		prim.U32(limit),
		pbBuffer.Data, pbBuffer.Len,
		prim.U32(ttl.Seconds()),
		prim.ToPointer(&blocked),
	).toError(); err != nil {
		return false, err
	}

	return blocked != 0, nil
}

// witx:
//
//	(@interface func (export "ratecounter_increment")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $delta u32)
//
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_increment
//go:noescape
func fastlyERLRateCounterIncrement(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	delta prim.U32,
) FastlyStatus

func RateCounterIncrement(rateCounter, entry string, delta uint32) error {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	return fastlyERLRateCounterIncrement(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(delta),
	).toError()
}

// witx:
//
//	(@interface func (export "ratecounter_lookup_rate")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $window u32)
//
//	    (result $err (expected $rate (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_lookup_rate
//go:noescape
func fastlyERLRateCounterLookupRate(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	window prim.U32,
	rate prim.Pointer[prim.U32],
) FastlyStatus

func RateCounterLookupRate(rateCounter, entry string, window RateWindow) (uint32, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var rate prim.U32

	if err := fastlyERLRateCounterLookupRate(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(window.value),
		prim.ToPointer(&rate),
	).toError(); err != nil {
		return 0, err
	}

	return uint32(rate), nil
}

// witx:
//
//	(@interface func (export "ratecounter_lookup_count")
//	    (param $rc string)
//	    (param $entry string)
//	    (param $duration u32)
//
//	    (result $err (expected $count (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl ratecounter_lookup_count
//go:noescape
func fastlyERLRateCounterLookupCount(
	rcData prim.Pointer[prim.U8], rcLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	duration prim.U32,
	count prim.Pointer[prim.U32],
) FastlyStatus

func RateCounterLookupCount(rateCounter, entry string, duration CounterDuration) (uint32, error) {
	rcBuffer := prim.NewReadBufferFromString(rateCounter).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var count prim.U32

	if err := fastlyERLRateCounterLookupCount(
		rcBuffer.Data, rcBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(duration.value),
		prim.ToPointer(&count),
	).toError(); err != nil {
		return 0, err
	}

	return uint32(count), nil
}

// witx:
//
//	(@interface func (export "penaltybox_add")
//	    (param $pb string)
//	    (param $entry string)
//	    (param $ttl u32)
//
//	    (result $err (expected (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl penaltybox_add
//go:noescape
func fastlyERLPenaltyBoxAdd(
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	ttl prim.U32,
) FastlyStatus

func PenaltyBoxAdd(penaltyBox, entry string, ttl time.Duration) error {
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	return fastlyERLPenaltyBoxAdd(
		pbBuffer.Data, pbBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.U32(ttl.Seconds()),
	).toError()
}

// witx:
//
//	(@interface func (export "penaltybox_has")
//	    (param $pb string)
//	    (param $entry string)
//
//	    (result $err (expected $has (error $fastly_status)))
//	)
//
//go:wasmimport fastly_erl penaltybox_has
//go:noescape
func fastlyERLPenaltyBoxHas(
	pbData prim.Pointer[prim.U8], pbLen prim.Usize,
	entryData prim.Pointer[prim.U8], entryLen prim.Usize,
	has prim.Pointer[prim.U32],
) FastlyStatus

func PenaltyBoxHas(penaltyBox, entry string) (bool, error) {
	pbBuffer := prim.NewReadBufferFromString(penaltyBox).Wstring()
	entryBuffer := prim.NewReadBufferFromString(entry).Wstring()

	var has prim.U32

	if err := fastlyERLPenaltyBoxHas(
		pbBuffer.Data, pbBuffer.Len,
		entryBuffer.Data, entryBuffer.Len,
		prim.ToPointer(&has),
	).toError(); err != nil {
		return false, err
	}

	return has != 0, nil
}
