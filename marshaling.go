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
		notFirst bool
		b        bytes.Buffer
	)
	b.WriteRune('{')
	for id, mashaled := range m.graph {
		if notFirst {
			b.WriteRune(',')
		}
		notFirst = true

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
		notFirst bool
		b        bytes.Buffer
	)
	b.WriteRune('{')
	for index, value := range ws.data {
		if notFirst {
			b.WriteRune(',')
		}
		notFirst = true

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

// wsDestination is used during structScan to properly populate reused non-pointer struct references
type wsDestination struct {
	dest interface{}
	loci []reflect.Value
}

type wsDestinationMap map[string]*wsDestination

func (wsdm wsDestinationMap) setAllDestinations() {
	for _, d := range wsdm {
		for _, locus := range d.loci {
			destPtr := reflect.ValueOf(d.dest)
			// dests are stored as pointers but we are setting non-pointer destinations
			locus.Set(destPtr.Elem())
		}
	}
}

func (wsdm wsDestinationMap) addDestination(ws *Worksheet, dest interface{}) {
	if _, ok := wsdm[ws.Id()]; ok {
		panic("incorrect usage: cannot add new destination multiple times")
	}
	wsdm[ws.Id()] = &wsDestination{dest, nil}
}

func (wsdm wsDestinationMap) addLocus(ws *Worksheet, locus reflect.Value) {
	wsdm[ws.Id()].loci = append(wsdm[ws.Id()].loci, locus)
}

// StructScanner stores state allowing overrides for scanning of registered types.
type StructScanner struct {
	converterRegistry map[reflect.Type]func(Value) (interface{}, error)
}

func NewStructScanner() *StructScanner {
	return &StructScanner{
		converterRegistry: make(map[reflect.Type]func(Value) (interface{}, error)),
	}
}

func (ss *StructScanner) RegisterConverter(t reflect.Type, converterFn func(Value) (interface{}, error)) {
	if _, ok := ss.converterRegistry[t]; ok {
		panic("incorrect usage: cannot add converter for type multiple times")
	}
	ss.converterRegistry[t] = converterFn
}

// structScanCtx keeps state for a single scan spanning potentially multiple worksheets through refs.
type structScanCtx struct {
	// dests stores refs to any worksheets that we have already scanned
	// for reuse (and cycle termination)
	dests wsDestinationMap
	// copy map from global registry for this run
	converters map[reflect.Type]func(Value) (interface{}, error)
}

func (ss *StructScanner) StructScan(ws *Worksheet, dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Type().Kind() != reflect.Ptr || v.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a *struct")
	}

	ctx := &structScanCtx{
		converters: ss.converterRegistry,
		dests:      make(wsDestinationMap),
	}

	ctx.dests.addDestination(ws, dest)

	err := ctx.structScan(ws)
	if err != nil {
		return err
	}

	ctx.dests.setAllDestinations()

	return nil
}

func (ws *Worksheet) StructScan(dest interface{}) error {
	ss := NewStructScanner()
	return ss.StructScan(ws, dest)
}

// getWsField allows us to get the ws field from either a ws tag
// or the StructField name itself, if the tag is not specified.
// It returns a boolean for fields that should be processed (were not explicitly/implicitly ignored).
func getWsField(ws *Worksheet, ft reflect.StructField) (*Field, bool, error) {
	tag, ok := ft.Tag.Lookup("ws")
	if ok {
		if tag == "" {
			return nil, false, fmt.Errorf("struct field %s: cannot have empty tag name", ft.Name)
		} else if tag == "-" {
			// explicitly ignored
			return nil, false, nil
		}
		field, ok := ws.def.fieldsByName[tag]
		if !ok {
			return nil, false, fmt.Errorf("struct field %s: unknown ws field %s", ft.Name, tag)
		}
		return field, true, nil
	} else {
		// no tag, use StructField name directly
		field, ok := ws.def.fieldsByName[ft.Name]
		if !ok {
			// don't blow up if not found, just ignore
			return nil, false, nil
		}
		return field, true, nil
	}
}

func (ctx *structScanCtx) structScan(ws *Worksheet) error {
	v := reflect.ValueOf(ctx.dests[ws.Id()].dest)
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)

		field, ok, err := getWsField(ws, ft)
		if err != nil {
			return err
		} else if !ok {
			continue
		}

		_, wsValue, _ := ws.get(field.name)

		// default conversion
		fieldCtx := structScanFieldCtx{
			sourceFieldName: field.name,
			sourceType:      field.typ,
			destFieldName:   ft.Name,
			destType:        ft.Type,
		}
		value, err := ctx.convert(fieldCtx, wsValue)
		if err != nil {
			return err
		}

		setOrDeferSet(ctx.dests, f, value, wsValue, ft.Type)
	}

	return nil
}

func setOrDeferSet(dests wsDestinationMap, f, v reflect.Value, wsValue Value, destType reflect.Type) {
	if childWs, ok := wsValue.(*Worksheet); ok && destType.Kind() != reflect.Ptr {
		dests.addLocus(childWs, f)
	} else {
		// since we allowed structScanConvert on the kind of types,
		// make sure we convert in case it's necessary
		f.Set(v.Convert(destType))
	}
}

// structScanFieldCtx makes it easier to test than passing dest as reflect.StructField
// and source as *Field.
type structScanFieldCtx struct {
	sourceFieldName string
	sourceType      Type
	destFieldName   string
	destType        reflect.Type
}

