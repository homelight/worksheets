// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worksheets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// Assert that Worksheets implement the json.Marshaler interface.
var _ json.Marshaler = &Worksheet{}

func (ws *Worksheet) MarshalJSON() ([]byte, error) {
	m := &marshaler{
		graph: make(map[string][]byte),
	}
	m.marshal(ws)

	var (
		not_first bool
		b         bytes.Buffer
	)
	b.WriteRune('{')
	for id, mashaled := range m.graph {
		if not_first {
			b.WriteRune(',')
		}
		not_first = true

		b.WriteRune('"')
		b.WriteString(id)
		b.WriteString(`":`)
		b.Write(mashaled)
	}
	b.WriteRune('}')
	return b.Bytes(), nil
}

type marshaler struct {
	graph map[string][]byte
}

func (m *marshaler) marshal(ws *Worksheet) {
	if _, ok := m.graph[ws.Id()]; ok {
		return
	}
	m.graph[ws.Id()] = nil

	var (
		not_first bool
		b         bytes.Buffer
	)
	b.WriteRune('{')
	for index, value := range ws.data {
		if not_first {
			b.WriteRune(',')
		}
		not_first = true

		b.WriteRune('"')
		b.WriteString(ws.def.fieldsByIndex[index].name)
		b.WriteString(`":`)
		m.marshalValue(&b, value)
	}
	b.WriteRune('}')
	m.graph[ws.Id()] = b.Bytes()
}

func (m *marshaler) marshalValue(b *bytes.Buffer, value Value) {
	switch v := value.(type) {
	case *Undefined:
		b.WriteString("null")

	case *Text:
		b.WriteString(strconv.Quote(v.value))

	case *Number:
		b.WriteRune('"')
		b.WriteString(value.String())
		b.WriteRune('"')

	case *Bool:
		b.WriteString(strconv.FormatBool(v.value))

	case *slice:
		b.WriteRune('[')
		for i := range v.elements {
			if i != 0 {
				b.WriteRune(',')
			}
			m.marshalValue(b, v.elements[i].value)
		}
		b.WriteRune(']')

	case *Worksheet:
		// 1. We write the ID.
		b.WriteRune('"')
		b.WriteString(v.Id())
		b.WriteRune('"')
		// 2. We ensure this ws is included in the overall marshall.
		m.marshal(v)

	default:
		panic(fmt.Sprintf("unexpected %v of %T", value, value))
	}
}

// WorksheetConverter is an interface used by StructScan.
type WorksheetConverter interface {
	// WorksheetConvert assigns a value from a worksheet field.
	//
	// The src value can be any defined worksheet value, e.g. text, bool,
	// number[n], or even a worksheet value.
	//
	// An error should be returned if the conversion cannot be done.
	WorksheetConvert(src Value) error
}

var worksheetConverterType = reflect.TypeOf((*WorksheetConverter)(nil)).Elem()

func (ws *Worksheet) StructScan(dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Type().Kind() != reflect.Ptr || v.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a *struct")
	}

	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		tag, ok := ft.Tag.Lookup("ws")

		if !ok {
			continue
		}

		if tag == "" {
			return fmt.Errorf("struct field %s: cannot have empty tag name", ft.Name)
		}

		field, ok := ws.def.fieldsByName[tag]
		if !ok {
			return fmt.Errorf("unknown field %s", tag)
		}

		// for now, no support for slices or worksheets
		if _, ok := field.typ.(*SliceType); ok {
			return fmt.Errorf("struct field %s: cannot StructScan slices (yet)", ft.Name)
		}

		if _, ok := field.typ.(*Definition); ok {
			return fmt.Errorf("struct field %s: cannot StructScan worksheets (yet)", ft.Name)
		}

		_, wsValue, _ := ws.get(tag)

		// undefined
		if _, ok := wsValue.(*Undefined); ok {
			if ft.Type.Kind() != reflect.Ptr {
				return fmt.Errorf("field %s to struct field %s: undefined into not nullable", tag, ft.Name)
			}
			f.Set(reflect.Zero(ft.Type))
			continue
		}

		// WorksheetConverter
		if ft.Type.AssignableTo(worksheetConverterType) {
			exporter := reflect.New(ft.Type.Elem()).Interface().(WorksheetConverter)
			if err := exporter.WorksheetConvert(wsValue); err != nil {
				return err
			}
			f.Set(reflect.ValueOf(exporter))
			continue
		} else if reflect.PtrTo(ft.Type).AssignableTo(worksheetConverterType) {
			exporter := reflect.New(ft.Type).Interface().(WorksheetConverter)
			if err := exporter.WorksheetConvert(wsValue); err != nil {
				return err
			}
			f.Set(reflect.ValueOf(exporter).Elem())
			continue
		}

		// default conversion
		value, err := convert(ft.Name, tag, ft.Type, field.typ, wsValue)
		if err != nil {
			return err
		}

		f.Set(value)
	}

	return nil
}

