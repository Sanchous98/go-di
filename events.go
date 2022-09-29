package di

import "sync/atomic"

type BaseEvent struct {
	stoppedPropagation uint32
	element            any
}

func (e *BaseEvent) StopPropagation()   { atomic.CompareAndSwapUint32(&e.stoppedPropagation, 0, 1) }
func (e *BaseEvent) CanPropagate() bool { return atomic.LoadUint32(&e.stoppedPropagation) == 0 }
func (e *BaseEvent) GetElement() any    { return e.element }
