[![Build and Test](https://github.com/wrouesnel/interface_convention/actions/workflows/integration.yml/badge.svg)](https://github.com/wrouesnel/interface_convention/actions/workflows/integration.yml)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/interface_convention/badge.svg?branch=main)](https://coveralls.io/github/wrouesnel/interface_convention?branch=main)

# Interface Convention

This is a helper package to simplify maintaining YAML based unmarshalling interfaces
which use type-discriminants when combined with a patched version of `go.yaml.in/yaml/v4`.

It allows you to express YAML configuration which looks like:

```yaml
my_object:
  contents:
  - type: interface_specialty_a
    config:
      specialA: hello
  - type: interface_specialty_b
    config:
      specialBInt: 10
```

and deserialize it in one shot to a Go lang struct defined as:

```go
type MyObject struct {
	Contents: []InterfaceSpecialties `yaml:"contents"`
}

type InterfaceSpecialties interface {
    Type() SpecialtyType	
}

type InterfaceSpecialtyA struct {
	SpecialA string `yaml:"specialA"`
}

type InterfaceSpecialtyB struct {
    SpecialA int `yaml:"specialBInt"`
}
```

It does require some boilerplate to make this work, but once setup it will generally just work
and vastly eases complicated configuration schemes. Namely, you want to use `github.com/abice/go-enum`
to define your enums.
