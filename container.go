package di

import (
	"sync/atomic"
)

// Use injectTag to inject dependency into a service
const injectTag = "inject"

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

func (c *serviceContainer) Set(option Option, options ...Option) {
	e := new(entry)
	option(e)

	for _, o := range options {
		o(e)
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

func (c *serviceContainer) Build(option Option, options ...Option) any {
	e := new(entry)
	option(e)

	for _, o := range options {
		o(e)
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
