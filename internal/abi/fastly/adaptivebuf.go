//go:build wasip1 && !nofastlyhostcalls

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

// withAdaptiveBuffer is a helper function that calls the provided function with
// an initial size, and repeats the call with the indicated buffer size when
// initSize is exceeded by the value.
func withAdaptiveBuffer(initSize int, f func(buf *prim.WriteBuffer) FastlyStatus) (*prim.WriteBuffer, error) {
	n := initSize
	for {
		buf := prim.NewWriteBuffer(n)
		status := f(buf)
		if status == FastlyStatusBufLen && buf.NValue() > 0 {
			n = int(buf.NValue())
			continue
		}
		if err := status.toError(); err != nil {
			return nil, err
		}
		return buf, nil
	}
}
