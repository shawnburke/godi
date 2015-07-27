package fbinject

import (
	"testing"

	"github.com/shawnburke/godi"
)

type I1 interface {
	CheckCheck() string
}

type D2 interface {
	Check() string
}

type T1 struct {
	Dep D2 `inject:""`
}

func (p T1) CheckCheck() string {
	return p.Dep.Check()
}

type TD2 struct {
}

func (p TD2) Check() string {
	return "hodor"
}

func TestFbInject(t *testing.T) {

	// setup the initializer
	var inject = NewFBInjectInstanceInitializer()
	deps := []interface{}{(*D2)(nil)}

	inject.AddInitializer(T1{}, deps)

	godi.RegisterInstanceInitializer(inject)

	// register the dependency
	godi.RegisterTypeImplementor((*I1)(nil), T1{}, false)
	godi.RegisterTypeImplementor((*D2)(nil), TD2{}, false)

	instance, err := godi.Resolve((*I1)(nil))

	if err != nil {
		t.Error(err.Error())
	}

	i1 := instance.(I1)

	if i1.CheckCheck() != "hodor" {
		t.Error("Expected hodor")
	}

}
