//go:build wasip1 && !nofastlyhostcalls

// Copyright 2026 Fastly, Inc.

package fastly


// witx:
//
//	(module $fastly_resvpnproxy
//	  (@interface func (export "lookup")
//	     (param $addr_octets (@witx pointer (@witx char8)))
//	     (param $addr_len (@witx usize))
//	     (param $buf (@witx pointer (@witx char8)))
//	     (param $buf_len (@witx usize))
//	     (param $nwritten_out (@witx pointer (@witx usize)))
//	     (result $err (expected (error $fastly_status)))
//	  )
//
// )
//

