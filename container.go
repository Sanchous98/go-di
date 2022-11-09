package di

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
	// Use envTag to inject environment variable
	envTag = "env"
)

type CompileEvent struct {
	BaseEvent
}

type eventHandlers map[int][]func(Event)

func (eh eventHandlers) Len() int {
	return len(eh)
}

func (eh eventHandlers) Swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
}

func (eh eventHandlers) Less(i, j int) bool {
	return i < j
}

type serviceContainer struct {
	resolversNum, resolvedNum    int
	resolvers, resolved, tagsMap sync.Map
	mu                           sync.RWMutex
	once                         sync.Once
	params                       map[string]string
	preCompile                   eventHandlers
	postCompile                  eventHandlers
}

func NewContainer() PrecompiledGlobalState {
	return &serviceContainer{
		params:      make(map[string]string),
		preCompile:  make(map[int][]func(Event)),
		postCompile: make(map[int][]func(Event)),
	}
}

func (c *serviceContainer) Get(_type any) any {
	var serviceType reflect.Type
	switch _type := _type.(type) {
	case reflect.Type:
		serviceType = _type
	default:
		serviceType = reflect.TypeOf(_type)

		if serviceType.Kind() == reflect.Ptr {
			serviceType = serviceType.Elem()
		}
	}

	var resolved, resolver any
	var ok bool

	if resolved, ok = c.resolved.Load(serviceType); !ok {
		if resolver, ok = c.resolvers.Load(serviceType); !ok {
			return nil
		}

		c.resolved.Store(serviceType, reflect.ValueOf(resolver).Call([]reflect.Value{reflect.ValueOf(c)})[0].Interface())
		resolved, _ = c.resolved.Load(serviceType)
	}

	return resolved
}

func (c *serviceContainer) Has(_type any) bool {
	var serviceType reflect.Type
	switch _type := _type.(type) {
	case reflect.Type:
		serviceType = _type
	default:
		serviceType = reflect.TypeOf(_type)

		if serviceType.Kind() == reflect.Ptr {
			serviceType = serviceType.Elem()
		}
	}

	_, ok := c.resolved.Load(serviceType)

	if ok {
		return true
	}

	_, ok = c.resolvers.Load(serviceType)

	return ok
}

func (c *serviceContainer) Set(resolver any, tags ...string) {
	c.mu.RLock()

	typeOf := reflect.TypeOf(resolver)

	if typeOf.Kind() == reflect.Func {
		if typeOf.NumOut() != 1 {
			panic("Resolver is expected to return 1 value")
		}
		if typeOf.NumIn() > 1 {
			panic("Resolver receives only 1 parameter")
		}
		if typeOf.NumIn() == 1 && !typeOf.In(0).Implements(reflect.TypeOf(new(Container)).Elem()) {
			panic("Resolver receives only Container")
		}

		returnType := typeOf.Out(0)

		if returnType.Kind() == reflect.Ptr {
			returnType = returnType.Elem()
		}

		c.resolvers.Store(returnType, resolver)

		if len(tags) > 0 {
			c.tagsMap.Store(returnType, tags)
		}

	} else {
		value := reflect.ValueOf(resolver)

		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		if value.Kind() != reflect.Struct {
			panic("Container can receive only Resolver or struct or pointer to struct")
		}

		c.resolvers.Store(value.Type(), func(Container) any {
			return c.fillService(resolver)
		})

		if len(tags) > 0 {
			c.tagsMap.Store(value.Type(), tags)
		}
	}
	c.resolversNum++
	c.mu.RUnlock()
}

func (c *serviceContainer) All() []any {
	all := make([]any, 0, c.resolvedNum)

	c.resolved.Range(func(_, service any) bool {
		all = append(all, service)

		return true
	})

	return all
}

func (c *serviceContainer) Compile() {
	sort.Sort(c.preCompile)
	sort.Sort(c.postCompile)

	event := &CompileEvent{BaseEvent{element: c}}

	for _, preCompiled := range c.preCompile {
		for _, preCompile := range preCompiled {
			if event.CanPropagate() {
				preCompile(event)
			}
		}
	}

	c.once.Do(c.compile)

	event = &CompileEvent{BaseEvent{element: c}}

	for _, postCompiled := range c.postCompile {
		for _, postCompile := range postCompiled {
			if event.CanPropagate() {
				postCompile(event)
			}
		}
	}
}

