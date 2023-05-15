package abi

import (
	"reflect"
	"unsafe"
)

const intSizeBytes = (32 << (^uint(0) >> 63)) / 8

type StructField struct {
	name   name    // name is always non-empty
	_      uintptr // type of field
	offset uintptr // byte offset of field
}

func (s *StructField) Name() string { return s.name.name() }

func (s *StructField) Tag() reflect.StructTag {
	if s.name.hasTag() {
		return reflect.StructTag(s.name.tag())
	}

	return ""
}

type Struct struct {
	_      [24 + 3*intSizeBytes]byte
	_      uintptr
	Fields []StructField
}

func FromRV(value reflect.Value) *Struct {
	if value.Kind() != reflect.Struct {
		panic("value must be struct")
	}

	type v struct {
		typ *Struct
		ptr unsafe.Pointer
	}

	return (*v)(unsafe.Pointer(&value)).typ
}

type name struct {
	bytes *byte
}

func (n name) data(off int, _ string) *byte {
	return (*byte)(unsafe.Add(unsafe.Pointer(n.bytes), off))
}

func (n name) hasTag() bool {
	return (*n.bytes)&(1<<1) != 0
}

func (n name) readVarint(off int) (int, int) {
	v := 0
	for i := 0; ; i++ {
		x := *n.data(off+i, "read varint")
		v += int(x&0x7f) << (7 * i)
		if x&0x80 == 0 {
			return i + 1, v
		}
	}
}

func (n name) name() string {
	if n.bytes == nil {
		return ""
	}
	i, l := n.readVarint(1)
	return unsafe.String(n.data(1+i, "non-empty string"), l)
}

func (n name) tag() string {
	if !n.hasTag() {
		return ""
	}
	i, l := n.readVarint(1)
	i2, l2 := n.readVarint(1 + i + l)
	return unsafe.String(n.data(1+i+l+i2, "non-empty string"), l2)
}
