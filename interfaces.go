package di

// Container handles services and fills them
type Container interface {
	Set(any, ...string)
	Build(any) any
	Has(any) bool
	Get(any) any
	All() []any
}

// PrecompiledContainer is an extension of Container that can fill services before usage
type PrecompiledContainer interface {
	Compile()
	Destroy()
	Container
}

// Sandbox runs a entryPoints function with a global state in form of Container
type Sandbox interface {
	AddEntryPoint(func(GlobalState))
	Run(bool)
	PrecompiledGlobalState
}

// Environment handles .env vars
type Environment interface {
	LoadEnv()
	GetParam(string) string
}

// GlobalState is represented by Container and Environment
type GlobalState interface {
	Container
	Environment
}

// PrecompiledGlobalState is a GlobalState with PrecompiledContainer
type PrecompiledGlobalState interface {
	PrecompiledContainer
	Environment
}

// Constructable is a service that has special method that initializes it
type Constructable interface {
	Constructor()
}

// Destructible is a service that has special method that destructs it
type Destructible interface {
	Destructor()
}

// Object is a Constructable and Destructible service
type Object interface {
	Constructable
	Destructible
}

type Launchable interface {
	Launch()
}

type Stoppable interface {
	Shutdown()
}

type Service interface {
	Launchable
	Stoppable
}
