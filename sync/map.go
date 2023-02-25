package sync

import "sync"

type Map[K, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	item, ok := m.Map.Load(key)

	if !ok {
		return *new(V), ok
	}

	return item.(V), ok
}
func (m *Map[K, V]) Store(key K, value V) {
	m.Map.Store(key, value)
}
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	item, ok := m.Map.LoadOrStore(key, value)

	if !ok {
		return *new(V), ok
	}

	return item.(V), ok
}
func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	item, ok := m.Map.LoadAndDelete(key)

	if !ok {
		return *new(V), ok
	}

	return item.(V), ok
}
func (m *Map[K, _]) Delete(key K) {
	m.Map.Delete(key)
}
func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	item, ok := m.Map.Swap(key, value)

	if !ok {
		return *new(V), ok
	}

	return item.(V), ok
}
func (m *Map[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {
	return m.Map.CompareAndSwap(key, old, new)
}
func (m *Map[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return m.Map.CompareAndDelete(key, old)
}
func (m *Map[K, V]) Range(f func(key K, value V) (shouldContinue bool)) {
	m.Map.Range(func(key, value any) bool { return f(key.(K), value.(V)) })
}
