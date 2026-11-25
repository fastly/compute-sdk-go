//go:build tinygo

package prim

import "unsafe"

// Pointer is the type of a pointer passed to a hostcall. TinyGo moves locals
// converted to a uintptr to the heap, so it is an unsafe.Pointer (4 bytes).
type Pointer[_ any] unsafe.Pointer

// ToPointer turns an arbitrary pointer into a Pointer
func ToPointer[T any](ptr *T) Pointer[T] {
	return Pointer[T](unsafe.Pointer(ptr))
}

// NullPointer makes a null pointer to a byte buffer.
func NullChar8Pointer() Pointer[Char8] {
	return nil
}

func sliceDataPointer(b []byte) Pointer[U8] {
	return Pointer[U8](unsafe.Pointer(unsafe.SliceData(b)))
}