func (ctx *structScanCtx) convert(fieldCtx structScanFieldCtx, value Value) (reflect.Value, error) {
	// this needs to be inside convert to make sure we check for elems in slices.
	// we need to do it before pointer logic to make sure we return the same pointer.
	if ws, ok := value.(*Worksheet); ok {
		if savedDest, ok := ctx.dests[ws.Id()]; ok {
			// we have seen this before, just return the saved dest ptr to struct or struct
			if fieldCtx.destType.Kind() == reflect.Ptr {
				return reflect.ValueOf(savedDest.dest), nil
			} else {
				return reflect.ValueOf(savedDest.dest).Elem(), nil
			}
		}
	}

	// let undefined->ptr be handled by a converter, custom or standard
	if _, ok := value.(*Undefined); !ok && fieldCtx.destType.Kind() == reflect.Ptr {
		// for empty slice ptr, we special case and return nil
		if sliceVal, ok := value.(*Slice); ok && len(sliceVal.Elements()) == 0 {
			return reflect.Zero(fieldCtx.destType), nil
		}
		fieldCtx.destType = fieldCtx.destType.Elem()
		v, err := ctx.convert(fieldCtx, value)
		if err != nil {
			return v, err
		}
		locus := reflect.New(fieldCtx.destType)
		locus.Elem().Set(v)
		return locus, nil
	}

	// check to see if the caller specified an override for a type, and if so, apply it
	if converterFn, ok := ctx.converters[fieldCtx.destType]; ok {
		vInterface, err := converterFn(value)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(vInterface), nil
	}

	// if we have a type that uses a custom converter, use it instead of standard conversion
	if reflect.PtrTo(fieldCtx.destType).AssignableTo(worksheetConverterType) {
		exporter := reflect.New(fieldCtx.destType).Interface().(WorksheetConverter)
		if err := exporter.WorksheetConvert(value); err != nil {
			return reflect.Value{}, err
		}
		if ws, ok := value.(*Worksheet); ok {
			ctx.dests.addDestination(ws, exporter)
		}
		return reflect.ValueOf(exporter).Elem(), nil
	}

	return value.structScanConvert(ctx, fieldCtx)
}

func (value *Undefined) structScanConvert(_ *structScanCtx, ctx structScanFieldCtx) (reflect.Value, error) {
	if ctx.destType.Kind() != reflect.Ptr {
		ctx.sourceType = value.Type()
		return ctx.cannotConvert(fmt.Sprintf("dest must be a *%s", ctx.destType.Name()))
	}
	return reflect.Zero(ctx.destType), nil
}

func (value *Text) structScanConvert(_ *structScanCtx, ctx structScanFieldCtx) (reflect.Value, error) {
	if ctx.destType.Kind() == reflect.String {
		return reflect.ValueOf(value.value), nil
	}
	return ctx.cannotConvert()
}

func (value *Bool) structScanConvert(_ *structScanCtx, ctx structScanFieldCtx) (reflect.Value, error) {
	if ctx.destType.Kind() == reflect.Bool {
		return reflect.ValueOf(value.value), nil
	} else if ctx.destType.Kind() == reflect.String {
		return reflect.ValueOf(value.String()), nil
	}
	return ctx.cannotConvert()
}

func (value *Number) structScanConvert(_ *structScanCtx, ctx structScanFieldCtx) (reflect.Value, error) {
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

func (value *Worksheet) structScanConvert(ctx *structScanCtx, fieldCtx structScanFieldCtx) (reflect.Value, error) {
	if fieldCtx.destType.Kind() != reflect.Struct {
		return fieldCtx.cannotConvert("dest must be a struct")
	}

	newVal := reflect.New(fieldCtx.destType)
	ctx.dests.addDestination(value, newVal.Interface())
	err := ctx.structScan(value)
	if err != nil {
		return reflect.Value{}, err
	}
	return newVal.Elem(), nil
}

func (value *Slice) structScanConvert(ctx *structScanCtx, fieldCtx structScanFieldCtx) (reflect.Value, error) {
	if fieldCtx.destType.Kind() != reflect.Slice {
		return fieldCtx.cannotConvert("dest must be a slice")
	}
	if len(value.Elements()) == 0 {
		return reflect.Zero(fieldCtx.destType), nil
	}
	locus := reflect.New(fieldCtx.destType)
	locus.Elem().Set(reflect.MakeSlice(fieldCtx.destType, len(value.Elements()), len(value.Elements())))
	fieldCtx.destType = fieldCtx.destType.Elem()
	for i, wsElem := range value.Elements() {
		fieldCtx.sourceType = wsElem.Type()
		newVal, err := ctx.convert(fieldCtx, wsElem)
		if err != nil {
			return reflect.Value{}, err
		}
		setOrDeferSet(ctx.dests, locus.Elem().Index(i), newVal, wsElem, fieldCtx.destType)
	}
	return locus.Elem(), nil
}

func (ctx structScanFieldCtx) valueOutOfRange() (reflect.Value, error) {
	return ctx.cannotConvert("value out of range")
}

func (ctx structScanFieldCtx) cannotConvert(msg ...string) (reflect.Value, error) {
	prefix := fmt.Sprintf("field %s to struct field %s: cannot convert %s to %s", ctx.sourceFieldName, ctx.destFieldName, ctx.sourceType, ctx.destType)
	if len(msg) == 0 {
		return reflect.Value{}, fmt.Errorf(prefix)
	} else {
		return reflect.Value{}, fmt.Errorf("%s, %s", prefix, msg[0])
	}
}
