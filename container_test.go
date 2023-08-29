package di

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type TestInterfaceValue struct {
	Test TestInterface `inject:""`
}

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
	container PrecompiledContainer
}

func (s *ContainerTestSuite) SetupTest() {
	s.container = NewContainer()
	s.container.Set(Service(s.T()))
}

func (s *ContainerTestSuite) TestResolverBinding() {
	s.container.Set(Resolver(func(cntr Container) *TestStruct {
		testStruct := new(TestStruct)
		testStruct.Dependency = cntr.Get(testStruct.Dependency).(*AnotherTestStruct)
		testStruct.Dependency2 = *cntr.Get(testStruct.Dependency2).(*AnotherTestStruct)
		testStruct.dependency3 = cntr.Get(testStruct.dependency3).(*AnotherTestStruct)
		testStruct.dependency4 = *cntr.Get(testStruct.dependency4).(*AnotherTestStruct)

		return testStruct
	}))
	s.container.Set(Resolver(func(cntr Container) *AnotherTestStruct { return new(AnotherTestStruct) }))
	s.Require().NotPanics(s.container.Compile)
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotNil(testStruct.dependency4)
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestServiceBinding() {
	s.container.Set(Default(new(AnotherTestStruct)))
	s.container.Set(Constructor[TestStruct](func(testStruct *AnotherTestStruct) *TestStruct {
		return &TestStruct{
			Dependency:  testStruct,
			Dependency2: *testStruct,
			dependency3: testStruct,
			dependency4: *testStruct,
		}
	}))
	s.Require().NotPanics(s.container.Compile)
	s.Require().True(s.container.Has((*TestStruct)(nil)))
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotEmpty(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotEmpty(testStruct.dependency4)
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestAutoWiring() {
	testStruct := new(TestStruct)
	s.container.Set(Default(testStruct))
	s.Require().NotPanics(s.container.Compile)

	s.NotNil(testStruct.Dependency)
	s.NotEmpty(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotEmpty(testStruct.dependency4)
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestSelfReferences() {
	testStruct := new(AnotherTestStruct)
	s.container.Set(Default(testStruct))
	s.Require().NotPanics(s.container.Compile)
	s.NotNil(testStruct.Container)
	s.IsType(new(serviceContainer), testStruct.Container)
}

func (s *ContainerTestSuite) TestTagged() {
	testStruct := new(TestStruct)
	anotherTestStruct := new(AnotherTestStruct)
	s.container.Set(Default(testStruct))
	s.container.Set(Default(anotherTestStruct), WithTags("test_tag"))

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
	s.NotPanics(func() {
		testStruct.TaggedDependency3[0].I()
	})
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestGetByTag() {
	for i := 0; i < 10; i++ {
		s.container.Set(Default(new(AnotherTestStruct)), WithTags("test_tag"))
	}

	s.Require().NotPanics(s.container.Compile)

	s.Len(s.container.GetByTag("test_tag"), 10)
}

func (s *ContainerTestSuite) TestAnnotating() {
	s.container.Set(Default(new(AnotherTestStruct)), Annotate[TestInterface]())
	s.NotPanics(s.container.Compile)

	s.True(s.container.Has((*AnotherTestStruct)(nil)))
	s.True(s.container.Has(new(TestInterface)))
	s.Len(s.container.All(), 4)
	s.Same(s.container.Get((*AnotherTestStruct)(nil)).(*AnotherTestStruct), s.container.Get(new(TestInterface)).(*AnotherTestStruct))
}

func (s *ContainerTestSuite) TestBuildRunsConstructor() {
	s.container.Set(Resolver(func(c Container) Container { return s.container }))
	s.container.Build(Default(new(AnotherTestStruct)))
	s.True(s.container.Get((*AnotherTestStruct)(nil)).(*AnotherTestStruct).called)
}

func (s *ContainerTestSuite) TestNotBuildingInterfaceFields() {
	s.container.Set(Default(new(TestInterfaceValue)))
	s.PanicsWithValue(
		`interface type without bound value. Remove "inject" tag or set a value, bound by this interface type`,
		s.container.Compile,
	)
}

func (s *ContainerTestSuite) TestCallbackServiceNotNil() {
	s.container.Set(Resolver(func(c Container) TestInterface {
		return c.Build(Default(new(AnotherTestStruct))).(TestInterface)
	}))
	s.container.Set(Default(new(TestInterfaceValue)))
	s.NotPanics(s.container.Compile)

	t := s.container.Get((*TestInterfaceValue)(nil)).(*TestInterfaceValue)
	s.NotNil(t.Test)
}

func (s *ContainerTestSuite) TestConstructorFunction() {
	s.container.Set(Default(new(TestStruct)))
	s.container.Set(Default(new(AnotherTestStruct)))
	s.Require().NotPanics(s.container.Compile)
	s.Require().True(s.container.Has((*TestStruct)(nil)))
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotEmpty(testStruct.Dependency2)
	s.NotNil(testStruct.dependency3)
	s.NotEmpty(testStruct.dependency4)
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func TestContainer(t *testing.T) { suite.Run(t, new(ContainerTestSuite)) }

type benchTestStruct struct {
	Dependency        *benchAnotherTestStruct   `inject:""`
	Dependency2       benchAnotherTestStruct    `inject:""`
	dependency3       *benchAnotherTestStruct   `inject:""`
	dependency4       benchAnotherTestStruct    `inject:""`
	TaggedDependency  []benchAnotherTestStruct  `inject:"test_tag"`
	TaggedDependency2 []*benchAnotherTestStruct `inject:"test_tag"`
	TaggedDependency3 []TestInterface           `inject:"test_tag"`
}

type benchAnotherTestStruct struct {
	Container       Container        `inject:""`
	CycleDependency *benchTestStruct `inject:""`
}

func (b *benchAnotherTestStruct) I() {}

func BenchmarkServiceContainer_Compile(b *testing.B) {
	b.ReportAllocs()
	container := NewContainer()

	for i := 0; i < 1000; i++ {
		testStruct := new(benchTestStruct)
		anotherTestStruct := new(benchAnotherTestStruct)
		container.Set(Service(testStruct), DefaultResolver())
		container.Set(Service(anotherTestStruct), DefaultResolver(), WithTags("test_tag"))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		container.Compile()
		container.Destroy()
	}
}