func (c *serviceContainer) compile() {
	c.LoadEnv()
	c.mu.Lock()
	// Self references. Is needed to inject Container as a service
	c.resolved.Store(reflect.TypeOf(new(Container)).Elem(), c)
	c.resolved.Store(reflect.TypeOf(new(PrecompiledContainer)).Elem(), c)
	c.resolved.Store(reflect.TypeOf(new(Environment)).Elem(), c)
	c.resolved.Store(reflect.TypeOf(new(GlobalState)).Elem(), c)
	c.resolved.Store(reflect.TypeOf(new(PrecompiledGlobalState)).Elem(), c)
	c.resolvedNum += 5

	c.resolvers.Range(func(_type, resolver any) bool {
		resolverValue := reflect.ValueOf(resolver)
		var args []reflect.Value = nil

		if resolverValue.Type().NumIn() > 0 {
			args = []reflect.Value{reflect.ValueOf(c)}
		}

		c.resolved.Store(_type.(reflect.Type), resolverValue.Call(args)[0].Interface())
		c.resolvedNum++

		return true
	})

	c.mu.Unlock()
}

func (c *serviceContainer) Destroy() {
	c.mu.Lock()
	c.resolved.Range(func(_, resolved any) bool {
		switch resolved.(type) {
		case Stoppable:
			resolved.(Stoppable).Shutdown()
		case Destructible:
			resolved.(Destructible).Destructor()
		}

		return true
	})

	c.resolved = sync.Map{}
	c.once = sync.Once{}
	c.resolvedNum = 0
	c.resolversNum = 0
	c.mu.Unlock()
}

// fillService builds a Service using singletons from Container or new instances of another Services
func (c *serviceContainer) fillService(service any) any {
	s := reflect.ValueOf(service)

	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	for i := 0; i < s.NumField(); i++ {
		tags := s.Type().Field(i).Tag
		envVar, ok := tags.Lookup(envTag)

		if ok {
			s.Field(i).Set(reflect.ValueOf(c.params[envVar]))
			continue
		}

		tag, ok := tags.Lookup(injectTag)

		if !ok {
			continue
		}

		field := s.Field(i)

		if len(tag) > 0 {
			if field.Type().Kind() != reflect.Slice {
				panic("tagged field must be slice")
			}

			field.Set(reflect.MakeSlice(field.Type(), 1, 1))
			_type := field.Index(0).Type()
			field.Set(field.Slice(0, 0))

			c.tagsMap.Range(func(_, tags any) bool {
				if inArray(tag, tags.([]string)) {
					field.Set(reflect.Append(field, c.buildService(_type.(reflect.Type))))
				}

				return true
			})

			continue
		}

		field.Set(c.buildService(field.Type()))
	}

	switch service.(type) {
	case Constructable:
		service.(Constructable).Constructor()
	}

	return service
}

func (c *serviceContainer) buildService(_type reflect.Type) reflect.Value {
	var newService any
	dependencyType := _type

	if dependencyType.Kind() == reflect.Ptr {
		dependencyType = dependencyType.Elem()
	}

	if c.Has(dependencyType) {
		// If service is bound, take it from the container
		newService = c.Get(dependencyType)
	} else {
		newService = reflect.New(dependencyType).Interface()
		c.fillService(newService)
		c.resolved.Store(dependencyType, newService)
		c.resolvedNum++
	}

	if _type.Kind() == reflect.Ptr || _type.Kind() == reflect.Interface {
		return reflect.ValueOf(newService)
	}
	return reflect.ValueOf(newService).Elem()
}

func (c *serviceContainer) LoadEnv() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, envVar := range os.Environ() {
		env := strings.Split(envVar, "=")
		c.params[env[0]] = env[1]
	}

	var err error
	var dir string

	// LoadEnv common env
	dir, err = os.Getwd()

	if err != nil {
		panic(err)
	}

	defaultEnv, err := os.Open(dir + ".env")

	if err != nil {
		return
	}

	defer defaultEnv.Close()
	c.loadEnv(bufio.NewReader(defaultEnv))
	env := c.GetParam("APP_ENV")

	if len(env) == 0 {
		return
	}

	concreteEnv, err := os.Open(dir + ".env." + env)

	if err != nil {
		return
	}

	defer concreteEnv.Close()
	c.loadEnv(bufio.NewReader(concreteEnv))
}

func (c *serviceContainer) loadEnv(reader *bufio.Reader) {
	var envVar []byte
	var err error

	for {
		if errors.Is(err, io.EOF) {
			break
		}

		envVar, err = reader.ReadBytes('\n')

		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}

		env := bytes.Split(bytes.TrimSpace(envVar), []byte{'='})
		c.params[string(env[0])] = string(env[1])
	}
}

func (c *serviceContainer) GetParam(param string) string {
	if _, ok := c.params[param]; !ok {
		c.params[param] = os.Getenv(param)
	}

	return c.params[param]
}

func (c *serviceContainer) PreCompile(handler func(Event), importance int) {
	c.preCompile[importance] = append(c.preCompile[importance], handler)
}

func (c *serviceContainer) PostCompile(handler func(Event), importance int) {
	c.postCompile[importance] = append(c.postCompile[importance], handler)
}

func inArray[T comparable](needle T, haystack []T) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}
