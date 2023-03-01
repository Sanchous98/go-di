package di

import (
	"errors"
	"github.com/Sanchous98/go-di/sync"
	"github.com/goccy/go-reflect"
	"github.com/joho/godotenv"
	"io"
	"os"
)

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
	// Use envTag to inject environment variable
	envTag = "env"
)

var EntryNotFound = errors.New("entry not found")

type serviceContainer struct {
	resolversNum int
	once         sync.Once

	mu     sync.Mutex
	params sync.Map[string, string]

	buildingStack visitedStack
	entries       []*entry
}

func NewContainer() PrecompiledGlobalState { return new(serviceContainer) }

func (c *serviceContainer) Get(_type any) any {
	var serviceType uintptr
	switch _type.(type) {
	case uintptr:
		serviceType = _type.(uintptr)
	case reflect.Type:
		serviceType = typeId(typeIndirect(_type.(reflect.Type)))
	default:
		serviceType = typeId(typeIndirect(reflect.TypeOf(_type)))
	}

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

func (c *serviceContainer) Has(_type any) bool {
	var serviceType uintptr
	switch _type := _type.(type) {
	case uintptr:
		serviceType = _type
	case reflect.Type:
		serviceType = typeId(typeIndirect(_type))
	default:
		serviceType = typeId(typeIndirect(reflect.TypeOf(_type)))
	}

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
			types: []uintptr{typeId(typeIndirect(typeOf.Out(0)))},
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

	var serviceType uintptr
	switch entryType.(type) {
	case uintptr:
		serviceType = entryType.(uintptr)
	case reflect.Type:
		serviceType = typeId(typeIndirect(entryType.(reflect.Type)))
	default:
		serviceType = typeId(typeIndirect(reflect.TypeOf(entryType)))
	}

	for _, e := range c.buildingStack {
		if e.TypeOf(serviceType) {
			for _, appendType := range appendTypes {
				switch entryType.(type) {
				case uintptr:
					e.AddType(appendType.(uintptr))
				case reflect.Type:
					e.AddType(typeId(typeIndirect(appendType.(reflect.Type))))
				default:
					e.AddType(typeId(typeIndirect(reflect.TypeOf(appendType))))
				}
			}
		}
	}

	for _, e := range c.entries {
		if e.TypeOf(serviceType) {
			for _, appendType := range appendTypes {
				switch entryType.(type) {
				case uintptr:
					e.AddType(appendType.(uintptr))
				case reflect.Type:
					e.AddType(typeId(typeIndirect(appendType.(reflect.Type))))
				default:
					e.AddType(typeId(typeIndirect(reflect.TypeOf(appendType))))
				}
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
	c.once.Do(c.compile)
}

func (c *serviceContainer) compile() {
	// Self references. Is needed to inject Container as a service
	c.entries = append(c.entries, &entry{
		types: []uintptr{
			typeId(reflect.TypeOf(new(Container)).Elem()),
			typeId(reflect.TypeOf(new(PrecompiledContainer)).Elem()),
			typeId(reflect.TypeOf(new(Environment)).Elem()),
			typeId(reflect.TypeOf(new(GlobalState)).Elem()),
			typeId(reflect.TypeOf(new(PrecompiledGlobalState)).Elem()),
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

	c.once = sync.Once{}
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
	p, _ := c.params.LoadOrStore(param, os.Getenv(param))
	return p
}

func validateFunc(typeOf reflect.Type) {
	if typeOf.Kind() != reflect.Func {
		panic("misuse ov validateFunc")
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
