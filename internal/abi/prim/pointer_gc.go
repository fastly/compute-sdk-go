//go:build !tinygo

package prim

import "unsafe"

// Pointer is the type of a pointer passed to a hostcall. With Go it is a
// uint32, as pointers are 8 bytes on GOARCH=wasm and the host expects 4.
type Pointer[_ any] uint32

// ToPointer turns an arbitrary pointer into a Pointer
func ToPointer[T any](ptr *T) Pointer[T] {
	return Pointer[T](uintptr(unsafe.Pointer(ptr)))
}

// NullPointer makes a null pointer to a byte buffer.
func NullChar8Pointer() Pointer[Char8] {
	return Pointer[Char8](uintptr(unsafe.Pointer(nil)))
}

func sliceDataPointer(b []byte) Pointer[U8] {
	return Pointer[U8](uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}
