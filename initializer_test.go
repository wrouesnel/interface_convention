package interface_convention_test

import (
	"embed"
	"io/fs"
	"os"

	"github.com/wrouesnel/interface_convention"

	"github.com/samber/lo"
	"go.yaml.in/yaml/v4"
	. "gopkg.in/check.v1"
)

var _ = Suite(&ProviderSuite{})

type ProviderSuite struct {
}

// Basic set of interface convention functions
type Discriminant string

func (d Discriminant) IsValid() bool {
	return true
}

type I interface {
	Type() Discriminant
}

type S struct {
	S string `yaml:"s"`
}

func (s *S) Type() Discriminant {
	return "s-type"
}

type P struct{}

func (p *P) Type() Discriminant {
	return "p-type"
}

type X struct{}

func (x *X) Type() Discriminant {
	return "x-type"
}

// jobDefaults is the packaged defaults for job configurations.
//
//go:embed testdata/defaults/*.yml
var defaults embed.FS

func Types() []I {
	return []I{
		new(S),
		new(P),
		new(X),
	}
}

type ListOf struct {
	ListOf []I `yaml:"list_of"`
}

func Provider() interface_convention.Mapper[Discriminant, I] {
	return interface_convention.CommonProvider(Types, lo.Must(fs.Sub(defaults, "testdata/defaults")))
}

// TestProvider tests that the basic decoding cycle with a provider works.
func (p *ProviderSuite) TestProviderGeneric(c *C) {
	loader := lo.Must(yaml.NewLoader(lo.Must(os.Open("testdata/test-provider.yml")),
		Provider().YamlOption(), Provider().DefaultYamlOption()))

	r := new(ListOf)
	err := loader.Load(&r)
	c.Assert(err, IsNil)

	// Check the list comes back sensibly
	c.Check(r.ListOf[0].(*P).Type(), Equals, Discriminant("p-type"))
	c.Check(r.ListOf[1].(*S).S, Equals, "default value")
	c.Check(r.ListOf[2].(*S).S, Equals, "non-default config")
	c.Check(r.ListOf[3].(*X).Type(), Equals, Discriminant("x-type"))
}