func convert(destFieldName, sourceFieldName string, destType reflect.Type, sourceType Type, value Value) (reflect.Value, error) {
	if destType.Kind() == reflect.Ptr {
		v, err := convert(destFieldName, sourceFieldName, destType.Elem(), sourceType, value)
		if err != nil {
			return v, err
		}
		locus := reflect.New(destType.Elem())
		locus.Elem().Set(v)
		return locus, nil
	}

	switch v := value.(type) {
	case *Text:
		if destType.Kind() == reflect.String {
			return reflect.ValueOf(v.value), nil
		}
	case *Bool:
		if destType.Kind() == reflect.Bool {
			return reflect.ValueOf(v.value), nil
		} else if destType.Kind() == reflect.String {
			return reflect.ValueOf(v.String()), nil
		}
	case *Number:
		// to string
		if destType.Kind() == reflect.String {
			return reflect.ValueOf(v.String()), nil
		}

		// to floats
		if destType.Kind() == reflect.Float32 {
			if f, err := strconv.ParseFloat(v.String(), 32); err == nil {
				return reflect.ValueOf(float32(f)), nil
			}
			return valueOutOfRange(destFieldName, sourceFieldName, destType, value)
		} else if destType.Kind() == reflect.Float64 {
			if f, err := strconv.ParseFloat(v.String(), 64); err == nil {
				return reflect.ValueOf(f), nil
			}
			return valueOutOfRange(destFieldName, sourceFieldName, destType, value)
		}

		// to ints
		if t, ok := sourceType.(*NumberType); ok && t.scale == 0 {
			var (
				i   int64
				err error
			)
			switch destType.Kind() {
			case reflect.Int:
				if i, err = strconv.ParseInt(v.String(), 0, 0); err == nil {
					return reflect.ValueOf(int(i)), nil
				}
			case reflect.Int8:
				if i, err = strconv.ParseInt(v.String(), 0, 8); err == nil {
					return reflect.ValueOf(int8(i)), nil
				}
			case reflect.Int16:
				if i, err = strconv.ParseInt(v.String(), 0, 16); err == nil {
					return reflect.ValueOf(int16(i)), nil
				}
			case reflect.Int32:
				if i, err = strconv.ParseInt(v.String(), 0, 32); err == nil {
					return reflect.ValueOf(int32(i)), nil
				}
			case reflect.Int64:
				if i, err := strconv.ParseInt(v.String(), 0, 64); err == nil {
					return reflect.ValueOf(int64(i)), nil
				}
			}
			if err != nil {
				return valueOutOfRange(destFieldName, sourceFieldName, destType, value)
			}
		}

		// to uints
		if t, ok := sourceType.(*NumberType); ok && t.scale == 0 {
			if v.value < 0 {
				return valueOutOfRange(destFieldName, sourceFieldName, destType, value)
			}

			var (
				i   uint64
				err error
			)
			switch destType.Kind() {
			case reflect.Uint:
				if i, err = strconv.ParseUint(v.String(), 0, 0); err == nil {
					return reflect.ValueOf(uint(i)), nil
				}
			case reflect.Uint8:
				if i, err = strconv.ParseUint(v.String(), 0, 8); err == nil {
					return reflect.ValueOf(uint8(i)), nil
				}
			case reflect.Uint16:
				if i, err = strconv.ParseUint(v.String(), 0, 16); err == nil {
					return reflect.ValueOf(uint16(i)), nil
				}
			case reflect.Uint32:
				if i, err = strconv.ParseUint(v.String(), 0, 32); err == nil {
					return reflect.ValueOf(uint32(i)), nil
				}
			case reflect.Uint64:
				if i, err := strconv.ParseUint(v.String(), 0, 64); err == nil {
					return reflect.ValueOf(uint64(i)), nil
				}
			}
			if err != nil {
				return valueOutOfRange(destFieldName, sourceFieldName, destType, value)
			}
		}
	default:
		panic(fmt.Sprintf("unexpected destType=%v, value=%v", destType, value))
	}

	return reflect.Value{}, fmt.Errorf("field %s to struct field %s: cannot convert %s to %s", sourceFieldName, destFieldName, value.Type(), destType)
}

func valueOutOfRange(destFieldName, sourceFieldName string, destType reflect.Type, value Value) (reflect.Value, error) {
	return reflect.Value{}, fmt.Errorf("field %s to struct field %s: cannot convert %s to %s, value out of range", sourceFieldName, destFieldName, value.Type(), destType)
}
