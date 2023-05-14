package abi

import (
	"reflect"
	"unsafe"
)

type SliceHeader struct {
	Data     unsafe.Pointer
	Len, Cap int
}

const intSizeBytes = (32 << (^uint(0) >> 63)) / 8

type rtype struct {
	size uintptr
	_    [24 + 2*intSizeBytes]byte
}

type SliceType struct {
	_    [24 + 3*intSizeBytes]byte
	elem *rtype
}

type Slice struct {
	typ *SliceType
	ptr *SliceHeader
}

type value struct {
	typ *rtype
	ptr unsafe.Pointer
}

func SliceFromRV(v reflect.Value) *Slice {
	return (*Slice)(unsafe.Pointer(&v))
}

func (s *Slice) SetAt(index int, v reflect.Value) {
	if uint(index) >= uint(s.ptr.Len) {
		panic("reflect: slice index out of range")
	}

	element := unsafe.Add(s.ptr.Data, uintptr(index)*s.typ.elem.size)

	*(*unsafe.Pointer)(element) = (*value)(unsafe.Pointer(&v)).ptr
}
