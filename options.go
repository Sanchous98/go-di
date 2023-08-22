package di

import "reflect"

type Option = func(*entry)

func Default[T any](s T) func(*entry) {
	return func(e *entry) {
		e.types = append(e.types, typeId(typeIndirect(typeOf[T]())))
		e.resolver = func(c *serviceContainer) any { return defaultBuilder(e, s, c) }
	}
}

func Service[T any](s T) func(*entry) {
	return func(e *entry) {
		Annotate[T]()(e)
		e.resolved = s
		e.built.Store(true)
	}
}

func DefaultResolver() func(*entry) {
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

func Annotate[T any]() func(*entry) {
	return func(e *entry) {
		e.types = append(e.types, typeId(typeIndirect(typeOf[T]())))
	}
}

func Resolver[T any, F ~func(Container) T](f F) func(*entry) {
	return func(e *entry) {
		e.built.Store(false)
		e.resolved = nil
		e.resolver = func(c *serviceContainer) any {
			return f(c)
		}

		e.types = append(e.types, valueTypeId(reflect.TypeOf(f).Out(0)))
	}
}

func WithTags(tags ...string) func(*entry) {
	return func(e *entry) { e.tags = append(e.tags, tags...) }
}

func typeOf[T any]() reflect.Type { return typeIndirect(reflect.TypeOf(new(T))) }
