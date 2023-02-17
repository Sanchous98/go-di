package di

import (
	"github.com/goccy/go-reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type entry struct {
	resolved any
	resolver func(*serviceContainer) any
	types    []uintptr
	tags     []string

	buildOnce    sync.Once
	waitBuilding sync.Mutex
}

func defaultEntry(service any) *entry {
	e := &entry{types: []uintptr{typeId(typeIndirect(reflect.TypeOf(service)))}}
	e.resolver = func(c *serviceContainer) any { return defaultBuilder(e, service, c) }

	return e
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
	stackSize := len(c.buildingStack)

	e.buildOnce.Do(func() {
		e.resolved = e.resolver(c)

		switch e.resolved.(type) {
		case Constructable:
			e.resolved.(Constructable).Constructor()
		}
	})
	c.buildingStack = c.buildingStack[:stackSize-1]

	return e.resolved
}

func (e *entry) Destroy() {
	switch e.resolved.(type) {
	case Destructible:
		e.resolved.(Destructible).Destructor()
	}
	e.resolved = nil
	e.buildOnce = sync.Once{}
}

func defaultBuilder(e *entry, service any, c *serviceContainer) any {
	e.resolved = service

	s := reflect.Indirect(reflect.ValueNoEscapeOf(service))

	if s.Type().Kind() == reflect.Interface {
		return c.Get(s.Type())
	}

	for i := 0; i < s.NumField(); i++ {
		tags := s.Type().Field(i).Tag
		field := s.Field(i)
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		if nameAndDefault, ok := tags.Lookup(envTag); ok {
			param := strings.Split(nameAndDefault, ":-")

			switch len(param) {
			case 2:
				envVar, defaultValue := param[0], param[1]

				if v := c.GetParam(envVar); v != "" {
					fillEnvVar(field, v)
				} else {
					fillEnvVar(field, defaultValue)
				}
			case 1:
				fillEnvVar(field, c.GetParam(param[0]))
			default:
				panic("wrong parameter for env tag")
			}

			continue
		}

		if tag, ok := tags.Lookup(injectTag); ok {
			if len(tag) > 0 {
				if field.Type().Kind() != reflect.Slice {
					panic("tagged field must be slice")
				}

				field.Set(reflect.MakeSlice(field.Type(), 1, 1))
				_t := field.Index(0).Type()
				field.Set(field.Slice(0, 0))

				for _, item := range c.entries {
					if item.HasTag(tag) {
						newService := reflect.ValueNoEscapeOf(item.Build(c))
						if _t.Kind() == reflect.Ptr || _t.Kind() == reflect.Interface {
							field.Set(reflect.Append(field, newService))
						} else {
							field.Set(reflect.Append(field, newService.Elem()))
						}
					}
				}

				continue
			}

			var newService any

			if !c.Has(typeId(typeIndirect(field.Type()))) {

				item := &entry{
					types: []uintptr{typeId(typeIndirect(field.Type()))},
				}
				item.resolver = func(c *serviceContainer) any {
					newService = reflect.New(idType(typeId(typeIndirect(field.Type())))).Interface()

					if field.Type().Kind() == reflect.Interface {
						return reflect.ValueNoEscapeOf(newService).Elem().Interface()
					}

					return defaultBuilder(item, newService, c)
				}
				c.entries = append(c.entries, item)
			}

			newService = c.Get(typeId(typeIndirect(field.Type())))

			if field.Type().Kind() == reflect.Ptr || field.Type().Kind() == reflect.Interface {
				if newService != nil {
					field.Set(reflect.ValueNoEscapeOf(newService))
				}
			} else {
				field.Set(reflect.ValueNoEscapeOf(newService).Elem())
			}
		}
	}

	return service
}

func fillEnvVar(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.Ptr:
		panic("cannot assign to ptr type")
	case reflect.String:
		field.Set(reflect.ValueNoEscapeOf(value))
	case reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(v))
	case reflect.Int:
		v, err := strconv.ParseInt(value, 10, 0)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(int(v)))
	case reflect.Int32:
		v, err := strconv.ParseInt(value, 10, 32)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(int32(v)))
	case reflect.Int16:
		v, err := strconv.ParseInt(value, 10, 16)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(int16(v)))
	case reflect.Int8:
		v, err := strconv.ParseInt(value, 10, 8)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(int8(v)))
	case reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, 64)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(v))
	case reflect.Uint:
		v, err := strconv.ParseUint(value, 10, 0)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(uint(v)))
	case reflect.Uint32:
		v, err := strconv.ParseUint(value, 10, 32)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(uint32(v)))
	case reflect.Uint16:
		v, err := strconv.ParseUint(value, 10, 16)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(uint16(v)))
	case reflect.Uint8:
		v, err := strconv.ParseUint(value, 10, 8)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(uint8(v)))
	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(v))
	case reflect.Float32:
		v, err := strconv.ParseFloat(value, 64)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(float32(v)))
	case reflect.Complex128:
		v, err := strconv.ParseComplex(value, 128)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(v))
	case reflect.Complex64:
		v, err := strconv.ParseComplex(value, 64)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(complex64(v)))
	case reflect.Bool:
		v, err := strconv.ParseBool(value)

		if err != nil {
			panic(err)
		}

		field.Set(reflect.ValueNoEscapeOf(v))
	default:
		panic("invalid type for env variable")
	}
}
