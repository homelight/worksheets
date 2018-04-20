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
	"fmt"
)

// Type represents the type of a value.
type Type interface {
	// AssignableTo reports whether a value of the type is assignable to type u.
	AssignableTo(u Type) bool

	// String returns a string representation of the type.
	String() string

	dbReadValue(l *loader, value string) (Value, error)
}

// NamedType represents types which are uniquely identified by their names, such
// worksheet types, or enums.
type NamedType interface {
	Type

	// Name returns the name of the named type.
	Name() string
}

// Assert that all types (which are not 'named') implement the Type interface.
var _ []Type = []Type{
	&UndefinedType{},
	&TextType{},
	&BoolType{},
	&NumberType{},
	&SliceType{},
}

// Assert that named types implement the NamedType.
var _ []NamedType = []NamedType{
	&Definition{},
	&EnumType{},
}

type UndefinedType struct{}

func (typ *UndefinedType) AssignableTo(_ Type) bool {
	return true
}

func (typ *UndefinedType) String() string {
	return "undefined"
}

type TextType struct{}

func (typ *TextType) AssignableTo(u Type) bool {
	_, ok := u.(*TextType)
	return ok
}

func (typ *TextType) String() string {
	return "text"
}

type BoolType struct{}

func (typ *BoolType) AssignableTo(u Type) bool {
	_, ok := u.(*BoolType)
	return ok
}

func (typ *BoolType) String() string {
	return "bool"
}

type NumberType struct {
	scale int
}

func (typ *NumberType) AssignableTo(u Type) bool {
	uNum, ok := u.(*NumberType)
	return ok && typ.scale <= uNum.scale
}

func (typ *NumberType) String() string {
	return fmt.Sprintf("number[%d]", typ.scale)
}

func (t *NumberType) Scale() int {
	return t.scale
}

type SliceType struct {
	elementType Type
}

func (s *SliceType) ElementType() Type {
	return s.elementType
}

func (typ *SliceType) AssignableTo(u Type) bool {
	other, ok := u.(*SliceType)
	return ok && typ.elementType.AssignableTo(other.elementType)
}

func (typ *SliceType) String() string {
	return fmt.Sprintf("[]%s", typ.elementType)
}

func (def *Definition) AssignableTo(u Type) bool {
	// Since we do type resolution, pointer equality suffices to
	// guarantee assignability.
	return def == u
}

func (def *Definition) Name() string {
	return def.name
}

func (def *Definition) String() string {
	return def.name
}

func (def *Definition) FieldByName(name string) *Field {
	return def.fieldsByName[name]
}

func (def *Definition) Fields() []*Field {
	var fields []*Field
	for _, field := range def.fieldsByIndex {
		fields = append(fields, field)
	}
	return fields
}

type EnumType struct {
	name     string
	elements map[string]bool
}

func (typ *EnumType) AssignableTo(u Type) bool {
	return false
}

func (typ *EnumType) Name() string {
	return typ.name
}

func (typ *EnumType) String() string {
	return typ.name
}
