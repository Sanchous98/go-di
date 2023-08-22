package di

import (
	"errors"
	"reflect"
	"sync/atomic"
)

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
)

var EntryNotFound = errors.New("entry not found")

type serviceContainer struct {
	build atomic.Bool

	buildingStack visitedStack
	entries       []*entry
}

func NewContainer() PrecompiledContainer { return new(serviceContainer) }

func (c *serviceContainer) Get(_type any) any {
	serviceType := valueTypeId(_type)

	for _, e := range c.buildingStack {
		if e.TypeOf(serviceType) {
			return e.Build(c)
		}
	}

	for _, e := range c.entries {
		if e.TypeOf(serviceType) {
			return e.Build(c)
		}
	}

	return nil
}

func (c *serviceContainer) GetByTag(tag string) []any {
	byTag := make([]any, 0)

	for _, e := range c.buildingStack {
		if e.HasTag(tag) {
			byTag = append(byTag, e.Build(c))
		}
	}

	for _, e := range c.entries {
		if e.HasTag(tag) {
			byTag = append(byTag, e.Build(c))
		}
	}

	return byTag
}

func (c *serviceContainer) Has(_type any) bool {
	serviceType := valueTypeId(_type)

	for _, e := range c.entries {
		if e.TypeOf(serviceType) {
			return true
		}
	}

	return false
}

func (c *serviceContainer) Set(options ...Option) {
	e := new(entry)

	for _, option := range options {
		option(e)
	}

	c.entries = append(c.entries, e)
}

func (c *serviceContainer) All() []any {
	all := make([]any, 0, len(c.entries))
	for _, e := range c.entries {
		all = append(all, e.Build(c))
	}

	return all
}

func (c *serviceContainer) Compile() {
	if c.build.CompareAndSwap(false, true) {
		c.compile()
	}
}

func (c *serviceContainer) compile() {
	// Self references. Is needed to inject Container as a service
	c.Set(Service[Container](c), Annotate[PrecompiledContainer]())

	for _, e := range c.entries {
		e.Build(c)
	}

	c.buildingStack = nil
}

func (c *serviceContainer) Build(options ...Option) any {
	e := new(entry)

	for _, option := range options {
		option(e)
	}

	if s := e.Build(c); s != nil {
		return s
	}

	panic("something went wrong. Nil result of Container.Build method can be due to self-depending service, which cannot be resolved")
}

func (c *serviceContainer) Destroy() {
	for _, e := range c.entries {
		e.Destroy()
	}

	c.build.Store(false)
}

func validateFunc(typeOf reflect.Type) {
	if typeOf.Kind() != reflect.Func {
		panic("misuse of validateFunc")
	}

	if typeOf.NumIn() > 1 {
		panic("Resolver receives only 1 parameter")
	}
	if typeOf.NumIn() == 1 && !typeOf.In(0).Implements(reflect.TypeOf(new(Container)).Elem()) {
		panic("Resolver receives only Container")
	}

	if typeOf.NumOut() == 0 {
		panic("resolver must return service")
	}
}
