package godi

import (
	"fmt"
	"reflect"
)

type InitializeCallback func(interface{}) bool

type typeRegistration struct {
	targetType  *typeInfo
	implType    *typeInfo
	initializer InitializeCallback
	instance    interface{}
	cached      bool
	id          int
}

func (p *typeRegistration) ensureImplementor(impl reflect.Type, target reflect.Type) error {
	if !impl.Implements(target) {
		return fmt.Errorf("Expected %v to implement %v", impl, target)
	}
	return nil
}

func (p *typeRegistration) realize() (interface{}, bool, error) {

	// do we have an instance?
	//
	var i interface{}
	created := false
	if p.instance == nil || !p.cached {
		i = reflect.New(p.implType.Type()).Interface()
		created = true
	} else {
		i = p.instance
	}
	p.instance = i
	return i, created, nil
}
