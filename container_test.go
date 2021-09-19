package di

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

type AnotherTestStruct struct {
	Container Container `inject:""`
}

type TestStruct struct {
	Dependency  *AnotherTestStruct `inject:""`
	Dependency2 AnotherTestStruct  `inject:""`
	Dependency3 *AnotherTestStruct `inject:""`
	Dependency4 AnotherTestStruct  `inject:""`
}

func TestResolverBinding(t *testing.T) {
	container := &serviceContainer{params: make(map[string]string)}
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
	assert.Len(t, container.All(), container.resolvedNum)
}

func TestServiceBinding(t *testing.T) {
	container := &serviceContainer{params: make(map[string]string)}
	container.Set(&TestStruct{})
	container.Set(&AnotherTestStruct{})
	container.Compile()
	testStruct := container.Get((*TestStruct)(nil)).(*TestStruct)
	assert.NotNil(t, testStruct.Dependency)
	assert.NotNil(t, testStruct.Dependency2)
	assert.NotNil(t, testStruct.Dependency3)
	assert.NotNil(t, testStruct.Dependency4)
	assert.Len(t, container.All(), container.resolvedNum)
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

func TestSelfReferences(t *testing.T) {
	container := NewContainer()
	testStruct := &AnotherTestStruct{}
	container.Set(testStruct)
	container.Compile()
	assert.NotNil(t, testStruct.Container)
	assert.IsType(t, &serviceContainer{}, testStruct.Container)
}

func TestServiceContainer_loadEnv(t *testing.T) {
	container := &serviceContainer{params: make(map[string]string)}
	container.loadEnv(bufio.NewReader(bytes.NewReader([]byte("APP_ENV=dev\nDB_NAME=test"))))
	assert.EqualValues(t, "dev", container.GetParam("APP_ENV"))
	assert.EqualValues(t, "test", container.GetParam("DB_NAME"))
}

func TestServiceContainer_CompileEvents(t *testing.T) {
	container := NewContainer()
	container.PreCompile(func(event Event) {
		assert.NotNil(t, event.GetElement())
	}, 0)

	container.PostCompile(func(event Event) {
		assert.NotNil(t, event.GetElement())
	}, 0)

	container.Compile()
}
