package encoder

import (
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type UserFacingHermodUnit interface {
	GetDefinition() *Unit
	DecodeAbstract(data *[]byte) (UserFacingHermodUnit, error)
}

func UserToFilledUnit(u UserFacingHermodUnit) (*FilledUnit, error) {
	filledUnit := FilledUnit{
		Unit:   u.GetDefinition(),
		Values: map[Field]FieldValue{},
	}

	v := reflect.ValueOf(u)
	dType := v.Type()

	for _, field := range filledUnit.Unit.Fields {
		for i := 0; i < dType.NumField(); i++ {
			if strcase.ToCamel(field.Name) == strcase.ToCamel(dType.Field(i).Name) {
				filledUnit.Values[field] = FieldValue{
					Value:      v.Field(i).Interface(),
					ParentUnit: &filledUnit,
				}
			}
		}
	}
	return &filledUnit, nil
}

func UserEncode(u UserFacingHermodUnit) (*[]byte, error) {
	filledUnit, err := UserToFilledUnit(u)
	if err != nil {
		return nil, err
	}

	return EncodeUnit(filledUnit)
}

func FilledUnitToUser(filledUnit *FilledUnit, u UserFacingHermodUnit) (UserFacingHermodUnit, error) {
	fieldMap := map[string]interface{}{}
	for field, value := range filledUnit.Values {
		fieldMap[strcase.ToCamel(field.Name)] = value.Value
	}

	result := &u
	err := mapstructure.Decode(fieldMap, result)
	return *result, err
}

func UserDecode(u UserFacingHermodUnit, data *[]byte) (UserFacingHermodUnit, error) {
	definition := u.GetDefinition()
	filledUnit, err := DecodeUnit(data, *definition)
	if err != nil {
		return nil, err
	}
	return FilledUnitToUser(filledUnit, u)
}
