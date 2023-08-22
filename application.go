package di

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func NewApplication(name string) Runner {
	return &application{name: name, PrecompiledContainer: new(serviceContainer)}
}

// application is a global state for program
type application struct {
	PrecompiledContainer

	name string
}

func (a *application) Name() string { return a.name }

func (a *application) Run(ctx context.Context) {
	a.Compile()

	var stop context.CancelFunc
	ctx, stop = signal.NotifyContext(context.WithValue(ctx, "container", a.PrecompiledContainer), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stop()

	all := a.PrecompiledContainer.All()

	for _, service := range all {
		switch service.(type) {
		case Launchable:
			go service.(Launchable).Launch(ctx)
		}
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
