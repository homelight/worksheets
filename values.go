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
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/satori/go.uuid"
)

var (
	vUndefined = &Undefined{}
	vZero      = &Number{0, &NumberType{0}}
)

// RoundingMode describes the rounding mode to be used in an operation.
type RoundingMode string

const (
	ModeUp   RoundingMode = "up"
	ModeDown              = "down"
	ModeHalf              = "half"
)

// Value represents a runtime value.
type Value interface {
	// Type returns this value's type.
	Type() Type

	// Equal returns a comparison on this value against that value.
	Equal(that Value) bool

	// String returns a string representation of the value.
	String() string

	expression
	dbWriteValue() string
	jsonMarshalValue(m *marshaler, b *bytes.Buffer)
	structScanConvert(dests map[string]interface{}, ctx convertCtx) (reflect.Value, error)

	// assignableTo returns whether this value is assignable to type typ.
	//
	// For simplicity, we are not implementing automatic boxing right now
	// (e.g. text "foo" boxed to enum(some_enum, "foo")), and as a result
	// assignability checks must be dynamically calculated. For instance
	// []text may or may not be assignable to []some_enum depending on the
	// values contained in the slice. This has runtime impact, which we are
	// comfortable paying right now.
	assignableTo(typ Type) bool
}

var _ []Value = []Value{
	// Assert that all literals are Value.
	vUndefined,
	&Number{},
	&Text{},
	&Bool{},

	// Internals.
	&Slice{},
	&Worksheet{},
}

// Undefined represents an undefined value.
type Undefined struct{}

// Number represents a fixed decimal number.
type Number struct {
	value int64
	typ   *NumberType
}

// Text represents a string.
type Text struct {
	value string
}

// Bool represents a boolean.
type Bool struct {
	value bool
}

func (value *Bool) Value() bool {
	return value.value
}

func NewValue(value string) (Value, error) {
	reader := strings.NewReader(value)
	p := newParser(reader)
	lit, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	if reader.Len() != 0 {
		return nil, fmt.Errorf("expecting eof")
	}
	return lit, nil
}

func MustNewValue(value string) Value {
	lit, err := NewValue(value)
	if err != nil {
		panic(err)
	}
	return lit
}

func NewUndefined() Value {
	return vUndefined
}

func (value *Undefined) Type() Type {
	return &UndefinedType{}
}

func (value *Undefined) Equal(that Value) bool {
	_, ok := that.(*Undefined)
	return ok
}

func (value *Undefined) String() string {
	return "undefined"
}

// NewNumberFromString returns a new Number from a string representation.
func NewNumberFromString(value string) (*Number, error) {
	v, err := NewValue(value)
	if err != nil {
		return nil, err
	}
	n, ok := v.(*Number)
	if !ok {
		return nil, fmt.Errorf("not a number %s", value)
	}
	return n, nil
}

// NewNumberFromInt returns a new Number from int.
func NewNumberFromInt(num int) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromInt8 returns a new Number from int8.
func NewNumberFromInt8(num int8) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromInt16 returns a new Number from int16.
func NewNumberFromInt16(num int16) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromInt32 returns a new Number from int32.
func NewNumberFromInt32(num int32) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromInt64 returns a new Number from int64.
func NewNumberFromInt64(num int64) *Number {
	return &Number{num, &NumberType{0}}
}

// NewNumberFromUint returns a new Number from uint.
func NewNumberFromUint(num uint) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromUint8 returns a new Number from uint8.
func NewNumberFromUint8(num uint8) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromUint16 returns a new Number from uint16.
func NewNumberFromUint16(num uint16) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromUint32 returns a new Number from uint32.
func NewNumberFromUint32(num uint32) *Number {
	return &Number{int64(num), &NumberType{0}}
}

// NewNumberFromFloat32 returns a new Number from float32.
func NewNumberFromFloat32(num float32) *Number {
	return NewNumberFromFloat64(float64(num))
}

// NewNumberFromFloat64 returns a new Number from float64.
func NewNumberFromFloat64(num float64) *Number {
	value, err := NewNumberFromString(strconv.FormatFloat(num, 'f', -1, 64))
	if err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}
	return value
}

func (value *Number) Type() Type {
	return value.typ
}

