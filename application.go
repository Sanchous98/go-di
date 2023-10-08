package di

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func NewApplication(name string, logger Logger) Runner {
	return &application{name: name, PrecompiledContainer: new(serviceContainer), logger: logger}
}

// application is a global state for program
type application struct {
	PrecompiledContainer

	name   string
	logger Logger
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
			go a.launch(ctx, service.(Launchable))
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

func (a *application) launch(ctx context.Context, service Launchable) {
	defer func() {
		if err := recover(); err != nil {
			a.logger.Errorln(err)
			go a.launch(ctx, service)
		}
	}()

	service.Launch(ctx)
}
