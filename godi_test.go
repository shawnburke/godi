package godi

import (
	"strconv"
	"strings"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type (
	I1 interface {
		F1() string
	}
	T1 struct {
		s string
	}
	T2 struct {
		f2 bool
	}
	T3 struct {
		n int
	} 
	I2 interface {
		Bar()
	}
	TestInitializer struct {}
	
	GoDiTestSuite struct {
		suite.Suite
	}
)

var (
	_ Initializable = &T3{}
	initS = "hodor"
)

func (p T1) F1() string {
	return p.s
}

func (p T2) F1() string {
	return "t2"
}

func (p T2) F2() string {
	return "t2f2"
}

func (p *T3) Initialize() bool {
	p.n = 42
	return false
}

func (p T3) F1() string {
	return strconv.Itoa(p.n)
}

func (p TestInitializer) CanInitialize(instance interface{}, typeName string) bool {
	if typeName == "godi.T1" {
		return true
	}
	return false
}

func (p TestInitializer) Initialize(instance interface{}, typeName string) (interface{}, error) {

	if typeName == "godi.T1" {
		t1 := instance.(*T1)
		t1.s = initS
		return t1, nil
	}
	return instance, nil
}

func (s *GoDiTestSuite) SetupTest() {
	Reset()
}

func (s *GoDiTestSuite) TestRegisterType() {
	RegisterType((*I1)(nil))
	count := len(*getRegisteredTypes())

	assert.Equal(s.T(), 1, count)
}

func (s *GoDiTestSuite) TestRegisterDupe() {
	RegisterType(T1{})

	err := RegisterType(T1{})

	assert.NotNil(s.T(), err)
}

func (s *GoDiTestSuite) TestResolveInstance() {
	i1 := (*I1)(nil)
	t1 := &T1{s: "foobarx"}

	res, err := RegisterInstanceImplementor(i1, t1)
	assert.Nil(s.T(), err)

	t1_r, err2 := Resolve(i1)
	assert.Nil(s.T(), err2)

	t1_val := t1_r.(I1)
	str := t1_val.F1()
	assert.Equal(s.T(), t1.s, str)

	res.Close()

	_, err3 := Resolve(i1)
	assert.NotNil(s.T(), err3)
}

func (s *GoDiTestSuite) TestResolveInstanceFail() {
	i1 := (*I1)(nil)
	t1 := &T1{s: "foobarx"}

	res, _ := RegisterInstanceImplementor(i1, t1)

	t2, err2 := Resolve((*I2)(nil))
	assert.Nil(s.T(), t2)
	assert.Equal(s.T(), ErrorRegistrationNotFound, err2.Error())
	
	res.Close()
}

func (s *GoDiTestSuite) TestResolveOverride() {
	i1 := (*I1)(nil)
	t1 := &T1{s: "foobar1"}
	t2 := &T1{s: "foobar2"}

	res, err := RegisterInstanceImplementor(i1, t1)
	assert.Nil(s.T(), err)

	t1_r, err2 := Resolve(i1)
	assert.Nil(s.T(), err2)

	t1_val := t1_r.(I1)
	str := t1_val.F1()
	assert.Equal(s.T(), t1.s, str)

	res2, err2 := RegisterInstanceImplementor(i1, t2)
	t1_r, _ = Resolve(i1)
	assert.Equal(s.T(), t1.s, str)

	res2.Close()

	t1_r, _ = Resolve(i1)
	assert.Equal(s.T(), t1.s, str)

	res.Close()

	_, err3 := Resolve(i1)
	assert.NotNil(s.T(), err3)
}

func (s *GoDiTestSuite) TestResolveType() {
	i1 := (*I1)(nil)
	t1 := T1{}

	res, err := RegisterTypeImplementor(i1, t1, false, nil)
	assert.Nil(s.T(), err)

	t1_r, err2 := Resolve(i1)
	assert.NotNil(s.T(), t1_r)
	assert.Nil(s.T(), err2)

	t1_val := t1_r.(I1)

	str := t1_val.F1()
	assert.Equal(s.T(), t1.s, str)

	res.Close()

	_, err3 := Resolve(i1)
	assert.NotNil(s.T(), err3)
}

func (s *GoDiTestSuite) TestInstanceInitializer() {
	init := TestInitializer{}

	RegisterInstanceInitializer(init)

	i1 := (*I1)(nil)
	RegisterTypeImplementor(i1, T1{}, false, nil)
	t1_r, _ := Resolve(i1)
	t1_c := t1_r.(*T1)

	assert.Equal(s.T(), initS, t1_c.s)
}

func (s *GoDiTestSuite) TestResolvePendingFail() {
	defer func() {
		if e := recover(); e != nil {
			assert.False(s.T(), strings.Contains(e.(string), "I1"))
			assert.True(s.T(), strings.Contains(e.(string), "di.T2"))
		}
	}()

	RegisterByName("godi.I1", "godi.T2", false)

	i1 := (*I1)(nil)

	RegisterType(i1)
	Resolve(i1)
	s.T().Error("Expected panic.")
}

func (s *GoDiTestSuite) TestResolvePending() {
	RegisterByName("godi.I1", "godi.T2", false)

	i1 := (*I1)(nil)
	e1 := RegisterType(i1)
	assert.Nil(s.T(), e1)
	
	e2 := RegisterType(T2{})
	assert.Nil(s.T(), e2)

	r1, _ := Resolve(i1)
	r2 := r1.(I1).F1()
	assert.Equal(s.T(), "t2", r2)
}

func (s *GoDiTestSuite) TestCreateScope() {
	i1 := (*I1)(nil)
	t1 := T1{}

	e1 := RegisterType(i1); 
	assert.Nil(s.T(), e1)
	
	e2 := RegisterType(T1{})
	assert.Nil(s.T(), e2)

	RegisterInstanceImplementor(i1, t1)

	r1, _ := Resolve(i1)
	r2 := r1.(I1).F1()
	assert.Equal(s.T(), "", r2)

	// push a scope
	s2 := CreateScope(true)

	t2 := T2{}
	RegisterInstanceImplementor(i1, t2)

	r3, _ := Resolve(i1)
	r2 = r3.(I1).F1()
	assert.Equal(s.T(), "t2", r2)

	s2.Close()
}

func (s *GoDiTestSuite) TestFormatType() {
	typeName := "*list.List"

	typeName = formatType(typeName)
	assert.Equal(s.T(), "list.List", typeName)
}

func (s *GoDiTestSuite) TestInitializerInterface() {
	i1 := (*I1)(nil)
	RegisterTypeImplementor(i1, T3{}, true, nil)

	r3, _ := Resolve(i1)
	r2 := r3.(I1).F1()
	assert.Equal(s.T(), "42", r2)
}

func (s *GoDiTestSuite) TestInitializeCallback() {
	i1 := (*I1)(nil)

	init := func(inst interface{}) (bool, error) {
		t3 := inst.(*T3)
		t3.n = 100
		return false, nil
	}

	RegisterTypeImplementor(i1, T3{}, true, init)

	r3, _ := Resolve(i1)
	r2 := r3.(I1).F1()
	assert.Equal(s.T(), "100", r2)
}

func TestGoDiTestSuite(t *testing.T) {
    suite.Run(t, new(GoDiTestSuite))
}