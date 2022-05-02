package compiler

import "gopkg.in/yaml.v2"

type fieldDefinition struct {
	Name     string
	FieldId  uint16 `yaml:"id"`
	Extended bool
	RawType  string `yaml:"type"`
	Repeated bool
}

type unitDefinition struct {
	Name           string
	TransmissionId uint16 `yaml:"id"`
	Fields         []fieldDefinition
}

type endpointArgumentDefinition struct {
	UnitName string `yaml:"unit"`
	Streamed bool
}

type endpointDefinition struct {
	Path string
	Id   uint16
	In   endpointArgumentDefinition
	Out  endpointArgumentDefinition
}

type serviceDefinition struct {
	Name      string
	Endpoints []endpointDefinition
}

type config struct {
	Package  string
	Units    []unitDefinition
	Services []serviceDefinition
}

func parseFile(contents []byte) (*config, error) {
	c := config{}
	err := yaml.Unmarshal(contents, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
