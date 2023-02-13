package di

import (
	"bytes"
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

func (s *ContainerTestSuite) TestEnvVars() {
	var testEnv EnvTestStruct
	s.container.Set(&testEnv)

	var builder bytes.Buffer
	builder.WriteString("STRING=test\n")
	builder.WriteString("COMPLEX64=2+3i\n")
	builder.WriteString("COMPLEX128=3+4i\n")
	builder.WriteString("FLOAT32=1.25\n")
	builder.WriteString("FLOAT64=2.5\n")
	builder.WriteString("BOOL=true\n")
	builder.WriteString("UINT8=1\n")
	builder.WriteString("UINT32=1\n")
	builder.WriteString("UINT=1\n")
	builder.WriteString("UINT64=1\n")
	builder.WriteString("INT8=-1\n")
	builder.WriteString("INT16=-1\n")
	builder.WriteString("INT32=-1\n")
	builder.WriteString("INT=-1\n")
	builder.WriteString("INT64=-1\n")

	s.container.(*serviceContainer).loadEnv(&builder)
	s.container.Compile()

	s.Equal("test", testEnv.str)
	s.Equal(complex64(2+3i), testEnv.complex64)
	s.Equal(complex128(3+4i), testEnv.complex128)
	s.Equal(float32(1.25), testEnv.float32)
	s.Equal(float64(2.5), testEnv.float64)
	s.Equal(true, testEnv.bl)
	s.Equal(uint8(1), testEnv.ui8)
	s.Equal(byte(1), testEnv.b)
	s.Equal(uint16(80), testEnv.ui16)
	s.Equal(uint32(1), testEnv.ui32)
	s.Equal(uint(1), testEnv.ui)
	s.Equal(uint64(1), testEnv.ui64)
	s.Equal(int8(-1), testEnv.i8)
	s.Equal(int16(-1), testEnv.i16)
	s.Equal(int32(-1), testEnv.i32)
	s.Equal(-1, testEnv.i)
	s.Equal(int64(-1), testEnv.i64)

	builder.WriteString("UINT16=1\n")
}

func (s *ContainerTestSuite) TestBuildRunsConstructor() {
	s.container.Set(func(container Container) *AnotherTestStruct {
		return container.Build(new(AnotherTestStruct)).(*AnotherTestStruct)
	})

	s.container.Compile()
	s.True(s.container.Get((*AnotherTestStruct)(nil)).(*AnotherTestStruct).called)
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
