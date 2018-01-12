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
}

// Assert that all type literals are Type.
var _ []Type = []Type{
	&tUndefinedType{},
	&tTextType{},
	&tBoolType{},
	&tNumberType{},
	&SliceType{},
	&Definition{},
}

func (typ *tUndefinedType) AssignableTo(_ Type) bool {
	return true
}

func (typ *tUndefinedType) String() string {
	return "undefined"
}

func (typ *tTextType) AssignableTo(u Type) bool {
	_, ok := u.(*tTextType)
	return ok
}

func (typ *tTextType) String() string {
	return "text"
}

func (typ *tBoolType) AssignableTo(u Type) bool {
	_, ok := u.(*tBoolType)
	return ok
}

func (typ *tBoolType) String() string {
	return "bool"
}

func (typ *tNumberType) AssignableTo(u Type) bool {
	uNum, ok := u.(*tNumberType)
	return ok && typ.scale <= uNum.scale
}

func (typ *tNumberType) String() string {
	return fmt.Sprintf("number[%d]", typ.scale)
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

func (def *Definition) String() string {
	return def.name
}

func (def *Definition) FieldByName(name string) *Field {
	return def.fieldsByName[name]
}

func (def *Definition) FieldNames() []string {
	fieldNames := []string{}
	for fieldName, _ := range def.fieldsByName {
		fieldNames = append(fieldNames, fieldName)
	}
	return fieldNames
}
