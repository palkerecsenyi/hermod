// Package encoder provides a lower level encoding subsystem to translate Go struct-based Hermod units into their
// inter-compatible platform-agnostic binary forms.
//
// You usually don't want to use any of the functions provided here directly. Instead, the code generated by the Hermod
// compiler creates convenient pre-typed wrappers that reference these functions, with support for encoding and decoding
// any of the units you've created. Furthermore, the service package will automatically handle encoding and decoding any
// units you specify as input/outputs in endpoints.
package encoder

import (
	"errors"
	"fmt"
	"reflect"
)

// Unit is essentially what's contained inside the YAML file used to defined Hermod units. It contains full definitions
// like the unit's name and all its fields.
type Unit struct {
	Name           string // a user-readable debug name for this Unit
	TransmissionId uint16 // a unique identifier for this Unit within scope
	Fields         []Field
}

// Field is the full definition of a field contained within a unit, as contained inside the YAML file used to define Hermod
// units.
type Field struct {
	Name     string
	FieldId  uint16
	Type     reflect.Value
	Extended bool // if true, increases maximum value length to 2^64 bytes. otherwise, limit is 2^36 bytes.
	Repeated bool // if true, allows multiple values in the style of a list
}

// FieldValue is used in FilledUnit to denote the specific user-provided value of a field.
type FieldValue struct {
	ParentUnit *FilledUnit
	Value      interface{}
}

// FilledUnit is like a unit but with the values for each of the fields also added.
type FilledUnit struct {
	*Unit
	Values map[Field]FieldValue
}

// EncodeUnit converts a Unit into a Hermod-encoded byte slice.
// [2 bytes transmission ID] then for each field value:
// [2 bytes field ID] [4 bytes content length in bytes (n)] [n bytes content]
func EncodeUnit(unit *FilledUnit) (*[]byte, error) {
	id := unit.TransmissionId
	encodedUnit := u16to8(id)

	for field, value := range unit.Values {
		encodedValue, err := encodeValue(value, field.Repeated)
		if err != nil {
			return nil, err
		}

		length := len(encodedValue)
		if length > 0xffff && field.Extended != true ||
			length > 0xffffffff && field.Extended == true {
			return nil, errors.New(fmt.Sprintf("value of %s over size limit", field.Name))
		}

		encodedUnit = *Add16ToSlice(field.FieldId, &encodedUnit)
		encodedUnit = *addLengthMarker(length, field.Extended, &encodedUnit)
		encodedUnit = append(encodedUnit, encodedValue...)
	}

	return &encodedUnit, nil
}

// DecodeUnit attempts (blindly) to decode a Hermod-encoded byte slice into a FilledUnit
// Errors are returned only for content issues. Structural issues may cause unexpected behaviour, panics, or even infinite loops!
func DecodeUnit(_rawUnit *[]byte, unit Unit) (*FilledUnit, error) {
	rawUnit := *_rawUnit
	filledUnit := FilledUnit{
		Unit:   &unit,
		Values: map[Field]FieldValue{},
	}
	intendedTransmissionId := SliceToU16(rawUnit[0:2])
	if intendedTransmissionId != unit.TransmissionId {
		return nil, errors.New(fmt.Sprintf("transmission ID %d did not match expected ID %d", intendedTransmissionId, unit.TransmissionId))
	}

	index := 2
	for {
		fieldId := SliceToU16(rawUnit[index : index+2])
		index += 2
		var field *Field
		for _, thisField := range unit.Fields {
			if thisField.FieldId == fieldId {
				field = &thisField
				break
			}
		}

		if field == nil {
			return nil, errors.New(fmt.Sprintf("field ID %d was not found", fieldId))
		}

		length := 0
		if field.Extended {
			length = int(SliceToU32(rawUnit[index : index+8]))
			index += 8
		} else {
			length = int(SliceToU32(rawUnit[index : index+4]))
			index += 4
		}

		rawValue := rawUnit[index:(index + length)]
		index += length

		decodedValue, err := decodeValue(field, rawValue)
		if err != nil {
			return nil, err
		}

		filledUnit.Values[*field] = FieldValue{
			ParentUnit: &filledUnit,
			Value:      decodedValue,
		}

		if index == len(rawUnit) {
			break
		}
	}

	return &filledUnit, nil
}
