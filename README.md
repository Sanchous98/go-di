# Go Dependency Injection Container

### Build easily microservice applications

To start you have to create container instance, fill it with services and compile

```go
package main

import "github.com/Sanchous98/go-di"

func main() {
    container := di.NewContainer()
    container.Set(&ExampleService{})
    container.Compile()
    container.Get((*ExampleService)(nil)).(*ExampleService) // Returns filled ExampleService
}
```

There are some ways container can compile a service:

1. Use resolver function. It's the most preferable method to integrate your service in container, because it has less
   performance overhead and permits to bind a service to an interface type. Resolver can have no parameters or receive
   the container. It's better to use interface, you need, because of LSP

```go
container.Set(func (di.Container) *ExampleInterfaceType {
    return &ExampleService{} 
})
container.Get((*ExampleInterfaceType)(nil)).(*ExampleInterfaceType) // Returns *ExampleService
```

2. Pass service to container. In this case fields tagged by "inject" tag will be filled from container. Important:
   precompile container to use it without any performance impact.

```go
container.Set(&ExampleService{})
container.Compile()
container.Get((*ExampleService)(nil)).(*ExampleService) // Returns *ExampleService
```

3. Leave as it is. In this case container will also resolve your service, but only as a dependency for other services.
   Useful for libraries.



Services can have default steps to self initialize and self destroy. To use this feature, implement Constructable and
Destructible interfaces. It's useful for graceful shutdown feature.

```go
type Constructable interface {
    Constructor()
}

type Destructible interface {
    Destructor()
}
```

Also services can be long-living. Implement Launchable and Stoppable interfaces to launch and stop your services
gracefully.

```go
type Launchable interface {
    Launch(context.Context)
}

type Stoppable interface {
    Shutdown(context.Context)
}
```

```Constructor()``` method is called on service compiling. ```Destructor()``` method is called on application exiting
when the container destroys. ```Launch()``` is called on ```Run()``` method calling. ```Shutdown()``` method is called
on application exiting before the container destroys.