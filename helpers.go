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

func valueTypeId(_type any) (serviceType uintptr) {
	switch _type.(type) {
	case uintptr:
		serviceType = _type.(uintptr)
	case reflect.Type:
		serviceType = typeId(typeIndirect(_type.(reflect.Type)))
	default:
		serviceType = typeId(typeIndirect(reflect.TypeOf(_type)))
	}

	return
}
