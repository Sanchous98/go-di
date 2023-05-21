package di

import (
	"context"
)

// Container handles services and fills them
type Container interface {
	// Set defines new entry in container
	Set(any, ...string)
	// Build sets new entry in container without adding it to container
	Build(any) any
	// AppendTypes adds referenced types to an existing entry. Returns error if
	AppendTypes(any, ...any) error
	// Has checks whether the service of passed type exists
	Has(any) bool
	// Get returns service from container
	Get(any) any
	// GetByTag returns tagged services from container
	GetByTag(string) []any
	// All return all services
	All() []any
}

// PrecompiledContainer is an extension of Container that can fill services before usage
type PrecompiledContainer interface {
	Compile()
	Destroy()
	Container
}

type Runner interface {
	PrecompiledContainer
	Run(context.Context)
}

// Constructable is a service that has special method that initializes it
type Constructable interface {
	Constructor()
}

// Destructible is a service that has special method that destructs it
type Destructible interface {
	Destructor()
}

type Launchable interface {
	Launch(context.Context)
}

type Stoppable interface {
	Shutdown(context.Context)
}
