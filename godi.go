package godi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

//
// Error strings
//
const (
	ErrorRegistrationNotFound = "NotFound"
)

//
// Global State and helpers
//
var typeMap = make(map[string]*reflect.Type)
var registrationCounter int
var rootContext = newregistrationContext(nil)
var currentContext = rootContext

func getRegisteredTypes() *map[string]*reflect.Type {
	return &typeMap
}

func ExtractType(val interface{}) (reflect.Type, string) {
	t := reflect.TypeOf(val)

	if !strings.Contains(typeToString(t), "rtype") {
		t = reflect.TypeOf(val)

	} else {
		t = val.(reflect.Type)
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := typeToString(t)
	return t, name
}

func typeToString(t reflect.Type) string {
	return fmt.Sprintf("%v", t)
}

func reset() {
	typeMap = make(map[string]*reflect.Type)
	rootContext.Reset()
	currentContext = rootContext
}

//
// Interfaces
//

// Closable is an interface for items that can be deterministically closed
type Closable interface {
	Close()
}

// RegistrationContext is a scoped registration handler that allows registering
// of implementors in a scoped fashion, but downstream callers must have a refence to the scope to retrieve
// them
type RegistrationContext interface {
	Closable
	RegisterPending(target string, implementor string, cached bool) Closable
	RegisterInstanceImplementor(target interface{}, instance interface{}) (Closable, error)
	RegisterTypeImplementor(target interface{}, implementorType interface{}, cached bool) (Closable, error)
	Resolve(target interface{}) (interface{}, error)
	CreateScope() RegistrationContext
	Reset()
}

// InstanceInitializer allows post-create access to zero-values
// created by the DI system
type InstanceInitializer interface {
	CanInitialize(instance interface{}, typeName string) bool
	Initialize(instance interface{}, typeName string) (interface{}, error)
}

// RegistrationToken allows removal of a registration by the caller
type RegistrationToken struct {
	context      *registrationContext
	registration *typeRegistration
}

// Close removes a registration from it's parent scope.
func (p *RegistrationToken) Close() {
	if p.context != nil {
		p.context.removeRegistration(p.registration)
		p.context = nil
	}
}

// Register registers a type with the DI framework.  This is required for using the type downstream, and generally
// is to be done in the init() method of the package you wish to use with DI.
//
// Example For interface:
//
// func init() {
//	  Register((*MyInterfaceType)(nil))
// }
//
// Example For type:
//
// func init() {
//	  Register(MyStructType{})
// }
//
func Register(val interface{}) error {

	t, name := ExtractType(val)

	if typeMap[name] != nil {
		return errors.New("Already registered: " + name)
	}
	typeMap[name] = &t
	//fmt.Printf("Registered %s\n", name)
	return nil
}

func RegisterInstanceInitializer(initializer InstanceInitializer) error {
	return currentContext.RegisterInstanceInitializer(initializer)
}

func instanceToType(instance interface{}) reflect.Type {
	t, _ := ExtractType(instance)
	return t
}

// RegisterInstanceImplementor registers an instance as the implementor of
// an interface for this scope
// -target The target interface
func RegisterInstanceImplementor(target interface{}, instance interface{}) (Closable, error) {
	return currentContext.RegisterInstanceImplementor(target, instance)
}

// RegisterTypeImplementor registers a type as the implementor of an interface for this scope
// -target The target interface
// -implementorType The implementing type
// -cached Set true to return the same instance for subsequent calls, false to create a new one each time
func RegisterTypeImplementor(target interface{}, implementorType interface{}, cached bool) (Closable, error) {
	return currentContext.RegisterTypeImplementor(target, implementorType, cached)
}

// RegisterPending allow registration of targets and implmentors by name.  When the
// corresponding types are Registered, these registrations will be available.
// -target The target interface
// -implementor The implementing type
// -cached If true, returns the same instance for each type.
func RegisterPending(target string, implementor string, cached bool) Closable {
	return currentContext.RegisterPending(target, implementor, cached)
}

// Resolve returns an instance of the requested interface, or an error
// -target The targetType
func Resolve(instance interface{}) (interface{}, error) {
	return currentContext.Resolve(instance)
}

// ResolveByName returns an instance of the requested interface, by name, like
// package.Type (e.g. myPackage.MyInterface)
func ResolveByName(target string) (interface{}, error) {
	reg := currentContext.findRegistration(target)
	if reg == nil {
		return nil, errors.New(ErrorRegistrationNotFound)
	}
	return currentContext.resolveCore(*reg.targetType.reflectType)
}

// CreateScope creates a new registration scope.
// -pushScope if true, this new scope will become global, until Close is called
func CreateScope(pushScope bool) RegistrationContext {

	var onclose closeHandler
	var newCtx *registrationContext

	if pushScope {
		onclose = func() {
			if currentContext == newCtx {
				currentContext = newCtx.parent
			}
		}
	}

	newCtx = currentContext.createScopeCore(onclose)
	return newCtx
}
