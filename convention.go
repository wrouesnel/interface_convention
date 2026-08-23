// package interface_convention provides a convention for registering discriminated type serializations
// for interfaces, given a comparable type discriminator.
package interface_convention

import (
	"errors"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v4"
)

// Discriminator is the type specification for how interface discriminators work.
type Discriminator interface {
	~string
	IsValid() bool
}

// InterfaceType is the interface to use for objects which implement the interface convention.
type InterfaceType[K Discriminator] interface {
	// Type returns the discriminant type identify of an instance
	Type() K
}

// InterfaceConvention generates a yaml.Option which provides a custom marshaller for the given interface,
// rather than concrete type.
func InterfaceConvention[K Discriminator, T InterfaceType[K]](typeMap map[K]reflect.Type) yaml.Option {

	// marshaler simply substitutes the object being encoded for a type discriminated version
	marshaler := func(i any) (any, error) {
		if i == nil {
			return nil, nil
		}
		// type parameters guarantee this is valid
		ic := (i).(T)

		value := struct {
			Type   K
			Config any
		}{
			ic.Type(),
			i,
		}

		return value, nil
	}

	unmarshaler := func(out any, node *yaml.Node) error {
		var contentType K
		var configNode *yaml.Node
		// We have a fairly raw representation here, so what we want to do is look for pairs of nodes.
		for i := 0; i < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "type":
				if i+1 < len(node.Content) {
					contentType = K(node.Content[i+1].Value)
				}
			case "config":
				if i+1 < len(node.Content) {
					configNode = node.Content[i+1]
				}
			}
		}

		// If it's valid then the provider can supply it
		if !contentType.IsValid() {
			return &UnmarshalError{
				Node: node,
				Err:  errors.New("no content type could be found"),
			}
		}

		var err error
		// We should have been handed a pointer to the interface for our input,
		// so this scary looking dereference works.
		concreteTyp, ok := typeMap[contentType]
		if !ok {
			return &UnmarshalError{
				Node: node,
				Err:  fmt.Errorf("content type %v cannot be mapped to any known type", contentType),
			}
		}

		// Only remake the object if the interface is nil, otherwise we want to
		// unmarshal into it. This trips right over Go's weird handling of pointers
		// to interfaces which are nil and so is quite verbose
		if reflect.ValueOf(out).Elem().IsNil() {
			*(out.(*T)) = reflect.New(concreteTyp).Interface().(T)
			if err != nil {
				return &UnmarshalError{
					Node: node,
					Err:  err,
				}
			}
		}

		// Replace the decoding context.
		return &yaml.SubstituteError{
			Node:        configNode,
			Dereference: true,
		}
	}
	var interfaceType T
	typ := reflect.TypeOf(&interfaceType).Elem()

	return yaml.Options(yaml.WithCustomTypeMarshaler(typ, marshaler), yaml.WithCustomTypeUnmarshaler(typ, unmarshaler))
}

// DefaultUnmarshaler creates an unmarshaller which handles a custom reflect.Type
func DefaultUnmarshaler(typ reflect.Type, defaults *yaml.Node) yaml.Option {
	// Configure the default document as a defaults node (note: this assumes raw defaults - you will need more code
	// if you get fancy).
	const anchorName = "defaults"
	defaults = defaults.Content[0]
	defaults.Anchor = anchorName

	// Make the unmarshaller
	unmarshaler := func(out any, node *yaml.Node) error {
		// Make a copy of the top-level node and then inject a merge key for the defaults configuration
		newnode := *node

		// We have to randomize the value field of the merge tag because this injection takes place
		// past when merge tags would be processed, *and* because if we have nested structs inheriting
		// defaults multiple times, then it's possible to have multiple merge tags in the tree.
		autoNum := 0
		mergeTagValue := fmt.Sprintf("_%s_%v", anchorName, autoNum)
		knownValues := map[string]struct{}{}
		for _, n := range node.Content {
			knownValues[n.Value] = struct{}{}
			// generate autonums till we find one which doesn't collide
			for found := true; found != false; _, found = knownValues[mergeTagValue] {
				autoNum += 1
				mergeTagValue = fmt.Sprintf("_%s_%v", anchorName, autoNum)
			}
		}

		// Inject a top level merge to the configuration pointing to the defaults tree
		newnode.Content = append(node.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!merge",
				Value: mergeTagValue,
			},
			&yaml.Node{
				Kind:  yaml.AliasNode,
				Alias: defaults,
				Value: anchorName,
			},
		)

		// Notify the decoder to re-run decoding
		return &yaml.SubstituteError{
			Node: &newnode,
			Once: true,
		}
	}
	return yaml.WithCustomTypeUnmarshaler(typ, unmarshaler)
}
