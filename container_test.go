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

type EnvTestStruct struct {
	i64        int64      `env:"INT64"`
	i          int        `env:"INT"`
	i32        int32      `env:"INT32"`
	i16        int16      `env:"INT16"`
	i8         int8       `env:"INT8"`
	ui64       uint64     `env:"UINT64"`
	ui         uint       `env:"UINT"`
	ui32       uint32     `env:"UINT32"`
	ui16       uint16     `env:"UINT16:-80"`
	ui8        uint8      `env:"UINT8"`
	b          byte       `env:"UINT8"`
	bl         bool       `env:"BOOL"`
	float64    float64    `env:"FLOAT64"`
	float32    float32    `env:"FLOAT32"`
	complex128 complex128 `env:"COMPLEX128"`
	complex64  complex64  `env:"COMPLEX64"`
	str        string     `env:"STRING"`
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
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestServiceBinding() {
	s.container.Set(new(TestStruct))
	s.container.Set(new(AnotherTestStruct))
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
	s.container.Set(testStruct)
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
	s.NotPanics(func() {
		testStruct.TaggedDependency3[0].I()
	})
	s.Len(s.container.(*serviceContainer).entries, 4)
	s.Len(s.container.All(), len(s.container.(*serviceContainer).entries))
}

func (s *ContainerTestSuite) TestGetByTag() {
	for i := 0; i < 10; i++ {
		anotherTestStruct := new(AnotherTestStruct)
		s.container.Set(anotherTestStruct, "test_tag")
	}

	s.Require().NotPanics(s.container.Compile)

	s.Len(s.container.GetByTag("test_tag"), 10)
}

func (s *ContainerTestSuite) TestAppendingTypes() {
	s.container.Set(new(AnotherTestStruct))
	s.Nil(s.container.AppendTypes((*AnotherTestStruct)(nil), new(TestInterface)))
	s.NotPanics(s.container.Compile)

	s.True(s.container.Has((*AnotherTestStruct)(nil)))
	s.True(s.container.Has(new(TestInterface)))
	s.Len(s.container.All(), 4)
	s.Same(s.container.Get((*AnotherTestStruct)(nil)).(*AnotherTestStruct), s.container.Get(new(TestInterface)).(*AnotherTestStruct))
}

func (s *ContainerTestSuite) TestBuildRunsConstructor() {
	s.container.Set(func(c Container) Container {
		return s.container
	})
	s.container.Build(new(AnotherTestStruct))
	s.True(s.container.Get((*AnotherTestStruct)(nil)).(*AnotherTestStruct).called)
}

func (s *ContainerTestSuite) TestNotBuildingInterfaceFields() {
	s.container.Set(new(TestInterfaceValue))
	s.PanicsWithValue(
		`interface type without bound value. Remove "inject" tag or set a value, bound by this interface type`,
		s.container.Compile,
	)
}

func (s *ContainerTestSuite) TestCallbackServiceNotNil() {
	s.T().Skip()

	s.container.Set(func(c Container) TestInterface {
		return c.Build(new(AnotherTestStruct)).(TestInterface)
	})
	s.container.Set(new(TestInterfaceValue))
	s.container.Compile()

	t := s.container.Get((*TestInterfaceValue)(nil)).(*TestInterfaceValue)
	s.NotNil(t.Test)
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
		container.Set(testStruct)
		container.Set(anotherTestStruct, "test_tag")
	}

	for i := 0; i < b.N; i++ {
		container.Compile()
		container.Destroy()
	}
}
