package fbinject

import (
	"godi"

	"github.com/facebookgo/inject"
)

type InitItem struct {
	typeName     string
	dependencies []interface{}
}

type FBInjectInstanceInitializer struct {
	initializers map[string]*InitItem
}

func NewFBInjectInstanceInitializer() *FBInjectInstanceInitializer {
	return &FBInjectInstanceInitializer{
		initializers: make(map[string]*InitItem),
	}
}

func (p FBInjectInstanceInitializer) AddInitializer(target interface{}, dependencies []interface{}) {
	_, name := godi.ExtractType(target)
	item := InitItem{typeName: name, dependencies: dependencies}
	p.initializers[name] = &item
}

func (p FBInjectInstanceInitializer) CanInitialize(instance interface{}, typeName string) bool {
	return p.initializers[typeName] != nil
}

func (p FBInjectInstanceInitializer) Initialize(instance interface{}, typeName string) (interface{}, error) {

	var g inject.Graph

	if err := g.Provide(&inject.Object{Value: instance}); err != nil {
		return nil, err
	}

	if entry := p.initializers[typeName]; entry != nil {
		for _, v := range entry.dependencies {
			resolved, e1 := godi.Resolve(v)
			if e1 != nil {
				return nil, e1
			}
			obj := inject.Object{Value: resolved}
			e2 := g.Provide(&obj)
			if e2 != nil {
				return nil, e2
			}
		}
	}

	// construct the instance.
	//
	if e3 := g.Populate(); e3 != nil {
		return nil, e3
	}
	return instance, nil
}
