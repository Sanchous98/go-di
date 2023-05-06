package di

import (
	"errors"
	"github.com/Sanchous98/go-di/sync"
	"github.com/goccy/go-reflect"
	"github.com/joho/godotenv"
	"io"
	"os"
	"sync/atomic"
)

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
	// Use envTag to inject environment variable
	envTag = "env"
)

var EntryNotFound = errors.New("entry not found")

type serviceContainer struct {
	build atomic.Bool

	mu     sync.Mutex
	params sync.Map[string, string]

	buildingStack visitedStack
	entries       []*entry
}

func NewContainer() PrecompiledGlobalState { return new(serviceContainer) }

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

func (c *serviceContainer) Set(resolver any, tags ...string) {
	typeOf := reflect.TypeOf(resolver)

	if typeOf.Kind() == reflect.Func {
		validateFunc(typeOf)

		c.entries = append(c.entries, &entry{
			types: []uintptr{valueTypeId(typeOf.Out(0))},
			resolver: func(*serviceContainer) any {
				return reflect.ValueNoEscapeOf(resolver).Call([]reflect.Value{reflect.ValueNoEscapeOf(c)})[0].Interface()
			},
			tags: tags,
		})

		return
	}

	value := typeIndirect(reflect.TypeOf(resolver))

	if value.Kind() != reflect.Struct {
		panic("Container can receive only Resolver or struct or pointer to struct, including interfaces")
	}

	e := &entry{
		types: []uintptr{typeId(value)},
		tags:  tags,
	}

	e.resolver = func(c *serviceContainer) any {
		return defaultBuilder(e, resolver, c)
	}

	c.entries = append(c.entries, e)
}

func (c *serviceContainer) AppendTypes(entryType any, appendTypes ...any) error {
	if !c.Has(entryType) {
		return EntryNotFound
	}

	serviceType := valueTypeId(entryType)

	for _, e := range c.buildingStack {
		if e.TypeOf(serviceType) {
			for _, appendType := range appendTypes {
				e.AddType(valueTypeId(appendType))
			}
		}
	}

	for _, e := range c.entries {
		if e.TypeOf(serviceType) {
			for _, appendType := range appendTypes {
				e.AddType(valueTypeId(appendType))
			}
		}
	}

	return nil
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
	c.entries = append(c.entries, &entry{
		types: []uintptr{
			valueTypeId(new(Container)),
			valueTypeId(new(PrecompiledContainer)),
			valueTypeId(new(Environment)),
			valueTypeId(new(GlobalState)),
			valueTypeId(new(PrecompiledGlobalState)),
		},
		resolver: func(*serviceContainer) any { return c },
	})

	for _, e := range c.entries {
		e.Build(c)
	}

	c.buildingStack = nil
}

func (c *serviceContainer) Build(service any) any {
	if s := defaultEntry(service).Build(c); s != nil {
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

func (c *serviceContainer) LoadEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(".env")

	if err != nil {
		panic(err)
	}

	defer file.Close()

	if err = c.loadEnv(file); err != nil {
		panic(err)
	}
}

func (c *serviceContainer) loadEnv(file io.Reader) error {
	params, err := godotenv.Parse(file)

	if err != nil {
		return err
	}

	for param, value := range params {
		c.params.Store(param, value)
	}

	if env, ok := c.params.Load("APP_ENV"); ok {
		params, err = godotenv.Read(".env." + env)

		if err != nil {
			return nil
		}

		for key, value := range params {
			c.params.Store(key, value)
		}
	}

	return nil
}

func (c *serviceContainer) GetParam(param string) string {
	if p, loaded := c.params.LoadOrStore(param, os.Getenv(param)); loaded {
		return p
	}

	p, _ := c.params.Load(param)
	return p
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
