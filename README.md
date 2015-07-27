# godi
Simple, Pluggable DI Framework for Golang

### Referencing Types and Interfaces

godi handles Go types as interface{} parameters.  The method for passing these is:

* Interfaces: `(*InterfaceName)(nil)`
* Types: `StructName{}`

### Basic Usage

For example: imagine a type `Hippo` which satisfies interface `Animal`.

For callers that are interested in an animal, which is determined by DI
to be a `Hippo`-type that will be created for each caller:

    godi.RegisterTypeImplementor((*Animal)(nil), Hippo{}, false)

Later, when a caller is interested in getting an `Animal`:

    instance, err := godi.Resolve((*Animal)(nil))

Note that `false` passed above says not to cache the created instance and instead create a new one for each caller.

Likewise, if it is decided that all Animal-interested parties should get a created instance of `Zebra`:

    zebra := &Zebra{Gender: 'Female', Age:4}
    godi.RegisterInstanceImplementor((*Animal)(nil), zebra)

In this case, the all callers will resolve the `Zebra`.

### Configuration-Based Registration

In some cases, it's desirable to declare implementors without having access to the loaded types or packages.  Godi handles this via string-named types in the following way.

Types are referenced as `[package].[type]`.  And only the leaf-most package counts.  So the `container/list/List` type is "list.List", and `net/Addr` is "net.Addr"

In order for types-as-strings to be available, godi must be made aware of them via `RegisterType`, typically within the package `init` method.

    func init() {
    	godi.RegisterType((*Animal)(nil))
    	godi.RegisterType(Hippo{})
    	godi.RegisterType(Zebra{})
    }

Clearly, this forces some coupling between godi and a package's types.  Unfortunately due to limitations in the Go type system, this is required.

Once types have been registered, string-based registrations can be done via `RegisterByName`:

    godi.RegisterByName("safari.Animal", "safari.Hippo", false)

Later, this will have the same result as the first `RegisterTypeImplementor` call above, provided that the types are registered.

In this way, you can configure godi lookups via a configuration file.

### Instance Initialization

By default, godi will create a zero-instance of your implementor types.  The Dude does not abide.

To handle this, godi supports the `InstanceInitializer` interface:

    type InstanceInitializer interface {
	   CanInitialize(instance interface{}, typeName string) bool
	   Initialize(instance interface{}, typeName string) (interface{}, error)
    }

Implementations of this interface can be passed to godi as:

    godi.RegisterInstanceInitializer(myInstanceInitializer)

When godi creates a zero-instance of an implementor type, it will call `CanInitialize` the method on any registered instance initializers, in the order in which they were registered.  The first implementation to return *true* from `CanInitialize`, will then receive a call to `Initailize`, and the process will halt.

Note that implementors _are not_ required to return the same instance they are passed.  In other words, the zero-instance can be discarded and an instance of the implementors choosing can be replaced.  For example, one created using the `New...` method.  In all cases, the instance will be passed, along with the type name for easy lookup.

#### Integration with Facebook Inject

godi includes integration with [Facebook Inject](https://github.com/facebookgo/inject), which is usable as follows:

Given a type like:

    type Zoo struct {
    	exhibit Animal `inject:""`
    }

Then, configure `godi/fbinject` as follows:

    import "godi/fbinject"

    var fbinject FBInjectInstanceInitializer = godi.FBInjectInstanceInitializer{}

    // Register the type that will be created (Zoo), and
    // the targets (interfaces) it depends on
    fbinject.AddInitializer(Zoo, []interface{}{(*Animal)(nil)})
    godi.RegisterInstanceInitializer(fbinject)

You may need to `go get github.com/facebookgo/inject`.

### Scopes and Unregistration

godi suppoorts creating registration scopes via the `CreateScope` method, which will return a scoped registration context.  Scoped contexts allow for registration of types and instances that will be checked before parent scopes are called.  In other words, they over-ride the parent scope.

Scopes, along with other registrations, return an instance that implements the `Closable` interface.  That is, calling `Close()` on the instances will remove the registration from the scope it was created in.

## Installation

    go get github.com/shawnburke/godi

## Tests

    go test ./...

## License

godi is available under the [MIT License](http://opensource.org/licenses/MIT)

## 
