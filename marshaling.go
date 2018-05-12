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
		value.jsonMarshalValue(m, &b)
	}
	b.WriteRune('}')
	m.graph[ws.Id()] = b.Bytes()
}

func (value *Undefined) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	b.WriteString("null")
}

func (value *Text) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	b.WriteString(strconv.Quote(value.value))
}

func (value *Number) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	b.WriteRune('"')
	b.WriteString(value.String())
	b.WriteRune('"')
}

func (value *Bool) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	b.WriteString(strconv.FormatBool(value.value))
}

func (value *Slice) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	b.WriteRune('[')
	for i := range value.elements {
		if i != 0 {
			b.WriteRune(',')
		}
		value.elements[i].value.jsonMarshalValue(m, b)
	}
	b.WriteRune(']')
}

func (value *Worksheet) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	// 1. We write the ID.
	b.WriteRune('"')
	b.WriteString(value.Id())
	b.WriteRune('"')

	// 2. We ensure this ws is included in the overall marshall.
	m.marshal(value)
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

		// for now, no support for slices
		if _, ok := field.typ.(*SliceType); ok {
			return fmt.Errorf("struct field %s: cannot StructScan slices (yet)", ft.Name)
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
		ctx := convertCtx{
			sourceFieldName: field.name,
			sourceType:      field.typ,
			destFieldName:   ft.Name,
			destType:        ft.Type,
		}
		value, err := convert(ctx, wsValue)
		if err != nil {
			return err
		}

		f.Set(value)
	}

	return nil
}

// convertCtx makes it easier to test than passing dest as reflect.StructField
// and source as *Field.
type convertCtx struct {
	sourceFieldName string
	sourceType      Type
	destFieldName   string
	destType        reflect.Type
}

func convert(ctx convertCtx, value Value) (reflect.Value, error) {
	if ctx.destType.Kind() == reflect.Ptr {
		ctx.destType = ctx.destType.Elem()
		v, err := convert(ctx, value)
		if err != nil {
			return v, err
		}
		locus := reflect.New(ctx.destType)
		locus.Elem().Set(v)
		return locus, nil
	}

	return value.structScanConvert(ctx)
}

func (value *Undefined) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	panic("should never be called")
}

func (value *Text) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	if ctx.destType.Kind() == reflect.String {
		return reflect.ValueOf(value.value), nil
	}
	return ctx.cannotConvert()
}

func (value *Bool) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	if ctx.destType.Kind() == reflect.Bool {
		return reflect.ValueOf(value.value), nil
	} else if ctx.destType.Kind() == reflect.String {
		return reflect.ValueOf(value.String()), nil
	}
	return ctx.cannotConvert()
}

func (value *Number) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	// to string
	if ctx.destType.Kind() == reflect.String {
		return reflect.ValueOf(value.String()), nil
	}

	// to floats
	if ctx.destType.Kind() == reflect.Float32 {
		if f, err := strconv.ParseFloat(value.String(), 32); err == nil {
			return reflect.ValueOf(float32(f)), nil
		}
		return ctx.valueOutOfRange()
	} else if ctx.destType.Kind() == reflect.Float64 {
		if f, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return reflect.ValueOf(f), nil
		}
		return ctx.valueOutOfRange()
	}

	// to ints
	if t, ok := ctx.sourceType.(*NumberType); ok && t.scale == 0 {
		var (
			i   int64
			err error
		)
		switch ctx.destType.Kind() {
		case reflect.Int:
			if i, err = strconv.ParseInt(value.String(), 0, 0); err == nil {
				return reflect.ValueOf(int(i)), nil
			}
		case reflect.Int8:
			if i, err = strconv.ParseInt(value.String(), 0, 8); err == nil {
				return reflect.ValueOf(int8(i)), nil
			}
		case reflect.Int16:
			if i, err = strconv.ParseInt(value.String(), 0, 16); err == nil {
				return reflect.ValueOf(int16(i)), nil
			}
		case reflect.Int32:
			if i, err = strconv.ParseInt(value.String(), 0, 32); err == nil {
				return reflect.ValueOf(int32(i)), nil
			}
		case reflect.Int64:
			if i, err := strconv.ParseInt(value.String(), 0, 64); err == nil {
				return reflect.ValueOf(int64(i)), nil
			}
		}
		if err != nil {
			return ctx.valueOutOfRange()
		}
	}

	// to uints
	if t, ok := ctx.sourceType.(*NumberType); ok && t.scale == 0 {
		if value.value < 0 {
			return ctx.valueOutOfRange()
		}

		var (
			i   uint64
			err error
		)
		switch ctx.destType.Kind() {
		case reflect.Uint:
			if i, err = strconv.ParseUint(value.String(), 0, 0); err == nil {
				return reflect.ValueOf(uint(i)), nil
			}
		case reflect.Uint8:
			if i, err = strconv.ParseUint(value.String(), 0, 8); err == nil {
				return reflect.ValueOf(uint8(i)), nil
			}
		case reflect.Uint16:
			if i, err = strconv.ParseUint(value.String(), 0, 16); err == nil {
				return reflect.ValueOf(uint16(i)), nil
			}
		case reflect.Uint32:
			if i, err = strconv.ParseUint(value.String(), 0, 32); err == nil {
				return reflect.ValueOf(uint32(i)), nil
			}
		case reflect.Uint64:
			if i, err := strconv.ParseUint(value.String(), 0, 64); err == nil {
				return reflect.ValueOf(uint64(i)), nil
			}
		}
		if err != nil {
			return ctx.valueOutOfRange()
		}
	}

	return ctx.cannotConvert()
}

func (value *Worksheet) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	if ctx.destType.Kind() != reflect.Struct {
		return ctx.cannotConvert("dest must be a struct")
	}
	newVal := reflect.New(ctx.destType)
	err := value.StructScan(newVal.Interface())
	if err != nil {
		return reflect.Value{}, err
	}
	return newVal.Elem(), nil
}

func (value *Slice) structScanConvert(ctx convertCtx) (reflect.Value, error) {
	return ctx.cannotConvert("not supported yet")
}

func (ctx convertCtx) valueOutOfRange() (reflect.Value, error) {
	return ctx.cannotConvert("value out of range")
}

func (ctx convertCtx) cannotConvert(msg ...string) (reflect.Value, error) {
	prefix := fmt.Sprintf("field %s to struct field %s: cannot convert %s to %s", ctx.sourceFieldName, ctx.destFieldName, ctx.sourceType, ctx.destType)
	if len(msg) == 0 {
		return reflect.Value{}, fmt.Errorf(prefix)
	} else {
		return reflect.Value{}, fmt.Errorf("%s, %s", prefix, msg[0])
	}
}
