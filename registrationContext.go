package godi

import (
	"container/list"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

/// ---------------------------
///
/// registrationContext is the main class in the package
///
/// It manages a list of registrations and knows how to fall through to parent registrations
///
/// ---------------------------

type closeHandler func()

type registrationContext struct {
	parent        *registrationContext
	registrations map[string]*list.List
	initializers  *list.List
	onclose       closeHandler
	rwlock        sync.RWMutex
}

var _ RegistrationContext = &registrationContext{}

func newregistrationContext(parent *registrationContext) *registrationContext {
	p := &registrationContext{
		registrations: map[string]*list.List{},
		initializers:  list.New(),
	}
	if parent != nil {
		p.parent = parent
	}
	return p
}

//
// Initializer stuff
//

func (p *registrationContext) RegisterInstanceInitializer(initializer InstanceInitializer) error {
	p.initializers.PushFront(initializer)
	return nil
}

var initializableType, _ = ExtractType((*Initializable)(nil))

func (p *registrationContext) initializeInstance(instance interface{}, typeReg *typeRegistration) (interface{}, error) {

	// order of initialization is:
	// 1. Init callback
	// 2. Initialize ctor
	// 3. InstanceInitializer

	var err error
	callInitializers := true

	if typeReg.initializer != nil {
		callInitializers, err = typeReg.initializer(instance)
		if err != nil && !callInitializers {
			// if there is no other option for initializing, we should panic and stop the whole thing
			panic(fmt.Sprintf("Error with initializer for %s: %s", typeReg.implType.typeName, err.Error()))
		}
	}

	if callInitializers {
		if init, ok := instance.(Initializable); ok {
			if initErr := init.GodiInit(); initErr != nil {
				// if the built-in intializer fails, we are in big trouble...panic!
				//

				panic(fmt.Sprintf("Error initializing '%s' (registered for target '%s'): %v", typeReg.implType.typeName, typeReg.targetType.typeName, initErr))
			}
		}

		if callInitializers {
			l := p.initializers
			for e := l.Front(); e != nil; e = e.Next() {
				init := e.Value.(InstanceInitializer)
				if init != nil && init.CanInitialize(instance, typeReg.implType.typeName) {
					return init.Initialize(instance, typeReg.implType.typeName)
				}
			}

			if p.parent != nil {
				return p.parent.initializeInstance(instance, typeReg)
			}
		}
	}
	return instance, nil
}

//
// Helpers for managing registration list
//

func (p *registrationContext) addRegistration(reg *typeRegistration) {

	p.rwlock.Lock()
	defer p.rwlock.Unlock()
	tn := reg.targetType.typeName
	var l = p.registrations[tn]

	if l == nil {
		l = list.New()
		p.registrations[tn] = l
	}

	l.PushFront(reg)
}

func (p *registrationContext) findRegistration(typeName string) *typeRegistration {
	p.rwlock.RLock()
	defer p.rwlock.RUnlock()

	typeName = formatType(typeName)
	l := p.registrations[typeName]
	if l == nil || l.Len() == 0 {
		return nil
	}

	return l.Front().Value.(*typeRegistration)
}

func (p *registrationContext) removeRegistration(reg *typeRegistration) bool {

	p.rwlock.Lock()
	defer p.rwlock.Unlock()

	l := p.registrations[reg.targetType.typeName]
	if l == nil || l.Len() == 0 {
		return false
	}

	// Iterate through the type list looking for an ID match.
	// This is worst case O(n) and could be made O(1) with a little more work.
	// However, given this list should only be a couple of elements long, it won't buy much
	//
	for e := l.Front(); e != nil; e = e.Next() {
		r := e.Value.(*typeRegistration)
		if reg.id == r.id {
			l.Remove(e)
			return true
		}
	}
	return false
}

//
// Registration Stuff
//

func (p *registrationContext) RegisterByName(target string, implmentor string, cached bool) Closable {

	registrationCounter++
	tr := &typeRegistration{
		targetType: newtypeInfo(target, nil),
		implType:   newtypeInfo(implmentor, nil),
		cached:     cached,
		id:         registrationCounter,
	}

	p.addRegistration(tr)
	return &RegistrationToken{context: p, registration: tr}
}

func (p *registrationContext) RegisterInstanceImplementor(target interface{}, instance interface{}) (Closable, error) {
	t := instanceToType(target)

	rt := instanceToType(instance)

	registrationCounter++
	tr := &typeRegistration{
		targetType: newtypeInfo("", &t),
		implType:   newtypeInfo("", &rt),
		instance:   instance,
		cached:     true,
		id:         registrationCounter,
	}

	if err := tr.ensureImplementor(rt, t); err != nil {
		panic(err.Error())
	}

	p.addRegistration(tr)
	return &RegistrationToken{context: p, registration: tr}, nil
}

func (p *registrationContext) RegisterTypeImplementor(target interface{}, impl interface{}, cached bool, init InitializeCallback) (Closable, error) {

	t := instanceToType(target)
	implementor := instanceToType(impl)
	registrationCounter++
	tr := &typeRegistration{
		targetType:  newtypeInfo("", &t),
		implType:    newtypeInfo("", &implementor),
		initializer: init,
		cached:      cached,
		id:          registrationCounter,
	}

	if err := tr.ensureImplementor(implementor, t); err != nil {
		panic(err.Error())
	}

	p.addRegistration(tr)
	return &RegistrationToken{context: p, registration: tr}, nil
}

func (p *registrationContext) Resolve(target interface{}) (interface{}, error) {
	t := instanceToType(target)
	return p.resolveCore(t)
}

func (p *registrationContext) resolveCore(t reflect.Type) (interface{}, error) {
	name := typeToString(t)

	reg := p.findRegistration(name)

	if reg == nil && p.parent != nil {
		return p.parent.Resolve(t)
	}

	if reg != nil {
		raw, created, err := reg.realize()
		if err != nil {
			return nil, err
		}
		if created {
			return p.initializeInstance(raw, reg)
		}
		return raw, nil
	}
	return nil, errors.New(ErrorRegistrationNotFound)
}

func (p *registrationContext) Close() {

	p.rwlock.Lock()
	if p.registrations != nil {

		if p.onclose != nil {
			p.onclose()
			p.onclose = nil
		}

		p.parent = nil
	}
	// have to release because of the lock in reset.
	p.rwlock.Unlock()
	p.Reset()
}

func (p *registrationContext) createScopeCore(onclose func()) *registrationContext {
	ctx := newregistrationContext(p)
	if onclose != nil {
		ctx.onclose = onclose
	}
	return ctx
}

func (p *registrationContext) CreateScope() RegistrationContext {
	ctx := newregistrationContext(p)
	var rc RegistrationContext = ctx
	return rc
}

func (p *registrationContext) Reset() {
	p.rwlock.Lock()
	defer p.rwlock.Unlock()

	p.registrations = make(map[string]*list.List)
	p.initializers = list.New()
}

/// ----------------
///
/// End registrationContext
///
/// ----------------
