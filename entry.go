package di

import (
	"github.com/Sanchous98/go-di/abi"
	"reflect"
	"sync/atomic"
)

type entry struct {
	resolved any
	resolver func(*serviceContainer) any
	types    []uintptr
	tags     []string
	built    atomic.Bool
}

func (e *entry) AddTags(tags ...string) { e.tags = append(e.tags, tags...) }
func (e *entry) AddType(_type uintptr)  { e.types = append(e.types, _type) }
func (e *entry) TypeOf(_type uintptr) bool {
	for _, t := range e.types {
		if t == _type {
			return true
		}
	}

	return false
}
func (e *entry) HasTag(tag string) bool {
	for _, t := range e.tags {
		if t == tag {
			return true
		}
	}

	return false
}

func (e *entry) Build(c *serviceContainer) any {
	for _, item := range c.buildingStack {
		if e == item {
			return item.resolved
		}
	}

	c.buildingStack.Push(e)

	if e.built.CompareAndSwap(false, true) {
		e.resolved = e.resolver(c)

		switch e.resolved.(type) {
		case Constructable:
			e.resolved.(Constructable).Constructor()
		}
	}

	c.buildingStack = c.buildingStack[:len(c.buildingStack)-1]

	return e.resolved
}

func (e *entry) Destroy() {
	switch e.resolved.(type) {
	case Destructible:
		e.resolved.(Destructible).Destructor()
	}
	e.resolved = nil
	e.built.Store(false)
}

func defaultBuilder(e *entry, service any, c *serviceContainer) any {
	e.resolved = service

	s := reflect.Indirect(reflect.ValueOf(service))

	if s.Type().Kind() == reflect.Interface {
		return c.Get(s.Type())
	}

	for i := 0; i < s.NumField(); i++ {
		tags := abi.FromRV(s).Fields[i].Tag()
		field := s.Field(i)
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
		tag, ok := tags.Lookup(injectTag)

		if !ok {
			continue
		}

		if len(tag) > 0 {
			if field.Type().Kind() != reflect.Slice {
				panic("tagged field must be slice")
			}

			var count int
			for _, item := range c.entries {
				if item.HasTag(tag) {
					count++
				}
			}

			if count > 0 {
				var j int

				if field.IsNil() || field.Cap() < count {
					field.Set(reflect.MakeSlice(field.Type(), count, count))
				}

				_t := field.Type().Elem()

				for _, item := range c.entries {
					if item.HasTag(tag) {
						if _t.Kind() == reflect.Ptr || _t.Kind() == reflect.Interface {
							field.Index(j).Set(reflect.ValueOf(item.Build(c)))
						} else {
							field.Index(j).Set(reflect.ValueOf(item.Build(c)).Elem())
						}
						j++
					}
				}
			}

			continue
		}

		if !c.Has(field.Type()) {
			item := &entry{types: []uintptr{valueTypeId(field.Type())}}
			item.resolver = func(c *serviceContainer) any {
				newService := reflect.New(typeIndirect(field.Type()))

				if field.Type().Kind() == reflect.Interface {
					return newService.Elem().Interface()
				}

				return defaultBuilder(item, newService.Interface(), c)
			}
			c.entries = append(c.entries, item)
		}

		newService := c.Get(field.Type())

		switch field.Type().Kind() {
		case reflect.Interface:
			if newService == nil {
				panic(`interface type without bound value. Remove "inject" tag or set a value, bound by this interface type`)
			}

			fallthrough
		case reflect.Ptr:
			field.Set(reflect.ValueOf(newService))
		default:
			field.Set(reflect.ValueOf(newService).Elem())
		}
	}

	return service
}
