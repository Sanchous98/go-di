package di

import (
	"github.com/joho/godotenv"
	"os"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

type visitedStack []*any

func (v *visitedStack) Pop() *any {
	return v.PopFrom(len(*v) - 1)
}

func (v *visitedStack) PopFrom(i int) *any {
	if len(*v) == 0 {
		return nil
	}

	item := (*v)[i]
	*v = (*v)[:i]
	return item
}

func (v *visitedStack) Push(value *any) {
	*v = append(*v, value)
}

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
	// Use envTag to inject environment variable
	envTag = "env"
)

type serviceContainer struct {
	resolversNum, resolvedNum    int
	resolvers, resolved, tagsMap sync.Map
	mu                           sync.Mutex
	once                         sync.Once
	params                       map[string]string
	currentlyBuilding            visitedStack
}

func NewContainer() PrecompiledGlobalState {
	return &serviceContainer{params: make(map[string]string)}
}

func (c *serviceContainer) Get(_type any) any {
	var serviceType reflect.Type
	switch _type := _type.(type) {
	case reflect.Type:
		serviceType = _type
	default:
		serviceType = reflect.TypeOf(_type)
	}

	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
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
	}

	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	_, ok := c.resolved.Load(serviceType)

	if ok {
		return true
	}

	_, ok = c.resolvers.Load(serviceType)

	return ok
}

func (c *serviceContainer) Set(resolver any, tags ...string) {
	c.mu.Lock()

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
		value := reflect.Indirect(reflect.ValueOf(resolver))

		if value.Kind() != reflect.Struct {
			panic("Container can receive only Resolver or struct or pointer to struct")
		}

		c.resolvers.Store(value.Type(), func(Container) any {
			return c.Build(resolver)
		})

		if len(tags) > 0 {
			c.tagsMap.Store(value.Type(), tags)
		}
	}
	c.resolversNum++
	c.mu.Unlock()
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
	c.once.Do(c.compile)
}

func (c *serviceContainer) compile() {
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
	c.currentlyBuilding = nil
	runtime.GC()
	c.mu.Unlock()
}

func (c *serviceContainer) Destroy() {
	c.mu.Lock()
	c.resolved.Range(func(_, resolved any) bool {
		switch resolved.(type) {
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

func (c *serviceContainer) Build(service any) any {
	return c.fillService(service)
}

// fillService builds a Service using singletons from Container or new instances of another Services
func (c *serviceContainer) fillService(service any) any {
	c.currentlyBuilding.Push(&service)
	stackSize := len(c.currentlyBuilding)
	s := reflect.Indirect(reflect.ValueOf(service))

	for i := 0; i < s.NumField(); i++ {
		tags := s.Type().Field(i).Tag
		envVar, ok := tags.Lookup(envTag)
		field := s.Field(i)
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		if ok {
			field.Set(reflect.ValueOf(c.params[envVar]))
			continue
		}

		tag, ok := tags.Lookup(injectTag)

		if !ok {
			continue
		}

		if len(tag) > 0 {
			if field.Type().Kind() != reflect.Slice {
				panic("tagged field must be slice")
			}

			field.Set(reflect.MakeSlice(field.Type(), 1, 1))
			_t := field.Index(0).Type()
			field.Set(field.Slice(0, 0))

			c.tagsMap.Range(func(_type, tags any) bool {
				if in(tag, tags.([]string)) {
					newService := c.buildService(_type.(reflect.Type))

					if _t.Kind() == reflect.Ptr || _t.Kind() == reflect.Interface {
						field.Set(reflect.Append(field, newService))
					} else {
						field.Set(reflect.Append(field, newService.Elem()))
					}
				}

				return true
			})

			continue
		}

		newService := c.buildService(field.Type())

		if field.Type().Kind() == reflect.Ptr || field.Type().Kind() == reflect.Interface {
			field.Set(newService)
		} else {
			field.Set(newService.Elem())
		}
	}

	switch service.(type) {
	case Constructable:
		service.(Constructable).Constructor()
	}

	c.currentlyBuilding = c.currentlyBuilding[:stackSize]

	return service
}

func (c *serviceContainer) buildService(_type reflect.Type) reflect.Value {
	if _type.Kind() == reflect.Ptr {
		_type = _type.Elem()
	}

	for _, service := range c.currentlyBuilding {
		v := reflect.ValueOf(*service)
		if reflect.Indirect(v).Type() == _type {
			return v
		}
	}

	var newService any

	if c.Has(_type) {
		// If service is bound, take it from the container
		newService = c.Get(_type)
	} else {
		newService = reflect.New(_type).Interface()
		c.fillService(newService)
		c.resolved.Store(_type, newService)
		c.resolvedNum++
	}

	return reflect.ValueOf(newService)
}

func (c *serviceContainer) LoadEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.loadEnv(".env"); err != nil {
		panic(err)
	}
}

func (c *serviceContainer) loadEnv(filename string) error {
	var err error
	c.params, err = godotenv.Read(filename)

	if err != nil {
		return err
	}

	if env, ok := c.params["APP_ENV"]; ok {
		params, err := godotenv.Read(filename + "." + env)

		if err != nil {
			return nil
		}

		for key, value := range params {
			c.params[key] = value
		}
	}

	return nil
}

func (c *serviceContainer) GetParam(param string) string {
	if _, ok := c.params[param]; !ok {
		c.params[param] = os.Getenv(param)
	}

	return c.params[param]
}

func in[T comparable](needle T, haystack []T) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}
