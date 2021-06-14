package encoder

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type TinyInteger uint8
type SmallInteger uint16
type Integer uint32
type BigInteger uint64

type TinySignedInteger int8
type SmallSignedInteger int16
type SignedInteger int32
type BigSignedInteger int64

type String string

type Boolean uint8

var True = Boolean(0xff)
var False = Boolean(0x00)

func encodeValue(value FieldValue, repeated bool) ([]byte, error) {
	if value.Value == nil {
		return []byte{}, nil
	}

	if repeated {
		var values []byte
		for i := 0; i < reflect.ValueOf(value.Value).Len(); i++ {
			encodedSingleValue, err := encodeValue(FieldValue{
				Value: reflect.ValueOf(value.Value).Index(i).Interface(),
			}, false)
			if err != nil {
				return nil, err
			}

			values = *addLengthMarker(len(encodedSingleValue), false, &values)
			values = append(values, encodedSingleValue...)
		}
		return values, nil
	}

	switch v := value.Value.(type) {
	case TinyInteger:
		return []byte{byte(v)}, nil
	case SmallInteger:
		return u16to8(uint16(v)), nil
	case Integer:
		return u32to8(uint32(v)), nil
	case BigInteger:
		return u64to8(uint64(v)), nil

	case TinySignedInteger:
		return []byte{byte(v)}, nil
	case SmallSignedInteger:
		return u16to8(uint16(v)), nil
	case SignedInteger:
		return u32to8(uint32(v)), nil
	case BigSignedInteger:
		return u64to8(uint64(v)), nil

	case String:
		return []byte(v), nil

	case Boolean:
		return []byte{byte(v)}, nil
	}

	if v, ok := value.Value.(UserFacingHermodUnit); ok {
		encodedUnit, err := UserEncode(v)
		if err != nil {
			return nil, err
		}
		return *encodedUnit, nil
	} else {
		fmt.Println(reflect.TypeOf(value.Value).NumMethod())
		fmt.Println(value.Value)
	}

	return nil, errors.New(fmt.Sprintf("type not supported for value %s", value.Value))
}

func decodeValue(field *Field, rawValue []byte) (interface{}, error) {
	if field.Repeated {
		if len(rawValue) == 0 {
			return []interface{}{}, nil
		}

		index := 0
		var items []interface{}
		for {
			length := int(sliceToU32(rawValue[index : index+4]))
			index += 4

			rawItem := rawValue[index:(index + length)]
			index += length

			searchField := *field
			searchField.Repeated = false
			searchField.Type = reflect.New(field.Type.Type().Elem()).Elem()
			decodedItem, err := decodeValue(&searchField, rawItem)
			if err != nil {
				return nil, err
			}

			items = append(items, decodedItem)

			if index == len(rawValue) {
				break
			}
		}

		return items, nil
	}

	finalTypeName := strings.Split(field.Type.Type().String(), ".")[1]
	finalTypeName = strings.ReplaceAll(finalTypeName, "[]", "")
	switch finalTypeName {
	case "TinyInteger":
		return TinyInteger(rawValue[0]), nil
	case "SmallInteger":
		return SmallInteger(sliceToU16(rawValue)), nil
	case "Integer":
		return Integer(sliceToU32(rawValue)), nil
	case "BigInteger":
		return BigInteger(sliceToU64(rawValue)), nil

	case "String":
		return String(rawValue), nil

	case "Boolean":
		return Boolean(rawValue[0]), nil
	}

	decodeMethod := field.Type.MethodByName("DecodeAbstract")
	if decodeMethod.Kind() == reflect.Func && !decodeMethod.IsZero() {
		newStruct := reflect.New(field.Type.Type())

		outArguments := newStruct.MethodByName("DecodeAbstract").Call([]reflect.Value{reflect.ValueOf(&rawValue)})
		data, err := outArguments[0], outArguments[1]
		if !err.IsNil() {
			return nil, err.Interface().(error)
		}

		return data.Interface(), nil
	}

	return nil, errors.New(fmt.Sprintf("unmatchable type for field %s", field.Name))
}
