package di

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type TestInterface interface {
	I()
}

type AnotherTestStruct struct {
	Container       Container   `inject:""`
	CycleDependency *TestStruct `inject:""`

	t      *testing.T `inject:""`
	called bool
}

func (a *AnotherTestStruct) Constructor() {
	assert.False(a.t, a.called)
	a.called = true
}

func (a *AnotherTestStruct) I() {}

type TestStruct struct {
	Dependency        *AnotherTestStruct   `inject:""`
	Dependency2       AnotherTestStruct    `inject:""`
	dependency3       *AnotherTestStruct   `inject:""`
	dependency4       AnotherTestStruct    `inject:""`
	TaggedDependency  []AnotherTestStruct  `inject:"test_tag"`
	TaggedDependency2 []*AnotherTestStruct `inject:"test_tag"`
	TaggedDependency3 []TestInterface      `inject:"test_tag"`

	t      *testing.T
	called bool
}

func (t *TestStruct) Constructor() {
	assert.False(t.t, t.called)
	t.called = true
}

type ContainerTestSuite struct {
	suite.Suite
	container PrecompiledGlobalState
}

func (s *ContainerTestSuite) SetupTest() {
	s.container = NewContainer()
	s.container.Set(s.T())
}

func (s *ContainerTestSuite) TestResolverBinding() {
	s.container.Set(func(cntr Container) *TestStruct {
		testStruct := new(TestStruct)
		testStruct.Dependency = cntr.Get(testStruct.Dependency).(*AnotherTestStruct)
		testStruct.Dependency2 = *cntr.Get(testStruct.Dependency2).(*AnotherTestStruct)
		testStruct.dependency3 = cntr.Get(testStruct.dependency3).(*AnotherTestStruct)
		testStruct.dependency4 = *cntr.Get(testStruct.dependency4).(*AnotherTestStruct)

		return testStruct
	})
	s.container.Set(func(cntr Container) *AnotherTestStruct { return new(AnotherTestStruct) })
	s.Require().NotPanics(s.container.Compile)
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
	s.Equal(8, s.container.(*serviceContainer).resolvedNum)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func (s *ContainerTestSuite) TestServiceBinding() {
	s.container.Set(new(TestStruct))
	s.container.Set(new(AnotherTestStruct))
	s.Require().NotPanics(s.container.Compile)
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
	s.Equal(8, s.container.(*serviceContainer).resolvedNum)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func (s *ContainerTestSuite) TestAutoWiring() {
	testStruct := new(TestStruct)
	s.container.Set(testStruct)
	s.Require().NotPanics(s.container.Compile)

	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
	s.Equal(8, s.container.(*serviceContainer).resolvedNum)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func (s *ContainerTestSuite) TestCallbacks() {
	testStruct := new(TestStruct)
	s.container.Set(testStruct)
	s.container.Set(func(container Container) {})
	s.Require().NotPanics(s.container.Compile)

	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
}

func (s *ContainerTestSuite) TestSelfReferences() {
	testStruct := new(AnotherTestStruct)
	s.container.Set(testStruct)
	s.Require().NotPanics(s.container.Compile)
	s.NotNil(testStruct.Container)
	s.IsType(new(serviceContainer), testStruct.Container)
}

func (s *ContainerTestSuite) TestTagged() {
	testStruct := new(TestStruct)
	anotherTestStruct := new(AnotherTestStruct)
	s.container.Set(testStruct)
	s.container.Set(anotherTestStruct, "test_tag")

	s.Require().NotPanics(s.container.Compile)

	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
	s.NotNil(testStruct.TaggedDependency)
	s.True(len(testStruct.TaggedDependency) > 0)
	s.NotNil(testStruct.TaggedDependency2)
	s.True(len(testStruct.TaggedDependency2) > 0)
	s.NotNil(testStruct.TaggedDependency3)
	s.True(len(testStruct.TaggedDependency3) > 0)
	s.Equal(8, s.container.(*serviceContainer).resolvedNum)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func TestContainer(t *testing.T) { suite.Run(t, new(ContainerTestSuite)) }

func BenchmarkServiceContainer_Compile(b *testing.B) {
	b.ReportAllocs()
	container := NewContainer()
	//testStruct := new(TestStruct)
	//anotherTestStruct := new(AnotherTestStruct)
	//container.Set(testStruct)
	//container.Set(anotherTestStruct, "test_tag")

	for i := 0; i < b.N; i++ {
		container.Compile()
		container.Destroy()
	}
}
