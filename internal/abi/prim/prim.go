// Copyright 2022 Fastly, Inc.

package prim

import (
	"reflect"
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

// Wstring is a header for a string.
type Wstring struct {
	Data uint32
	Len  uint32
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
	hdr *reflect.SliceHeader
	n   Usize
}

// NewWriteBuffer creates a new WriteBuffer with the provided capacity.
func NewWriteBuffer(cap int) *WriteBuffer {
	return NewWriteBufferFromBytes(make([]byte, 0, cap))
}

// NewWriteBufferFromBytes creates a new WriteBuffer with the provided byte
// slice used as its buffer.
func NewWriteBufferFromBytes(buf []byte) *WriteBuffer {
	b := &WriteBuffer{buf: buf}                            // copy the slice header into our struct
	b.hdr = (*reflect.SliceHeader)(unsafe.Pointer(&b.buf)) // point to our copy of the slice header
	return b
}

// Char8Pointer returns a pointer to the buffer's data as a Char8.
func (b *WriteBuffer) Char8Pointer() *Char8 {
	return (*Char8)(unsafe.Pointer(b.hdr.Data))
}

// U8Pointer returns a pointer to the buffer's data as a Char8.
func (b *WriteBuffer) U8Pointer() *U8 {
	return (*U8)(unsafe.Pointer(b.hdr.Data))
}

// Cap returns the capacity of the buffer as a Usize.
func (b *WriteBuffer) Cap() Usize {
	return Usize(cap(b.buf))
}

// Len returns the length of data in the buffer as a Usize.
func (b *WriteBuffer) Len() Usize {
	return Usize(len(b.buf))
}

// NPointer returns a pointer to the number of bytes written to th buffer as a
// Usize.
func (b *WriteBuffer) NPointer() *Usize {
	return &b.n
}

// NValue returns the number of bytes written to th buffer as a Usize.
func (b *WriteBuffer) NValue() Usize {
	return b.n
}

// AsBytes returns a copy of the buffer's data as a byte slice.
func (b *WriteBuffer) AsBytes() []byte {
	return b.buf[:b.n]
}

// ToString returns a copy of the buffer's data as a string.
func (b *WriteBuffer) ToString() string {
	return string(b.AsBytes())
}

// ReadBuffer is like WriteBuffer, but only allows hostcalls to read the
// underlying memory via a smaller, more restricted API.
type ReadBuffer struct {
	buf []byte
	hdr *reflect.SliceHeader
}

// NewReadBufferFromString creates a ReadBuffer with its buffer based on the
// provided string.
func NewReadBufferFromString(s string) *ReadBuffer {
	return NewReadBufferFromBytes([]byte(s))
}

// NewReadBufferFromBytes creates a new ReadBuffer with the provided byte slice
// used as its buffer.
func NewReadBufferFromBytes(buf []byte) *ReadBuffer {
	b := &ReadBuffer{buf: buf}
	b.hdr = (*reflect.SliceHeader)(unsafe.Pointer(&b.buf))
	return b
}

// Wstring returns the buffers data as a Wstring.
func (b *ReadBuffer) Wstring() Wstring {
	return Wstring{
		Data: uint32(b.hdr.Data),
		Len:  uint32(b.hdr.Len),
	}
}

// ArrayU8 returns the buffers data as a ArrayU8.
func (b *ReadBuffer) ArrayU8() ArrayU8 {
	return ArrayU8{
		Data: uint32(b.hdr.Data),
		Len:  uint32(b.hdr.Len),
	}
}

// ArrayChar8 returns the buffers data as a ArrayChar8.
func (b *ReadBuffer) ArrayChar8() ArrayChar8 {
	return ArrayChar8{
		Data: uint32(b.hdr.Data),
		Len:  uint32(b.hdr.Len),
	}
}

// Char8Pointer returns a pointer to the buffer's data as a Char8.
func (b *ReadBuffer) Char8Pointer() *Char8 {
	return (*Char8)(unsafe.Pointer(b.hdr.Data))
}

// Len returns the length of data in the buffer as a Usize.
func (b *ReadBuffer) Len() Usize {
	return Usize(len(b.buf))
}
