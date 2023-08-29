package di

import "reflect"

type Option = func(*entry)

func Default[T any](s T) Option {
	return func(e *entry) {
		e.types = append(e.types, typeId(typeIndirect(typeOf[T]())))
		e.resolver = func(c *serviceContainer) any { return defaultBuilder(e, s, c) }
	}
}

func Service[T any](s T) Option {
	return func(e *entry) {
		Annotate[T]()(e)
		e.resolved = s
		e.built.Store(true)
	}
}

func DefaultResolver() Option {
	return func(e *entry) {
		if e.resolved == nil {
			panic("no service to resolve")
		}

		s := e.resolved
		e.resolved = nil
		e.resolver = func(c *serviceContainer) any { return defaultBuilder(e, s, c) }
		e.built.Store(false)
	}
}

func Annotate[T any]() Option {
	return func(e *entry) {
		e.types = append(e.types, typeId(typeIndirect(typeOf[T]())))
	}
}

func Resolver[T any, F ~func(Container) T](f F) Option {
	return func(e *entry) {
		e.built.Store(false)
		e.resolved = nil
		e.resolver = func(c *serviceContainer) any { return f(c) }
		e.types = append(e.types, valueTypeId(typeOf[T]()))
	}
}

func WithTags(tags ...string) Option {
	return func(e *entry) { e.tags = append(e.tags, tags...) }
}

func Constructor(constructor any) Option {
	return func(e *entry) {
		fn := reflect.ValueOf(constructor)

		if fn.Kind() != reflect.Func {
			panic("constructor must be a function")
		}

		if fn.Type().NumOut() == 0 {
			panic("constructor must return a service")
		}

		e.types = append(e.types, typeId(typeIndirect(fn.Type().Out(0))))
		e.resolver = func(c *serviceContainer) any {
			args := make([]reflect.Value, 0, fn.Type().NumIn())

			for i := 0; i < fn.Type().NumIn(); i++ {
				service := c.Get(fn.Type().In(i))

				if service == nil {
					service = reflect.Zero(fn.Type().In(i))
				}

				args = append(args, reflect.ValueOf(service))
			}

			return fn.Call(args)[0].Interface()
		}
	}
}

func typeOf[T any]() reflect.Type { return typeIndirect(reflect.TypeOf(new(T))) }
