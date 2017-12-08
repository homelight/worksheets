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

// Value represents a runtime value.
type Value interface {
	// Type returns this value's type.
	Type() Type

	// Equals returns a comparison on this value against that value.
	Equals(that Value) bool

	// String returns a string representation of the value.
	String() string
}

// Assert that all literals are Value.
var _ []Value = []Value{
	&tUndefined{},
	&tNumber{},
	&tText{},
	&tBool{},
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
	return &tUndefined{}
}

func (value *tUndefined) Type() Type {
	return &tUndefinedType{}
}

func (value *tUndefined) Equals(that Value) bool {
	_, ok := that.(*tUndefined)
	return ok
}

func (value *tUndefined) String() string {
	return "undefined"
}

func (value *tNumber) Type() Type {
	return value.typ
}

func (value *tNumber) Equals(that Value) bool {
	typed, ok := that.(*tNumber)
	if !ok {
		return false
	}
	return value.value == typed.value && value.typ.scale == typed.typ.scale
}

func (value *tNumber) String() string {
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

func NewText(value string) Value {
	return &tText{value}
}

func (value *tText) Type() Type {
	return &tTextType{}
}

func (value *tText) String() string {
	return strconv.Quote(value.value)
}

func (value *tText) Equals(that Value) bool {
	typed, ok := that.(*tText)
	if !ok {
		return false
	}
	return value.value == typed.value
}

func (value *tBool) Type() Type {
	return &tBoolType{}
}

func (value *tBool) Equals(that Value) bool {
	typed, ok := that.(*tBool)
	if !ok {
		return false
	}
	return value.value == typed.value
}

func (value *tBool) String() string {
	return strconv.FormatBool(value.value)
}
