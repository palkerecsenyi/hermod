package compiler

import (
	"errors"
	"fmt"
	"github.com/iancoleman/strcase"
	"hermod/encoder"
	"reflect"
)

func toEncoderType(typeName string) interface{} {
	switch typeName {
	case "string":
		return encoder.String("")
	case "integer":
		return encoder.Integer(0)
	case "boolean":
		return encoder.Boolean(0)
	}
	return nil
}

func getReflectedType(typeName string) reflect.Type {
	return reflect.TypeOf(toEncoderType(typeName))
}

func searchForUnit(typeName string, configs []*fileConfigPair) *unitDefinition {
	for _, config := range configs {
		for _, unit := range config.config.Units {
			if strcase.ToCamel(unit.Name) == strcase.ToCamel(typeName) {
				return &unit
			}
		}
	}
	return nil
}

func findTypeName(rawType string, repeated bool, configs []*fileConfigPair) (string, error) {
	fieldType := getReflectedType(rawType)
	var fieldTypeName string

	if fieldType == nil {
		linkedUnit := searchForUnit(rawType, configs)
		if linkedUnit != nil {
			fieldTypeName = strcase.ToCamel(linkedUnit.Name)
		} else {
			return "", errors.New(fmt.Sprintf("relationship type %s not found", rawType))
		}
	} else {
		fieldTypeName = "encoder." + fieldType.Name()
	}

	if repeated {
		fieldTypeName = "[]" + fieldTypeName
	}

	return fieldTypeName, nil
}
