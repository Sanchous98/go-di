package di

import (
	"log"
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

func (a *application) Run(envLoader func(), exitPoint func(os.Signal)) {
	if envLoader != nil {
		envLoader()
	}
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
	case s := <-osSignals:
		for _, service := range a.All() {
			switch service.(type) {
			case Stoppable:
				service.(Stoppable).Shutdown()
			}
		}

		a.Destroy()
		log.Printf(`Stopping application because of signal "%s"`, s.String())

		if exitPoint != nil {
			exitPoint(s)
		}
	}
}
