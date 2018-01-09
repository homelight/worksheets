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

type tWorksheet struct {
	name          string
	fields        []*tField
	fieldsByName  map[string]*tField
	fieldsByIndex map[int]*tField

	// derived values handling
	externals  map[int]ComputedBy
	dependents map[int][]int
}

func (ws *tWorksheet) addField(field *tField) {
	ws.fields = append(ws.fields, field)

	// Clobbering due to name reuse, or index reuse, is checked by validating
	// the tree.
	ws.fieldsByName[field.name] = field
	ws.fieldsByIndex[field.index] = field
}

type tField struct {
	index      int
	name       string
	typ        Type
	computedBy expression
	// also need constrainedBy *tExpression
}

type tUndefinedType struct{}

type tTextType struct{}

type tBoolType struct{}

type tNumberType struct {
	scale int
}

type tSliceType struct {
	elementType Type
}

type tWorksheetType struct {
	name string
}

type tOp string

const (
	opPlus     tOp = "plus"
	opMinus        = "minus"
	opMult         = "mult"
	opDiv          = "div"
	opNot          = "not"
	opEqual        = "equal"
	opNotEqual     = "not-equal"
	opOr           = "or"
	opAnd          = "and"
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

type tVar struct {
	name string
}
