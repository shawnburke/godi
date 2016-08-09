package godi

import (
	"fmt"
	"reflect"
	"sync"
)

type InitializeCallback func(interface{}) (bool, error)

type typeRegistration struct {
	targetType  *typeInfo
	implType    *typeInfo
	initializer InitializeCallback
	instance    interface{}
	cached      bool
	id          int
	lock        sync.RWMutex
}

func (p *typeRegistration) ensureImplementor(impl reflect.Type, target reflect.Type) error {
	if !impl.Implements(target) {
		// since a method can be declared on the pointer, you need to check both
		if !reflect.PtrTo(impl).Implements(target) {
			return fmt.Errorf("Expected %v to implement %v", impl, target)
		}
	}
	return nil
}

func (p *typeRegistration) realize() (interface{}, bool, error) {

	// do we have an instance?
	//

	created := false

	create := func() interface{} {
		created = true
		return reflect.New(p.implType.Type()).Interface()
	}

	// only lock if we're a cached item
	// we lock here to make sure we don't create the item twice.
	//
	p.lock.RLock()

	var instance interface{} = p.instance
	needsCachedInstance := p.cached && p.instance == nil

	if needsCachedInstance {
		// if we need an instance, upgrade the lock
		p.lock.RUnlock()
		p.lock.Lock()

		// check again to avoid races
		if p.instance == nil {
			instance = create()
			p.instance = instance
		} else {
			instance = p.instance
		}
		defer p.lock.Unlock()
	} else {
		defer p.lock.RUnlock()
	}

	if !p.cached {
		instance = create()
	}
	return instance, created, nil
}
