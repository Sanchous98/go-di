package di

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func Application(ctx context.Context) Sandbox {
	if ctx == nil {
		ctx = context.Background()
	}

	return &application{ctx: ctx, PrecompiledGlobalState: NewContainer(), entryPoints: make([]func(GlobalState), 0)}
}

// application is a global state for program
type application struct {
	PrecompiledGlobalState
	ctx         context.Context
	entryPoints []func(GlobalState)
}

func (a *application) AddEntryPoint(entryPoint func(GlobalState)) {
	a.entryPoints = append(a.entryPoints, entryPoint)
}

func (a *application) Run(envLoader func()) {
	if envLoader != nil {
		envLoader()
	}
	a.Compile()

	var stop context.CancelFunc
	a.ctx, stop = signal.NotifyContext(context.WithValue(a.ctx, "container", a.PrecompiledGlobalState), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stop()

	all := a.PrecompiledGlobalState.All()

	for _, service := range all {
		switch service.(type) {
		case Launchable:
			go service.(Launchable).Launch(a.ctx)
		}
	}

	for _, entryPoint := range a.entryPoints {
		go entryPoint(a.PrecompiledGlobalState)
	}

	select {
	case <-a.ctx.Done():
		var wg sync.WaitGroup

		for _, service := range all {
			switch service.(type) {
			case Stoppable:
				wg.Add(1)
				go service.(Stoppable).Shutdown(a.ctx)
			}
		}

		wg.Wait()
	}
}
