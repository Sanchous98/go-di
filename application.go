package di

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
)

func NewApplication() Sandbox {
	return &application{PrecompiledGlobalState: NewContainer(), entryPoints: make([]func(GlobalState), 0)}
}

// application is a global state for program
type application struct {
	PrecompiledGlobalState
	beforeCompile []func()
	entryPoints   []func(GlobalState)
}

func (a *application) AddEntryPoint(entryPoint func(GlobalState)) {
	a.entryPoints = append(a.entryPoints, entryPoint)
}

func (a *application) Set(service any, tags ...string) {
	_t := reflect.TypeOf(service)

	if _t.Kind() == reflect.Func && _t.NumOut() == 0 {
		a.beforeCompile = append(a.beforeCompile, func() {
			reflect.ValueOf(service).Call([]reflect.Value{reflect.ValueOf(a)})
		})
		return
	}

	a.PrecompiledGlobalState.Set(service, tags...)
}

func (a *application) Run(ctx context.Context, envLoader func()) {
	if envLoader != nil {
		envLoader()
	}

	for _, f := range a.beforeCompile {
		f()
	}

	a.Compile()

	var stop context.CancelFunc
	ctx, stop = signal.NotifyContext(context.WithValue(ctx, "container", a.PrecompiledGlobalState), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stop()

	all := a.PrecompiledGlobalState.All()

	for _, service := range all {
		switch service.(type) {
		case Launchable:
			go service.(Launchable).Launch(ctx)
		}
	}

	for _, entryPoint := range a.entryPoints {
		go entryPoint(a.PrecompiledGlobalState)
	}

	select {
	case <-ctx.Done():
		var wg sync.WaitGroup

		for _, service := range all {
			switch service.(type) {
			case Stoppable:
				wg.Add(1)
				go func(service Stoppable) {
					service.Shutdown(ctx)
					wg.Done()
				}(service.(Stoppable))
			}
		}

		wg.Wait()
	}
}
