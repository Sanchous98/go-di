package di

import (
	"github.com/goccy/go-reflect"
	"github.com/joho/godotenv"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

const (
	// Use injectTag to inject dependency into a service
	injectTag = "inject"
	// Use envTag to inject environment variable
	envTag = "env"
)

type serviceContainer struct {
	resolversNum, resolvedNum int
	once                      sync.Once

	mu     sync.Mutex
	params map[string]string

	buildingStack visitedStack[any]

	tagsMap, resolved, resolvers sync.Map
}

func NewContainer() PrecompiledGlobalState {
	return &serviceContainer{
		params: make(map[string]string),
	}
}

func (c *serviceContainer) Get(_type any) any {
	var serviceType uintptr
	switch _type := _type.(type) {
	case uintptr:
		serviceType = _type
	case reflect.Type:
		serviceType = typeId(typeIndirect(_type))
	default:
		serviceType = typeId(typeIndirect(reflect.TypeOf(_type)))
	}

	var resolved, resolver any
	var ok bool

	if resolved, ok = c.resolved.Load(serviceType); !ok {
		if resolver, ok = c.resolvers.Load(serviceType); !ok {
			return nil
		}

		c.executeResolver(serviceType, resolver)
		resolved, _ = c.resolved.Load(serviceType)
	}

	return resolved
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

	_, ok := c.resolved.Load(serviceType)

	if ok {
		return true
	}

	_, ok = c.resolvers.Load(serviceType)

	return ok
}

func (c *serviceContainer) Set(resolver any, tags ...string) {
	typeOf := reflect.TypeOf(resolver)

	if typeOf.Kind() == reflect.Func {
		if typeOf.NumIn() > 1 {
			panic("Resolver receives only 1 parameter")
		}
		if typeOf.NumIn() == 1 && !typeOf.In(0).Implements(reflect.TypeOf(new(Container)).Elem()) {
			panic("Resolver receives only Container")
		}

		if typeOf.NumOut() == 0 {
			// Just run callback if no return values
			c.resolvers.Store(c.resolversNum, resolver)
			c.resolversNum++
			return
		}

		returnType := typeId(typeIndirect(typeOf.Out(0)))

		c.resolvers.Store(returnType, resolver)

		if len(tags) > 0 {
			c.tagsMap.Store(returnType, tags)
		}
	} else {
		value := typeIndirect(reflect.TypeOf(resolver))

		if value.Kind() != reflect.Struct {
			panic("Container can receive only Resolver or struct or pointer to struct")
		}

		c.resolvers.Store(typeId(value), func(Container) any {
			return c.Build(resolver)
		})

		if len(tags) > 0 {
			c.tagsMap.Store(typeId(value), tags)
		}
	}
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
	// Self references. Is needed to inject Container as a service
	c.resolved.Store(typeId(reflect.TypeOf(new(Container)).Elem()), c)
	c.resolved.Store(typeId(reflect.TypeOf(new(PrecompiledContainer)).Elem()), c)
	c.resolved.Store(typeId(reflect.TypeOf(new(Environment)).Elem()), c)
	c.resolved.Store(typeId(reflect.TypeOf(new(GlobalState)).Elem()), c)
	c.resolved.Store(typeId(reflect.TypeOf(new(PrecompiledGlobalState)).Elem()), c)
	c.resolvedNum += 5

	c.resolvers.Range(c.executeResolver)
	c.buildingStack = nil
}

func (c *serviceContainer) executeResolver(_type, resolver any) bool {
	if _, ok := c.resolved.Load(_type); ok {
		return true
	}

	resolverValue := reflect.ValueNoEscapeOf(resolver)
	var args []reflect.Value

	if resolverValue.Type().NumIn() > 0 {
		args = []reflect.Value{reflect.ValueNoEscapeOf(c)}
	}

	if resolverValue.Type().NumOut() == 0 {
		resolverValue.Call(args)
	} else {
		resolved := resolverValue.Call(args)[0].Interface()

		switch resolved.(type) {
		case Constructable:
			resolved.(Constructable).Constructor()
		}

		c.resolved.Store(_type, resolved)
		c.resolvedNum++
	}

	return true
}

func (c *serviceContainer) Destroy() {
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
}

// Build builds a Service using singletons from Container or new instances of another Services
func (c *serviceContainer) Build(service any) any {
	c.buildingStack.Push(&service)
	stackSize := len(c.buildingStack)
	s := reflect.ValueNoEscapeOf(service)

	if s.Type().Kind() == reflect.Interface {
		panic("Trying to fill interface type")
	}

	s = reflect.Indirect(s)

	if s.Kind() == reflect.Interface {
		// Nothing to build in interface
		return service
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

				c.tagsMap.Range(func(_type, tags any) bool {
					if in(tag, tags.([]string)) {
						newService := c.buildService(_type.(uintptr))

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

			newService := c.buildService(typeId(typeIndirect(field.Type())))

			if field.Type().Kind() == reflect.Ptr || field.Type().Kind() == reflect.Interface {
				field.Set(newService)
			} else {
				field.Set(newService.Elem())
			}
		}
	}

	c.buildingStack = c.buildingStack[:stackSize]

	return service
}

func (c *serviceContainer) buildService(_type uintptr) reflect.Value {
	for _, service := range c.buildingStack {
		v := reflect.ValueNoEscapeOf(*service)
		if typeId(reflect.Indirect(v).Type()) == _type {
			return v
		}
	}

	var newService any

	if c.Has(_type) {
		// If service is bound, take it from the container
		newService = c.Get(_type)
	} else {
		newService = reflect.New(idType(_type)).Interface()
		c.Build(newService)
		c.resolved.Store(_type, newService)
		c.resolvedNum++

		switch newService.(type) {
		case Constructable:
			newService.(Constructable).Constructor()
		}
	}

	return reflect.ValueNoEscapeOf(newService)
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

func (c *serviceContainer) loadEnv(file io.Reader) (err error) {
	c.params, err = godotenv.Parse(file)

	if err != nil {
		return err
	}

	if env, ok := c.params["APP_ENV"]; ok {
		params, err := godotenv.Read(".env." + env)

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
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.params[param]; !ok {
		c.params[param] = os.Getenv(param)
	}

	return c.params[param]
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
