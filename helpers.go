package di

import (
	"github.com/goccy/go-reflect"
	"unsafe"
)

func typeIndirect(p reflect.Type) reflect.Type {
	if p.Kind() == reflect.Ptr {
		return p.Elem()
	}

	return p
}

func typeId(p reflect.Type) uintptr {
	return uintptr(unsafe.Pointer(p))
}

func idType(p uintptr) reflect.Type {
	return reflect.Type(unsafe.Pointer(p))
}

func in[T comparable](needle T, haystack []T) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}
