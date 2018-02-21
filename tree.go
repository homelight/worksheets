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

type Definition struct {
	name          string
	fieldsByName  map[string]*Field
	fieldsByIndex map[int]*Field
}

func (def *Definition) addField(field *Field) error {
	field.def = def

	// Parsing guarantees user-defined fields are non-negative.
	if field.index == 0 {
		return fmt.Errorf("%s.%s: index cannot be zero", def.name, field.name)
	}

	if _, ok := def.fieldsByIndex[field.index]; ok {
		return fmt.Errorf("%s.%s: index %d cannot be reused", def.name, field.name, field.index)
	}
	def.fieldsByIndex[field.index] = field

	if _, ok := def.fieldsByName[field.name]; ok {
		return fmt.Errorf("%s.%s: name %s cannot be reused", def.name, field.name, field.name)
	}
	def.fieldsByName[field.name] = field

	return nil
}

type Field struct {
	index         int
	name          string
	typ           Type
	def           *Definition
	dependents    []*Field
	computedBy    expression
	constrainedBy expression
}

func (f *Field) Type() Type {
	return f.typ
}

func (f *Field) Name() string {
	return f.name
}

func (f *Field) String() string {
	return fmt.Sprintf("field(%s.%s, %s)", f.def.name, f.name, f.typ)
}

type tOp string

const (
	opPlus               tOp = "plus"
	opMinus                  = "minus"
	opMult                   = "mult"
	opDiv                    = "div"
	opNot                    = "not"
	opEqual                  = "equal"
	opNotEqual               = "not-equal"
	opGreaterThan            = "greater-than"
	opGreaterThanOrEqual     = "greater-than-or-equal"
	opLessThan               = "less-than"
	opLessThanOrEqual        = "less-than-or-equal"
	opOr                     = "or"
	opAnd                    = "and"
)

type tRound struct {
	mode  RoundingMode
	scale int
}

func (t *tRound) String() string {
	return fmt.Sprintf("%s %d", t.mode, t.scale)
}

type tExternal struct{}

type tUnop struct {
	op   tOp
	expr expression
}

type tBinop struct {
	op          tOp
	left, right expression
	round       *tRound
}

func (t *tBinop) String() string {
	return fmt.Sprintf("binop(%s, %s, %s, %s)", t.op, t.left, t.right, t.round)
}

// tSelector represents a selector such as referencing a field `foo`, or
// referencing a field through a path such `foo.bar`.
type tSelector []string

type tReturn struct {
	expr expression
}
