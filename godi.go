// Package godi provides a simple, pluggable DI library for Go.
//
// godi handles Go types as interface{} parameters.  The method for passing these is:
//
// Interfaces: (*InterfaceName)(nil)
// Types: StructName{}
//
// For example: imagine a type Hippo which satisfies interface Animal.
//
// For callers that are interested in an animal, which is determined by DI
// to be a Hippo-type that will be created for each caller:
//
// godi.RegisterTypeImplementor((*Animal)(nil), Hippo{})
//
// Later, when a caller is interested in getting an animal:
//
// instance, err := godi.Resolve((*Animal)(nil))
//
// Likewise, if it is decided that all Animal-interested parties should get a
// created instance of Zebra
//
// zebra := &Zebra{Gender: 'Female', Age:4}
// godi.RegisterInstanceImplementor((*Animal)(nil), zebra)
//
// In this case, the all callers will resolve the Zebra.
//
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

// ExtractType is a helper method that returns the reflect.Type and [package].[type] name
// of an object
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
	name := fmt.Sprintf("%v", t)
	return formatType(name)
}

func Reset() {
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

// Initializable allows implementing an initialization interface on a type
// that will be called after creation
type Initializable interface {

	// GodiInit will be called to inialize an instance.
	GodiInit() error
}

// RegistrationContext is a scoped registration handler that allows registering
// of implementors in a scoped fashion, but downstream callers must have a refence to the scope to retrieve
// them
type RegistrationContext interface {
	Closable
	RegisterByName(target string, implementor string, cached bool) Closable
	RegisterInstanceImplementor(target interface{}, instance interface{}) (Closable, error)
	RegisterTypeImplementor(target interface{}, implementorType interface{}, cached bool, init InitializeCallback) (Closable, error)
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

// RegisterType registers a type with the DI framework.  This is required for using the type downstream, and generally
// is to be done in the init() method of the package you wish to use with DI.
//
// Example For interface:
//
// func init() {
//	  RegisterType((*MyInterfaceType)(nil))
// }
//
// Example For type:
//
// func init() {
//	  RegisterType(MyStructType{})
// }
//
func RegisterType(val interface{}) error {

	t, name := ExtractType(val)

	if typeMap[name] != nil {
		return errors.New("Already registered: " + name)
	}
	typeMap[name] = &t
	//fmt.Printf("Registered %s\n", name)
	return nil
}

// RegisterInstanceInitializer registers an object that will be invoked when a new object is created
// by the DI system.  See the InstanceInitializer interface.
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
// -init A callback to be called to initialize the object.
func RegisterTypeImplementor(target interface{}, implementorType interface{}, cached bool, init InitializeCallback) (Closable, error) {
	return currentContext.RegisterTypeImplementor(target, implementorType, cached, init)
}

// RegisterByName allow registration of targets and implmentors by name.  When the
// corresponding types are Registered, these registrations will be available.
// -target The target interface
// -implementor The implementing type
// -cached If true, returns the same instance for each type.
func RegisterByName(target string, implementor string, cached bool) Closable {
	return currentContext.RegisterByName(target, implementor, cached)
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