func (value *Number) String() string {
	scale := value.typ.scale
	if scale == 0 {
		return strconv.FormatInt(value.value, 10)
	}

	var (
		s      string
		buffer bytes.Buffer
	)

	if value.value < 0 {
		s = strconv.FormatInt(-value.value, 10)
		buffer.WriteRune('-')
	} else {
		s = strconv.FormatInt(value.value, 10)
	}

	// We count down from most significant digit in the number we are generating.
	// For instance 123 with scale 3 means 0.123 so the most significant digit
	// is 0 (at index 4), then 1 (at index 3), and so on. While counting down,
	// we generate the correct representation, by using the digits of the value
	// or introducing 0s as necessery. We also add the period at the appropriate
	// place while iterating.
	var (
		i = scale + 1
		l = len(s)
	)
	if l > i {
		i = l
	}
	for i > 0 {
		if i == scale {
			buffer.WriteRune('.')
		}
		if i > l {
			buffer.WriteRune('0')
		} else {
			buffer.WriteByte(s[l-i])
		}
		i--
	}

	return buffer.String()
}

func (value *Number) scaleUp(scale int) int64 {
	if scale < value.typ.scale {
		panic("must round to lower scale")
	}

	v := value.value
	for s := value.typ.scale; s < scale; s++ {
		v *= 10
	}

	return v
}

func (left *Number) numericEqual(right *Number) bool {
	if left.typ.scale > right.typ.scale {
		return left.value == right.scaleUp(left.typ.scale)
	}
	if left.typ.scale < right.typ.scale {
		return left.scaleUp(right.typ.scale) == right.value
	}
	return left.value == right.value
}

func (value *Number) Equal(that Value) bool {
	typed, ok := that.(*Number)
	if !ok {
		return false
	}
	return value.numericEqual(typed)
}

func (left *Number) GreaterThan(right *Number) bool {
	if left.typ.scale > right.typ.scale {
		return left.value > right.scaleUp(left.typ.scale)
	}
	if left.typ.scale < right.typ.scale {
		return left.scaleUp(right.typ.scale) > right.value
	}
	return left.value > right.value
}

func (left *Number) GreaterThanOrEqual(right *Number) bool {
	return left.numericEqual(right) || left.GreaterThan(right)
}

func (left *Number) LessThan(right *Number) bool {
	return !left.numericEqual(right) && !left.GreaterThan(right)
}

func (left *Number) LessThanOrEqual(right *Number) bool {
	return left.numericEqual(right) || !left.GreaterThan(right)
}

func (left *Number) Plus(right *Number) *Number {
	scale := left.typ.scale
	if scale < right.typ.scale {
		scale = right.typ.scale
	}
	lv, rv := left.scaleUp(scale), right.scaleUp(scale)

	return &Number{lv + rv, &NumberType{scale}}
}

func (left *Number) Minus(right *Number) *Number {
	scale := left.typ.scale
	if scale < right.typ.scale {
		scale = right.typ.scale
	}
	lv, rv := left.scaleUp(scale), right.scaleUp(scale)

	return &Number{lv - rv, &NumberType{scale}}
}

func (left *Number) Mult(right *Number) *Number {
	scale := left.typ.scale + right.typ.scale
	return &Number{left.value * right.value, &NumberType{scale}}
}

func (value *Number) Round(mode RoundingMode, scale int) *Number {
	if value.typ.scale == scale {
		return value
	} else if value.typ.scale < scale {
		v := value.scaleUp(scale)
		return &Number{v, &NumberType{scale}}
	}

	factor := int64(1)
	for i := value.typ.scale; i != scale; i-- {
		factor = factor * 10
	}

	remainder := value.value % factor

	v := value.value
	for i := value.typ.scale; i != scale; i-- {
		v = v / 10
	}

	switch mode {
	case ModeDown:
		return &Number{v, &NumberType{scale}}

	case ModeUp:
		var up int64
		if remainder != 0 {
			up = 1
		}
		return &Number{v + up, &NumberType{scale}}

	case ModeHalf:
		var up int64
		threshold := 5 * factor / 10
		if remainder > 0 && threshold <= remainder {
			up = 1
		} else if remainder < 0 && remainder <= -threshold {
			up = -1
		}
		return &Number{v + up, &NumberType{scale}}
	}

	return value
}

func (left *Number) Div(right *Number, mode RoundingMode, scale int) *Number {
	// tempScale = max(left.typ.scale, scale + right.typ.scale) + 1
	tempScale := scale + right.typ.scale
	if left.typ.scale > tempScale {
		tempScale = left.typ.scale
	}
	tempScale = tempScale + 1

	// scale up left, integer division, and round correctly to finalize
	lv := left.scaleUp(tempScale)
	temp := &Number{lv / right.value, &NumberType{tempScale - right.typ.scale}}
	return temp.Round(mode, scale)
}

func NewText(value string) Value {
	return &Text{value}
}

func (value *Text) Type() Type {
	return &TextType{}
}

