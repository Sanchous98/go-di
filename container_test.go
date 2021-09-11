package di

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type AnotherTestStruct struct{}

type TestStruct struct {
	Dependency  *AnotherTestStruct `inject:""`
	Dependency2 AnotherTestStruct  `inject:""`
	Dependency3 *AnotherTestStruct `inject:""`
	Dependency4 AnotherTestStruct  `inject:""`
}

func TestResolverBinding(t *testing.T) {
	container := NewContainer()
	container.Set(func(cntr Container) *TestStruct {
		testStruct := &TestStruct{}
		testStruct.Dependency = cntr.Get(testStruct.Dependency).(*AnotherTestStruct)
		testStruct.Dependency2 = *cntr.Get(testStruct.Dependency2).(*AnotherTestStruct)
		testStruct.Dependency3 = cntr.Get(testStruct.Dependency3).(*AnotherTestStruct)
		testStruct.Dependency4 = *cntr.Get(testStruct.Dependency4).(*AnotherTestStruct)

		return testStruct
	})
	container.Set(func(cntr Container) *AnotherTestStruct {
		return &AnotherTestStruct{}
	})
	container.Compile()
	var testStruct *TestStruct
	testStruct = container.Get(testStruct).(*TestStruct)
	assert.NotNil(t, testStruct.Dependency)
	assert.NotNil(t, testStruct.Dependency2)
	assert.NotNil(t, testStruct.Dependency3)
	assert.NotNil(t, testStruct.Dependency4)
}

func TestServiceBinding(t *testing.T) {
	container := NewContainer()
	container.Set(&TestStruct{})
	container.Set(&AnotherTestStruct{})
	container.Compile()
	testStruct := container.Get((*TestStruct)(nil)).(*TestStruct)
	assert.NotNil(t, testStruct.Dependency)
	assert.NotNil(t, testStruct.Dependency2)
	assert.NotNil(t, testStruct.Dependency3)
	assert.NotNil(t, testStruct.Dependency4)
}

func TestAutoWiring(t *testing.T) {
	container := NewContainer()
	testStruct := &TestStruct{}
	container.Set(testStruct)
	container.Compile()

	assert.NotNil(t, testStruct.Dependency)
	assert.NotNil(t, testStruct.Dependency2)
	assert.NotNil(t, testStruct.Dependency3)
	assert.NotNil(t, testStruct.Dependency4)
}
