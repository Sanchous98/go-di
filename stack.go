package di

type visitedStack []*entry

func (v *visitedStack) Pop() *entry {
	return v.PopFrom(len(*v) - 1)
}

func (v *visitedStack) PopFrom(i int) *entry {
	if len(*v) == 0 {
		return nil
	}

	item := (*v)[i]
	*v = (*v)[:i]
	return item
}

func (v *visitedStack) Push(value *entry) {
	*v = append(*v, value)
}