func (value *Text) Value() string {
	return value.value
}

func (value *Text) String() string {
	return strconv.Quote(value.value)
}

func (value *Text) Equal(that Value) bool {
	typed, ok := that.(*Text)
	if !ok {
		return false
	}
	return value.value == typed.value
}

func NewBool(value bool) Value {
	return &Bool{value}
}

func (value *Bool) Type() Type {
	return &BoolType{}
}

func (value *Bool) Equal(that Value) bool {
	typed, ok := that.(*Bool)
	if !ok {
		return false
	}
	return value.value == typed.value
}

func (value *Bool) String() string {
	return strconv.FormatBool(value.value)
}

type sliceElement struct {
	rank  int
	value Value
}

func (el sliceElement) String() string {
	return fmt.Sprintf("%d:%s", el.rank, el.value)
}

type Slice struct {
	id       string
	lastRank int
	typ      *SliceType
	elements []sliceElement
}

func newSlice(typ *SliceType) *Slice {
	return &Slice{
		id:  uuid.Must(uuid.NewV4()).String(),
		typ: typ,
	}
}

func newSliceWithIdAndLastRank(typ *SliceType, id string, lastRank int) *Slice {
	return &Slice{
		id:       id,
		typ:      typ,
		lastRank: lastRank,
	}
}

func (slice *Slice) Elements() []Value {
	var values []Value
	for _, element := range slice.elements {
		values = append(values, element.value)
	}
	return values
}

func (slice *Slice) doAppend(value Value) (*Slice, error) {
	// assignability check
	if err := canAssignTo("append", value, slice.typ.elementType); err != nil {
		return nil, err
	}

	// append
	nextRank := slice.lastRank + 1
	return &Slice{
		id:       slice.id,
		typ:      slice.typ,
		lastRank: nextRank,
		elements: append(slice.elements, sliceElement{
			rank:  nextRank,
			value: value,
		}),
	}, nil
}

func (value *Slice) doDel(index int) (*Slice, error) {
	if value == nil || index < 0 || len(value.elements) <= index {
		return nil, fmt.Errorf("index out of range")
	}

	var elements []sliceElement
	for i := 0; i < len(value.elements); i++ {
		if i != index {
			elements = append(elements, value.elements[i])
		}
	}
	return &Slice{
		id:       value.id,
		typ:      value.typ,
		lastRank: value.lastRank,
		// odd bug with this method... should investigate
		// elements: append(value.elements[:index], value.elements[index+1:]...),
		elements: elements,
	}, nil
}

func (value *Slice) Type() Type {
	return value.typ
}

func (value *Slice) Equal(that Value) bool {
	// Since slices structs are meant to be immutable, pointer equality is how
	// we check equality. See doXxx funcs for more details.
	return value == that
}

func (value *Slice) String() string {
	seen := make(map[string]bool)
	return value.stringerHelper(seen)
}

func (value *Slice) stringerHelper(seen map[string]bool) string {
	var buffer bytes.Buffer
	buffer.WriteRune('[')
	for i, element := range value.elements {
		if i != 0 {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(stringerHelperSwitch(seen, element.value))
	}
	buffer.WriteRune(']')
	return buffer.String()
}

func (ws *Worksheet) Type() Type {
	return ws.def
}

func (ws *Worksheet) Equal(that Value) bool {
	return ws == that
}

func (ws *Worksheet) String() string {
	seen := make(map[string]bool)
	return stringerHelperSwitch(seen, ws)
}

func (ws *Worksheet) stringerHelper(seen map[string]bool) string {
	if _, ok := seen[ws.Id()]; ok {
		return "<#ref>"
	}
	seen[ws.Id()] = true
	fieldNames := make([]string, 0, len(ws.data)-2)
	for index := range ws.data {
		if index != indexId && index != indexVersion {
			fieldNames = append(fieldNames, ws.def.fieldsByIndex[index].name)
		}
	}
	sort.Strings(fieldNames)

	var buffer bytes.Buffer
	buffer.WriteString("worksheet[")
	for i, fieldName := range fieldNames {
		value := ws.data[ws.def.fieldsByName[fieldName].index]

		if i != 0 {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(fieldName)
		buffer.WriteRune(':')
		buffer.WriteString(stringerHelperSwitch(seen, value))
	}
	buffer.WriteRune(']')
	return buffer.String()
}

func stringerHelperSwitch(seen map[string]bool, v Value) string {
	switch typedVal := v.(type) {
	case *Worksheet:
		return typedVal.stringerHelper(seen)
	case *Slice:
		return typedVal.stringerHelper(seen)
	default:
		return v.String()
	}
}
