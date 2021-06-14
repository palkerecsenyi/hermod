package compiler

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
)

func writeUnit(configs []*fileConfigPair, w *bytes.Buffer, unit *unitDefinition, packageName string) (imports []string, err error) {
	unitDefinitionName := strcase.ToLowerCamel((unit.Name) + "_Definition")

	imports = append(imports, fmt.Sprintf("%s/encoder", packageName))
	_writeln(w, fmt.Sprintf("// %s is used internally by Hermod to encode/decode data. Don't use this in your own code.", unitDefinitionName))
	_writeln(w, fmt.Sprintf("var %s = encoder.Unit{", unitDefinitionName))
	_writelni(w, 1, fmt.Sprintf("TransmissionId: %d,", unit.TransmissionId))
	_writelni(w, 1, fmt.Sprintf("Name: \"%s\",", unit.Name))

	_writelni(w, 1, "Fields: []encoder.Field{")
	var fieldTypeName string
	for _, field := range unit.Fields {
		fieldTypeName, err = findTypeName(field.RawType, field.Repeated, configs)
		if err != nil {
			return
		}

		_writelni(w, 2, "{")
		_writelni(w, 3, fmt.Sprintf("Name: \"%s\",", field.Name))
		_writelni(w, 3, fmt.Sprintf("FieldId: %d,", field.FieldId))
		_writelni(w, 3, fmt.Sprintf("Extended: %t,", field.Extended))
		_writelni(w, 3, fmt.Sprintf("Repeated: %t,", field.Repeated))
		imports = append(imports, "reflect")
		_writelni(w, 3, fmt.Sprintf("Type: reflect.ValueOf(*new(%s)),", fieldTypeName))
		_writelni(w, 2, "},")
	}
	_writelni(w, 1, "},")

	_writeln(w, "}")

	publicName := strcase.ToCamel(unit.Name)
	_writeln(w, fmt.Sprintf("type %s struct {", publicName))
	for _, field := range unit.Fields {
		fieldName := strcase.ToCamel(field.Name)
		fieldTypeName, err = findTypeName(field.RawType, field.Repeated, configs)
		if err != nil {
			return
		}

		_writelni(w, 1, fmt.Sprintf("%s\t%s", fieldName, fieldTypeName))
	}
	_writeln(w, "}")

	_writeln(w, fmt.Sprintf("func (d %s) GetDefinition() *encoder.Unit {", publicName))
	_writelni(w, 1, fmt.Sprintf("return &%s", unitDefinitionName))
	_writeln(w, "}")

	_writeln(w, fmt.Sprintf("func (d %s) Encode() (*[]byte, error) {", publicName))
	_writelni(w, 1, "return encoder.UserEncode(d)")
	_writeln(w, "}")

	_writeln(w, fmt.Sprintf("func (d %s) DecodeAbstract(data *[]byte) (encoder.UserFacingHermodUnit, error) {", publicName))
	_writelni(w, 1, "return encoder.UserDecode(d, data)")
	_writeln(w, "}")

	_writeln(w, fmt.Sprintf("func Decode%s(data *[]byte) (*%s, error) {", publicName, publicName))
	_writelni(w, 1, fmt.Sprintf("decodedData, err := encoder.UserDecode(%s{}, data)", publicName))
	_writelni(w, 1, "if err != nil {")
	_writelni(w, 2, "return nil, err")
	_writelni(w, 1, "}")
	_writelni(w, 1, fmt.Sprintf("u := decodedData.(%s)", publicName))
	_writelni(w, 1, "return &u, nil")
	_writeln(w, "}")

	_writeln(w, fmt.Sprintf("func New%s() *%s {", publicName, publicName))
	_writelni(w, 1, fmt.Sprintf("s := %s{}", publicName))
	_writelni(w, 1, "return &s")
	_writeln(w, "}")

	return
}
