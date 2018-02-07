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
	"sort"
	"strconv"
	"strings"

	"github.com/satori/go.uuid"
)

var (
	vZero = &Number{0, &NumberType{0}}
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
}

var _ []Value = []Value{
	// Assert that all literals are Value.
	&Undefined{},
	&Number{},
	&Text{},
	&Bool{},

	// Internals.
	&slice{},
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
	return &Undefined{}
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

func (value *Number) Type() Type {
	return value.typ
}

func (value *Number) Equal(that Value) bool {
	typed, ok := that.(*Number)
	if !ok {
		return false
	}
	return value.value == typed.value && value.typ.scale == typed.typ.scale
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

type slice struct {
	id       string
	lastRank int
	typ      *SliceType
	elements []sliceElement
}

func newSlice(typ *SliceType) *slice {
	return &slice{
		id:  uuid.NewV4().String(),
		typ: typ,
	}
}

func newSliceWithIdAndLastRank(typ *SliceType, id string, lastRank int) *slice {
	return &slice{
		id:       id,
		typ:      typ,
		lastRank: lastRank,
	}
}

func (value *slice) doAppend(element Value) (*slice, error) {
	if !element.Type().AssignableTo(value.typ.elementType) {
		return nil, fmt.Errorf("cannot append %s to %s", element.Type(), value.Type())
	}

	nextRank := value.lastRank + 1
	value.lastRank++

	slice := &slice{
		id:       value.id,
		typ:      value.typ,
		lastRank: value.lastRank,
		elements: append(value.elements, sliceElement{
			rank:  nextRank,
			value: element,
		}),
	}
	return slice, nil
}

func (value *slice) doDel(index int) (*slice, error) {
	if value == nil || index < 0 || len(value.elements) <= index {
		return nil, fmt.Errorf("index out of range")
	}

	var elements []sliceElement
	for i := 0; i < len(value.elements); i++ {
		if i != index {
			elements = append(elements, value.elements[i])
		}
	}
	return &slice{
		id:       value.id,
		typ:      value.typ,
		lastRank: value.lastRank,
		// odd bug with this method... should investigate
		// elements: append(value.elements[:index], value.elements[index+1:]...),
		elements: elements,
	}, nil
}

func (value *slice) Type() Type {
	return value.typ
}

func (value *slice) Equal(that Value) bool {
	// Since slices structs are meant to be immutable, pointer equality is how
	// we check equality. See doXxx funcs for more details.
	return value == that
}

func (value *slice) String() string {
	var buffer bytes.Buffer
	buffer.WriteRune('[')
	for i, element := range value.elements {
		if i != 0 {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(element.value.String())
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
	fieldNames := make([]string, 0, len(ws.data)-2)
	for index := range ws.data {
		if index != IndexId && index != IndexVersion {
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
		buffer.WriteString(value.String())
	}
	buffer.WriteRune(']')
	return buffer.String()
}
