package interface_convention

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"

	"github.com/samber/lo"
	"go.uber.org/multierr"
	"go.yaml.in/yaml/v4"
)

// CommonMapper should be implemented by Mapper to allow Mappers to be nested.
type CommonMapper interface {
	// YamlOption provides the necessary interfaces for common YAML loading with the provider
	YamlOption() yaml.Option
	// DefaultYamlOption provides the necessary interfaces to load in-built defaults for subtypes
	DefaultYamlOption() yaml.Option
}

type Mapper[K Discriminator, T InterfaceType[K]] interface {
	CommonMapper
	// InterfaceFor returns a new interface type T for the given discriminant
	InterfaceFor(K) (T, error)
	// DefaultInterfaceFor returns a new interface type T for the given discriminant with default configuration applied
	DefaultInterfaceFor(K) (T, error)
	// DiscriminantFor returns the discriminant for the object of type T
	DiscriminantFor(T) K
	// Names returns the list of all known discriminant names
	Names() []K
	// Types returns the reflected types of all known disciminants
	Types() []reflect.Type
	// TypeFor returns the reflected type for the given discriminant
	TypeFor(K) (reflect.Type, error)
}

// provider provides a default implementation of a TypeMapper and DefaultsMapper
type provider[K Discriminator, T InterfaceType[K]] struct {
	// typeMap maps discriminants to types
	typeMap map[K]reflect.Type
	// discMap maps types to discriminants
	discMap map[reflect.Type]K
	// defaultsMap
	defaultsMap map[K]*yaml.Node
	// interfacers are YAML custom marshaller/unmarshallers which apply a default merge tree
	interfacers yaml.Option
	// defaulters are YAML unmarshallers which apply a default merge tree
	defaulters []yaml.Option
	// mappers are subordinate mapper instances which are providing decoding and defaults for
	// any nested interface types.
	mappers []CommonMapper
}

// NewProvider initializes a new object provider which can initialize objects with a given set of defaults. Errors are returned if a default file cannot be
// loaded, but the provider will be functional. If a name collision in the discriminant is found, nil is returned as well as an error.
func NewProvider[K Discriminator, T InterfaceType[K]](values []T, defaultFunc func(K) ([]byte, error), mappers ...CommonMapper) (*provider[K, T], error) {
	var err error
	defaulters := make([]yaml.Option, 0)

	typeMap := map[K]reflect.Type{}
	discMap := map[reflect.Type]K{}
	defaultsMap := map[K]*yaml.Node{}

	hasNameCollisions := false
	for _, t := range values {
		objectType := reflect.Indirect(reflect.ValueOf(t)).Type()
		if existingType, found := typeMap[t.Type()]; found {
			err = multierr.Append(err, fmt.Errorf("discriminant name collision: %v is identified with %v so %v cannot be registered", existingType.String(), t.Type(), objectType.String()))
			hasNameCollisions = true
		}
		typeMap[t.Type()] = objectType
		discMap[objectType] = t.Type()

		if defaultFunc != nil {
			// Build the default providers for the YAML
			defaultConfig, lerr := defaultFunc(t.Type())
			if lerr != nil {
				err = multierr.Append(err, fmt.Errorf("no default for type %T could be loaded: %w", t, lerr))
			}

			typNode := new(yaml.Node)
			loader, err := yaml.NewLoader(bytes.NewReader(defaultConfig), yaml.WithV4Defaults())
			if err != nil {
				return nil, fmt.Errorf("could not configure loader for built in default: %v %w", objectType, err)
			}

			if err := loader.Load(typNode); errors.Is(err, io.EOF) {

			} else if err != nil {
				return nil, fmt.Errorf("could not load built in default: %v (%s) %w", objectType, t.Type(), err)
			} else {
				defaultsMap[t.Type()] = typNode
				defaulters = append(defaulters, DefaultUnmarshaler(objectType, typNode))
			}
		}
	}

	if hasNameCollisions {
		return nil, err
	}

	if mappers == nil {
		mappers = []CommonMapper{}
	}

	return &provider[K, T]{
		typeMap,
		discMap,
		defaultsMap,
		InterfaceConvention[K, T](typeMap),
		defaulters,
		mappers[:],
	}, nil
}

