package di

import (
	"os"
	"os/signal"
	"syscall"
)

var app Sandbox

func init() {
	app = &application{PrecompiledGlobalState: NewContainer(), entryPoints: make([]func(GlobalState), 0)}
	app.Set(Application)
}

func Application() Sandbox {
	return app
}

// application is a global state for program
type application struct {
	PrecompiledGlobalState
	entryPoints []func(GlobalState)
}

func (a *application) AddEntryPoint(entryPoint func(GlobalState)) {
	a.entryPoints = append(a.entryPoints, entryPoint)
}

func (a *application) Run() {
	a.Compile()

	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for _, service := range a.PrecompiledGlobalState.All() {
		switch service := service.(type) {
		case Launchable:
			go service.Launch()
		}
	}

	for _, entryPoint := range a.entryPoints {
		go entryPoint(a.PrecompiledGlobalState)
	}

	select {
	case <-osSignals:
		a.Destroy()
		os.Exit(0)
	}
}
