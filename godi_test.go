package godi

import (
	"fmt"
	"strings"
	"testing"
)

type I1 interface {
	F1() string
}

type T1 struct {
	s string
}

func (p T1) F1() string {
	return p.s
}

type T2 struct {
	f2 bool
}

func (p T2) F1() string {
	return "t2"
}

func (p T2) F2() string {
	return "t2f2"
}

func TestRegister(t *testing.T) {
	reset()

	Register((*I1)(nil))
	count := len(*getRegisteredTypes())

	if count != 1 {
		t.Error(fmt.Sprintf("Expected 1 types, got %d", count))
	}
}

func TestRegisterDupe(t *testing.T) {
	reset()
	Register(T1{})

	err := Register(T1{})

	if err == nil {
		t.Error("Expected Error")
	}
}

func TestResolveInstance(t *testing.T) {
	reset()

	i1 := (*I1)(nil)
	t1 := &T1{s: "foobarx"}

	res, err := RegisterInstanceImplementor(i1, t1)

	if err != nil {
		t.Error("Expected reg")
	}

	t1_r, err2 := Resolve(i1)
	if err2 != nil {
		t.Error("Error resolving: " + err2.Error())
	}

	t1_val := t1_r.(I1)
	str := t1_val.F1()

	if str != t1.s {
		t.Error("Got " + str)
	}

	res.Close()

	_, err3 := Resolve(i1)

	if err3 == nil {
		t.Error("Expected unregistration error")
	}

}

type I2 interface {
	Bar()
}

func TestResolveInstanceFail(t *testing.T) {
	reset()

	i1 := (*I1)(nil)
	t1 := &T1{s: "foobarx"}

	res, _ := RegisterInstanceImplementor(i1, t1)

	t2, err2 := Resolve((*I2)(nil))

	if t2 != nil {
		t.Error("unexpected value for t2")
	}

	if err2.Error() != ErrorRegistrationNotFound {
		t.Error(fmt.Sprintf("Error resolving: %v", err2))
	}

	res.Close()

}

func TestResolveOverride(t *testing.T) {
	reset()

	i1 := (*I1)(nil)
	t1 := &T1{s: "foobar1"}
	t2 := &T1{s: "foobar2"}

	res, err := RegisterInstanceImplementor(i1, t1)

	if err != nil {
		t.Error("Expected reg")
	}

	t1_r, err2 := Resolve(i1)
	if err2 != nil {
		t.Error("Error resolving: " + err2.Error())
	}

	t1_val := t1_r.(I1)
	str := t1_val.F1()

	if str != t1.s {
		t.Error("Got " + str)
	}

	res2, err2 := RegisterInstanceImplementor(i1, t2)
	t1_r, _ = Resolve(i1)
	if str != t1.s {
		t.Error("Got " + str)
	}

	res2.Close()

	t1_r, _ = Resolve(i1)
	if str != t1.s {
		t.Error("Got " + str)
	}

	res.Close()

	_, err3 := Resolve(i1)

	if err3 == nil {
		t.Error("Expected unregistration error")
	}

}
func TestResolveType(t *testing.T) {
	reset()

	i1 := (*I1)(nil)
	t1 := T1{}

	res, err := RegisterTypeImplementor(i1, t1, false)

	if err != nil {
		t.Error("Expected reg")
	}

	t1_r, err2 := Resolve(i1)
	if err2 != nil || t1_r == nil {
		t.Error("Error resolving: " + err2.Error())
	}

	t1_val := t1_r.(I1)

	str := t1_val.F1()

	if str != t1.s {
		t.Error("Got " + str)
	}

	res.Close()

	_, err3 := Resolve(i1)

	if err3 == nil {
		t.Error("Expected unregistration error")
	}

}

type TestInitializer struct {
}

func (p TestInitializer) CanInitialize(instance interface{}, typeName string) bool {
	if typeName == "godi.T1" {
		return true
	}
	return false
}

var initS = "hodor"

func (p TestInitializer) Initialize(instance interface{}, typeName string) (interface{}, error) {

	if typeName == "godi.T1" {
		t1 := instance.(*T1)
		t1.s = initS
		return t1, nil
	}
	return instance, nil
}

func TestInstanceInitializer(t *testing.T) {
	reset()

	init := TestInitializer{}

	RegisterInstanceInitializer(init)

	i1 := (*I1)(nil)
	RegisterTypeImplementor(i1, T1{}, false)
	t1_r, _ := Resolve(i1)
	t1_c := t1_r.(*T1)

	if t1_c.s != initS {
		t.Error("Expected " + initS)
	}

}

func TestResolvePendingFail(t *testing.T) {

	defer func() {
		if e := recover(); e != nil {
			if strings.Contains(e.(string), "I1") {
				t.Error("Didn't expect I1")
			} else if !strings.Contains(e.(string), "di.T2") {
				t.Error("Expected T2")
			}
		}
	}()

	reset()

	RegisterPending("godi.I1", "godi.T2", false)

	i1 := (*I1)(nil)

	Register(i1)
	Resolve(i1)
	t.Error("Expected panic.")
}

func TestResolvePending(t *testing.T) {
	reset()
	RegisterPending("godi.I1", "godi.T2", false)

	i1 := (*I1)(nil)
	if e1 := Register(i1); e1 != nil {
		t.Error(e1)
	}
	if e2 := Register(T2{}); e2 != nil {
		t.Error(e2)
	}

	r1, _ := Resolve(i1)
	r2 := r1.(I1).F1()
	if r2 != "t2" {
		t.Error(fmt.Sprintf("pending resolve fail %v", r2))
	}
}

func TestCreateScope(t *testing.T) {
	reset()
	i1 := (*I1)(nil)
	t1 := T1{}

	if e1 := Register(i1); e1 != nil {
		t.Error(e1)
	}

	if e2 := Register(T1{}); e2 != nil {
		t.Error(e2)
	}

	RegisterInstanceImplementor(i1, t1)

	r1, _ := Resolve(i1)
	r2 := r1.(I1).F1()
	if r2 != "" {
		t.Error(fmt.Sprintf("pending resolve fail %v", r2))
	}

	// push a scope
	s2 := CreateScope(true)

	t2 := T2{}
	RegisterInstanceImplementor(i1, t2)

	r3, _ := Resolve(i1)
	r2 = r3.(I1).F1()
	if r2 != "t2" {
		t.Error(fmt.Sprintf("pending resolve fail %v", r2))
	}

	s2.Close()

}
