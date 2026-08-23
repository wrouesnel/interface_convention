package interface_convention

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type ErrUnknownType struct {
	TypeName  string
	Interface interface{}
}

func (e ErrUnknownType) Error() string {
	return fmt.Sprintf("unknown object type for %T: %v", e.Interface, e.TypeName)
}

type UnmarshalError struct {
	Node *yaml.Node
	Err  error
}

func (u *UnmarshalError) Error() string {
	m := map[string]interface{}{}
	err := u.Node.Load(m)
	if err != nil {
		return fmt.Sprintf("Position: %v:%v\nError:%v\n", u.Node.Line, u.Node.Column, err.Error())
	}
	b, err := yaml.Dump(m)
	if err != nil {
		return fmt.Sprintf("Position: %v:%v\nError:%v\n", u.Node.Line, u.Node.Column, err.Error())
	}
	return fmt.Sprintf("Position: %v:%v\nError:%v\n%v", u.Node.Line, u.Node.Column, u.Err.Error(), string(b))
}

func (u *UnmarshalError) Unwrap() error {
	return u.Err
}