func CommonProvider[K Discriminator, T InterfaceType[K]](typesFn func() []T,
	defaults fs.FS, providers ...CommonMapper) Mapper[K, T] {

	p, err := NewProvider(typesFn(), func(discriminator K) ([]byte, error) {
		return fs.ReadFile(defaults, fmt.Sprintf("%v.yml", discriminator))
	}, providers...)
	if err != nil {
		panic(err)
	}
	return p
}

// YamlOption returns the provider's unified YAML decoding options
func (d *provider[K, T]) YamlOption() yaml.Option {
	options := []yaml.Option{}
	for _, mapper := range d.mappers {
		options = append(options, mapper.YamlOption())
	}
	options = append(options, d.interfacers)

	return yaml.Options(options...)
}

// YamlOption returns the provider's unified yaml default setting options
func (d *provider[K, T]) DefaultYamlOption() yaml.Option {
	options := []yaml.Option{}
	for _, mapper := range d.mappers {
		options = append(options, mapper.DefaultYamlOption())
	}

	options = append(options, d.defaulters...)

	return yaml.Options(options...)
}

// TypeFor returns the reflected type for a given discriminant value
func (d *provider[K, T]) TypeFor(k K) (reflect.Type, error) {
	t, found := d.typeMap[k]
	if !found {
		return nil, ErrUnknownType{TypeName: fmt.Sprintf("%v", k), Interface: new(T)}
	}
	return t, nil
}

// InterfaceFor initializes a new empty object subtype which provides the overall interface.
// It specifically is so data - discriminator types - can instantiate objects indirectly.
func (d *provider[K, T]) InterfaceFor(k K) (T, error) {
	var result T
	t, found := d.typeMap[k]
	if !found {
		return result, ErrUnknownType{TypeName: fmt.Sprintf("%v", k), Interface: new(T)}
	}
	r := reflect.New(t).Interface()

	result = r.(T)

	return result, nil
}

// DefaultInterfaceFor initializes a new object subtype which implements the interface T
// and provides default configuration.
func (d *provider[K, T]) DefaultInterfaceFor(k K) (T, error) {
	var result T
	t, found := d.typeMap[k]
	if !found {
		return result, ErrUnknownType{TypeName: fmt.Sprintf("%v", k), Interface: new(T)}
	}
	r := reflect.New(t).Interface()

	defaultsNode := d.defaultsMap[k]

	// Important: DefaultYamlOption is needed to get defaults to be applied.
	if err := defaultsNode.Load(r, d.YamlOption(), d.DefaultYamlOption()); err != nil {
		return result, err
	}

	result = r.(T)

	return result, nil
}

func (d *provider[K, T]) DiscriminantFor(i T) K {
	t := reflect.Indirect(reflect.ValueOf(i)).Type()
	return d.discMap[t]
}

// Names returns the currently known names of types
func (d *provider[K, T]) Names() []K {
	return lo.MapToSlice(d.typeMap, func(key K, value reflect.Type) K {
		return key
	})
}

// Types returns the currently known types
func (d *provider[K, T]) Types() []reflect.Type {
	return lo.MapToSlice(d.typeMap, func(key K, value reflect.Type) reflect.Type {
		return value
	})
}

// New is the generic function to create a new object from provider with the correct type and default config when initialized
// by object type.
func New[E any, EP interface {
	InterfaceType[K]
	*E
}, U Mapper[K, T], K Discriminator, T InterfaceType[K]](provider U) (*E, error) {
	// Extract the type and load the defaults
	var sample E

	// Call the interface builder method...
	var err error
	var result any
	result, err = provider.DefaultInterfaceFor(EP(&sample).Type())
	// ...and then narrow the interface via this hack (which we forcibly make true elsewhere)
	return result.(*E), err
}
