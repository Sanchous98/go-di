package di

type visitedStack[T any] []*T

func (v *visitedStack[T]) Pop() *T {
	return v.PopFrom(len(*v) - 1)
}

func (v *visitedStack[T]) PopFrom(i int) *T {
	if len(*v) == 0 {
		return nil
	}

	item := (*v)[i]
	*v = (*v)[:i]
	return item
}

func (v *visitedStack[T]) Push(value *T) {
	*v = append(*v, value)
}
