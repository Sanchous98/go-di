package di

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/suite"
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

type ContainerTestSuite struct {
	suite.Suite
	container PrecompiledGlobalState
}

func (s *ContainerTestSuite) SetupTest() { s.container = NewContainer() }

func (s *ContainerTestSuite) TestResolverBinding() {
	s.container.Set(func(cntr Container) *TestStruct {
		testStruct := new(TestStruct)
		testStruct.Dependency = cntr.Get(testStruct.Dependency).(*AnotherTestStruct)
		testStruct.Dependency2 = *cntr.Get(testStruct.Dependency2).(*AnotherTestStruct)
		testStruct.Dependency3 = cntr.Get(testStruct.Dependency3).(*AnotherTestStruct)
		testStruct.Dependency4 = *cntr.Get(testStruct.Dependency4).(*AnotherTestStruct)

		return testStruct
	})
	s.container.Set(func(cntr Container) *AnotherTestStruct { return new(AnotherTestStruct) })
	s.container.Compile()
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.Dependency3)
	s.NotNil(testStruct.Dependency4)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func (s *ContainerTestSuite) TestServiceBinding() {
	s.container.Set(new(TestStruct))
	s.container.Set(new(AnotherTestStruct))
	s.container.Compile()
	testStruct := s.container.Get((*TestStruct)(nil)).(*TestStruct)
	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.Dependency3)
	s.NotNil(testStruct.Dependency4)
	s.Len(s.container.All(), s.container.(*serviceContainer).resolvedNum)
}

func (s *ContainerTestSuite) TestAutoWiring() {
	testStruct := new(TestStruct)
	s.container.Set(testStruct)
	s.container.Compile()

	s.NotNil(testStruct.Dependency)
	s.NotNil(testStruct.Dependency2)
	s.NotNil(testStruct.Dependency3)
	s.NotNil(testStruct.Dependency4)
}

func (s *ContainerTestSuite) TestSelfReferences() {
	testStruct := new(AnotherTestStruct)
	s.container.Set(testStruct)
	s.container.Compile()
	s.NotNil(testStruct.Container)
	s.IsType(new(serviceContainer), testStruct.Container)
}

func (s *ContainerTestSuite) Test_loadEnv() {
	s.container.(*serviceContainer).loadEnv(bufio.NewReader(bytes.NewReader([]byte("APP_ENV=dev\nDB_NAME=test"))))
	s.EqualValues("dev", s.container.GetParam("APP_ENV"))
	s.EqualValues("test", s.container.GetParam("DB_NAME"))
}

func (s *ContainerTestSuite) TestCompileEvents() {
	s.container.PreCompile(func(event Event) { s.NotNil(event.GetElement()) }, 0)
	s.container.PostCompile(func(event Event) { s.NotNil(event.GetElement()) }, 0)
	s.container.Compile()
}

func TestContainer(t *testing.T) { suite.Run(t, new(ContainerTestSuite)) }
