package di

import "sync/atomic"

type Event interface {
	StopPropagation()
	CanPropagate() bool
	GetElement() interface{}
}

type BaseEvent struct {
	stoppedPropagation uint32
	element            interface{}
}

func (e *BaseEvent) StopPropagation() {
	if atomic.LoadUint32(&e.stoppedPropagation) == 0 {
		atomic.StoreUint32(&e.stoppedPropagation, 1)
	}
}

func (e *BaseEvent) CanPropagate() bool {
	return atomic.LoadUint32(&e.stoppedPropagation) == 0
}

func (e *BaseEvent) GetElement() interface{} {
	return e.element
}
