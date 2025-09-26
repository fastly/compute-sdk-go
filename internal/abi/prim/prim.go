// Copyright 2022 Fastly, Inc.

package prim

import (
	"unsafe"
)

// Usize is an unsigned integer who's size is based on the system architecture.
type Usize uint32

// Char8 is an unsigned 8 bit integer.
type Char8 uint8

// U8 is an unsigned 8 bit integer.
type U8 uint8

// U16 is an unsigned 16 bit integer.
type U16 uint16

// U32 is an unsigned 32 bit integer.
type U32 uint32

// U64 is an unsigned 64 bit integer.
type U64 uint64

// Pointer is the type of a pointer
type Pointer[_ any] uint32

// ToPointer turns an arbitrary pointer into a Pointer
func ToPointer[T any](ptr *T) Pointer[T] {
	return Pointer[T](uintptr(unsafe.Pointer(ptr)))
}

// NullPointer makes a null pointer to a byte buffer.
func NullChar8Pointer() Pointer[Char8] {
	return Pointer[Char8](uintptr(unsafe.Pointer(nil)))
}

// Wstring is a header for a string.
type Wstring struct {
	Data Pointer[U8]
	Len  Usize
}

func NewWstringFromChar8(p Pointer[Char8], n U32) Wstring {
	return Wstring{
		Data: (Pointer[U8])(p),
		Len:  Usize(n),
	}
}

func (w Wstring) String() string {
	return unsafe.String(*(**byte)(unsafe.Pointer(&w.Data)), w.Len)
}

// ArrayU8 is a header for an array of U8.
type ArrayU8 Wstring

// ArrayChar8 is a header for an array of Char8.
type ArrayChar8 Wstring

// WriteBuffer provides some memory that hostcalls can write into.
//
// Technically, Go's GC is permitted to move memory around whenever it wants
// (with a few exceptions). This is normally safe, because it updates references
// to that memory at the same time. But unsafe.Pointer isn't understood by the
// GC as a reference, which means that our usage here is technically unsafe: if
// the GC moved the buffer around during a hostcall, the hostcall would end up
// writing to an invalid location.
//
// This works fine, though, because hostcalls only happen under +build tinygo,
// and all of the GC implementations provided by TinyGo don't do any of that
// fancy stuff. But it's definitely a risk we need to be aware of when upgrading
// TinyGo in the future.
type WriteBuffer struct {
	buf []byte
	n   Usize
}

// NewWriteBuffer creates a new WriteBuffer with the provided capacity.
func NewWriteBuffer(cap int) *WriteBuffer {
	return NewWriteBufferFromBytes(make([]byte, 0, cap))
}

// NewWriteBufferFromBytes creates a new WriteBuffer with the provided byte
// slice used as its buffer.
func NewWriteBufferFromBytes(buf []byte) *WriteBuffer {
	return &WriteBuffer{buf: buf}
}

// Char8Pointer returns a pointer to the buffer's data as a Char8.
func (b *WriteBuffer) Char8Pointer() *Char8 {
	return (*Char8)(unsafe.SliceData(b.buf))
}

// U8Pointer returns a pointer to the buffer's data as a U8.
func (b *WriteBuffer) U8Pointer() *U8 {
	return (*U8)(unsafe.SliceData(b.buf))
}

// Cap returns the capacity of the buffer as a Usize.
func (b *WriteBuffer) Cap() Usize {
	return Usize(cap(b.buf))
}

// Len returns the length of data in the buffer as a Usize.
func (b *WriteBuffer) Len() Usize {
	return Usize(len(b.buf))
}

// NPointer returns a pointer to the number of bytes written to the buffer as a
// Usize.
func (b *WriteBuffer) NPointer() *Usize {
	return &b.n
}

// NValue returns the number of bytes written to the buffer as a Usize.
func (b *WriteBuffer) NValue() Usize {
	return b.n
}

// AsBytes returns a slice of the buffer's data as a byte slice.
func (b *WriteBuffer) AsBytes() []byte {
	return b.buf[:b.n:b.n]
}

// ToString returns a copy of the buffer's data as a string.
func (b *WriteBuffer) ToString() string {
	return string(b.AsBytes())
}

// ReadBuffer is like WriteBuffer, but only allows hostcalls to read the
// underlying memory via a smaller, more restricted API.
type ReadBuffer struct {
	buf []byte
}

// NewReadBufferFromString creates a ReadBuffer with its buffer based on the
// provided string.
func NewReadBufferFromString(s string) *ReadBuffer {
	return NewReadBufferFromBytes([]byte(s))
}

// NewReadBufferFromBytes creates a new ReadBuffer with the provided byte slice
// used as its buffer.
func NewReadBufferFromBytes(buf []byte) *ReadBuffer {
	return &ReadBuffer{buf: buf}
}

// Wstring returns the buffers data as a Wstring.
func (b *ReadBuffer) Wstring() Wstring {
	return Wstring{
		Data: Pointer[U8](uintptr(unsafe.Pointer(unsafe.SliceData(b.buf)))),
		Len:  Usize(len(b.buf)),
	}
}

// ArrayU8 returns the buffers data as a ArrayU8.
func (b *ReadBuffer) ArrayU8() ArrayU8 {
	return ArrayU8{
		Data: Pointer[U8](uintptr(unsafe.Pointer(unsafe.SliceData(b.buf)))),
		Len:  Usize(len(b.buf)),
	}
}

// ArrayChar8 returns the buffers data as a ArrayChar8.
func (b *ReadBuffer) ArrayChar8() ArrayChar8 {
	return ArrayChar8{
		Data: Pointer[U8](uintptr(unsafe.Pointer(unsafe.SliceData(b.buf)))),
		Len:  Usize(len(b.buf)),
	}
}

// Char8Pointer returns a pointer to the buffer's data as a Char8.
func (b *ReadBuffer) Char8Pointer() *Char8 {
	return (*Char8)(unsafe.Pointer(unsafe.SliceData(b.buf)))
}

// U8Pointer returns a pointer to the buffer's data as a U8.
func (b *ReadBuffer) U8Pointer() *U8 {
	return (*U8)(unsafe.Pointer(unsafe.SliceData(b.buf)))
}

// Len returns the length of data in the buffer as a Usize.
func (b *ReadBuffer) Len() Usize {
	return Usize(len(b.buf))
}
