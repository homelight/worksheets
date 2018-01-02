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
	"strconv"
	"strings"
)

var (
	vZero = &Number{0, &tNumberType{0}}
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

// Assert that all literals are Value.
var _ []Value = []Value{
	&Undefined{},
	&Number{},
	&Text{},
	&Bool{},
}

// Undefined represents an undefined value.
type Undefined struct{}

// Number represents a fixed decimal number.
type Number struct {
	value int64
	typ   *tNumberType
}

// Text represents a string.
type Text struct {
	value string
}

// Bool represents a boolean.
type Bool struct {
	value bool
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
	return &tUndefinedType{}
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
	s := strconv.FormatInt(value.value, 10)
	scale := value.typ.scale
	if scale == 0 {
		return s
	}

	// We count down from most significant digit in the number we are generating.
	// For instance 123 with scale 3 means 0.123 so the most significant digit
	// is 0 (at index 4), then 1 (at index 3), and so on. While counting down,
	// we generate the correct representation, by using the digits of the value
	// or introducing 0s as necessery. We also add the period at the appropriate
	// place while iterating.
	var (
		i      = scale + 1
		l      = len(s)
		buffer bytes.Buffer
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
	scale := left.typ.scale + right.typ.scale
	lv, rv := left.scaleUp(scale), right.scaleUp(scale)

	return &Number{lv + rv, &tNumberType{scale}}
}

func (left *Number) Minus(right *Number) *Number {
	scale := left.typ.scale + right.typ.scale
	lv, rv := left.scaleUp(scale), right.scaleUp(scale)

	return &Number{lv - rv, &tNumberType{scale}}
}

func (left *Number) Mult(right *Number) *Number {
	scale := left.typ.scale + right.typ.scale
	return &Number{left.value * right.value, &tNumberType{scale}}
}

func (value *Number) Round(mode RoundingMode, scale int) *Number {
	if value.typ.scale == scale {
		return value
	} else if value.typ.scale < scale {
		v := value.scaleUp(scale)
		return &Number{v, &tNumberType{scale}}
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
		return &Number{v, &tNumberType{scale}}

	case ModeUp:
		var up int64
		if remainder != 0 {
			up = 1
		}
		return &Number{v + up, &tNumberType{scale}}

	case ModeHalf:
		panic("not implemented")
	}

	return value
}

func NewText(value string) Value {
	return &Text{value}
}

func (value *Text) Type() Type {
	return &tTextType{}
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
	return &tBoolType{}
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
