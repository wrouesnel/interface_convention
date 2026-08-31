module github.com/wrouesnel/interface_convention

go 1.26

require (
	github.com/samber/lo v1.53.0
	go.uber.org/multierr v1.11.0
	go.yaml.in/yaml/v4 v4.0.0-rc.6
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
)

require (
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.1.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

replace go.yaml.in/yaml/v4 => github.com/wrouesnel/yaml.go-yaml/v4 v4.0.0-20260823094537-5260188c94a6
