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
	"strings"
)

type expression interface {
	Args() []string
	Compute(ws *Worksheet) (Value, error)
}

// Assert that all expressions implement the expression interface
var _ = []expression{
	&Undefined{},
	&Number{},
	&Text{},
	&Bool{},

	&tExternal{},
	&ePlugin{},
	tSelector(nil),
	&tUnop{},
	&tBinop{},
	&tReturn{},
	&tCall{},
}

func (e *tExternal) Args() []string {
	panic(fmt.Sprintf("unresolved plugin in worksheet"))
}

func (e *tExternal) Compute(ws *Worksheet) (Value, error) {
	panic(fmt.Sprintf("unresolved plugin in worksheet(%s)", ws.def.name))
}

func (e *Undefined) Args() []string {
	return nil
}

func (e *Undefined) Compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Number) Args() []string {
	return nil
}

func (e *Number) Compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Text) Args() []string {
	return nil
}

func (e *Text) Compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Bool) Args() []string {
	return nil
}

func (e *Bool) Compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e tSelector) Args() []string {
	return []string{strings.Join([]string(e), ".")}
}

func (e tSelector) Compute(ws *Worksheet) (Value, error) {
	// TODO(pascal): raw get for internal use?
	value, ok := ws.data[ws.def.fieldsByName[e[0]].index]
	if !ok {
		value = &Undefined{}
	}

	// base case
	if len(e) == 1 {
		return value, nil
	}

	// recursive case
	if _, ok := value.(*Undefined); ok {
		return value, nil
	} else if selectedWs, ok := value.(*Worksheet); ok {
		return tSelector(e[1:]).Compute(selectedWs)
	} else if selectedSlice, ok := value.(*Slice); ok {
		subWsDef, ok := ws.def.fieldsByName[e[0]].Type().(*SliceType).ElementType().(*Definition)
		if !ok {
			return nil, fmt.Errorf("sorry! more complex selectors are not supported yet!")
		}
		var elementType Type = subWsDef.fieldsByName[e[1]].Type()
		var elements []sliceElement
		for _, elem := range selectedSlice.elements {
			subWs, ok := elem.value.(*Worksheet)
			if !ok {
				return nil, fmt.Errorf("sorry! more complex selectors are not supported yet!")
			}
			subValue, err := tSelector(e[1:]).Compute(subWs)
			if err != nil {
				return nil, err
			}
			elements = append(elements, sliceElement{
				value: subValue,
			})
		}
		return &Slice{
			elements: elements,
			typ:      &SliceType{elementType},
		}, nil
	}

	return nil, fmt.Errorf("sorry! more complex selectors are not supported yet!")
}

func (e *tUnop) Args() []string {
	return e.expr.Args()
}

func (e *tUnop) Compute(ws *Worksheet) (Value, error) {
	result, err := e.expr.Compute(ws)
	if err != nil {
		return nil, err
	}

	if _, ok := result.(*Undefined); ok {
		return result, nil
	}

	switch e.op {
	case opNot:
		bResult, ok := result.(*Bool)
		if !ok {
			return nil, fmt.Errorf("! on non-bool")
		}
		return &Bool{!bResult.value}, nil
	default:
		panic(fmt.Sprintf("not implemented for %s", e.op))
	}
}

func (e *tBinop) Args() []string {
	left := e.left.Args()
	right := e.right.Args()
	return append(left, right...)
}

func (e *tBinop) Compute(ws *Worksheet) (Value, error) {
	left, err := e.left.Compute(ws)
	if err != nil {
		return nil, err
	}

	// bool operations
	if e.op == opAnd || e.op == opOr {
		if _, ok := left.(*Undefined); ok {
			return left, nil
		}

		bLeft, ok := left.(*Bool)
		if !ok {
			return nil, fmt.Errorf("op on non-bool")
		}

		if (e.op == opAnd && !bLeft.value) || (e.op == opOr && bLeft.value) {
			return bLeft, nil
		}

		right, err := e.right.Compute(ws)
		if err != nil {
			return nil, err
		}

		if _, ok := right.(*Undefined); ok {
			return right, nil
		}

		bRight, ok := right.(*Bool)
		if !ok {
			return nil, fmt.Errorf("op on non-bool")
		}

		return bRight, nil
	}

	right, err := e.right.Compute(ws)
	if err != nil {
		return nil, err
	}

	// equality
	if e.op == opEqual {
		return &Bool{left.Equal(right)}, nil
	}
	if e.op == opNotEqual {
		return &Bool{!left.Equal(right)}, nil
	}

	// numerical operations
	if _, ok := left.(*Undefined); ok {
		return left, nil
	}

	nLeft, ok := left.(*Number)
	if !ok {
		return nil, fmt.Errorf("op on non-number")
	}

	if _, ok := right.(*Undefined); ok {
		return right, nil
	}

	nRight, ok := right.(*Number)
	if !ok {
		return nil, fmt.Errorf("op on non-number")
	}

	switch e.op {
	case opGreaterThan:
		return &Bool{nLeft.GreaterThan(nRight)}, nil
	case opGreaterThanOrEqual:
		return &Bool{nLeft.GreaterThanOrEqual(nRight)}, nil
	case opLessThan:
		return &Bool{nLeft.LessThan(nRight)}, nil
	case opLessThanOrEqual:
		return &Bool{nLeft.LessThanOrEqual(nRight)}, nil
	}

	var result *Number
	switch e.op {
	case opPlus:
		result = nLeft.Plus(nRight)
	case opMinus:
		result = nLeft.Minus(nRight)
	case opMult:
		result = nLeft.Mult(nRight)
	case opDiv:
		if e.round == nil {
			return nil, fmt.Errorf("division without rounding mode")
		}
		return nLeft.Div(nRight, e.round.mode, e.round.scale), nil
	default:
		panic(fmt.Sprintf("not implemented for %s", e.op))
	}

	if e.round != nil {
		result = result.Round(e.round.mode, e.round.scale)
	}

	return result, nil
}

func (e *tReturn) Args() []string {
	return e.expr.Args()
}

func (e *tReturn) Compute(ws *Worksheet) (Value, error) {
	return e.expr.Compute(ws)
}

func (e *tCall) Args() []string {
	var args []string
	for _, expr := range e.args {
		args = append(args, expr.Args()...)
	}
	return args
}

var functions = map[string]struct {
	argsNum int
	fn      func([]Value) (Value, error)
}{
	"len": {1, func(args []Value) (Value, error) {
		arg := args[0]
		switch v := arg.(type) {
		case *Undefined:
			return v, nil
		case *Text:
			return NewNumberFromInt(len(v.value)), nil
		case *Slice:
			return NewNumberFromInt(len(v.elements)), nil
		default:
			return nil, fmt.Errorf("len expects argument #1 to be text, or slice")
		}
	}},
	"sum": {1, func(args []Value) (Value, error) {
		arg := args[0]
		switch v := arg.(type) {
		case *Slice:
			numType, ok := v.typ.elementType.(*NumberType)
			if !ok {
				return nil, fmt.Errorf("sum expects argument #1 to be slice of numbers")
			}
			sum := &Number{0, numType}
			for _, elem := range v.elements {
				if num, ok := elem.value.(*Number); ok {
					sum = sum.Plus(num)
				} else {
					return &Undefined{}, nil
				}
			}
			return sum, nil
		default:
			return nil, fmt.Errorf("sum expects argument #1 to be slice of numbers")
		}
	}},
	"sumiftrue": {2, func(args []Value) (Value, error) {
		values, ok := args[0].(*Slice)
		if !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #1 to be slice of numbers")
		} else if _, ok := values.typ.elementType.(*NumberType); !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #1 to be slice of numbers")
		}

		conditions, ok := args[1].(*Slice)
		if !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #2 to be slice of bools")
		} else if _, ok := conditions.typ.elementType.(*BoolType); !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #2 to be slice of bools")
		}

		if len(values.Elements()) != len(conditions.Elements()) {
			return nil, fmt.Errorf("sumiftrue expects argument #1 and argument #2 to be the same length")
		}

		numType, _ := values.typ.elementType.(*NumberType)
		sum := &Number{0, numType}
		for i := 0; i < len(values.Elements()); i++ {
			if num, ok := values.elements[i].value.(*Number); ok {
				if val, ok := conditions.elements[i].value.(*Bool); ok {
					if val.Value() {
						sum = sum.Plus(num)
					}
				} else {
					return &Undefined{}, nil
				}
			} else {
				return &Undefined{}, nil
			}
		}
		return sum, nil
	}},
}

func (e *tCall) Compute(ws *Worksheet) (Value, error) {
	fn, ok := functions[e.name[0]]
	if len(e.name) != 1 || !ok {
		return nil, fmt.Errorf("unknown function %s", e.name)
	}

	if len(e.args) != fn.argsNum {
		return nil, fmt.Errorf("%s expects %d argument(s)", e.name, fn.argsNum)
	}

	var args []Value
	for _, expr := range e.args {
		arg, err := expr.Compute(ws)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return fn.fn(args)
}

type ePlugin struct {
	computedBy ComputedBy
}

func (e *ePlugin) Args() []string {
	return e.computedBy.Args()
}

func (e *ePlugin) Compute(ws *Worksheet) (Value, error) {
	args := e.computedBy.Args()
	values := make([]Value, len(args), len(args))
	for i, arg := range args {
		selector := argToSelector(arg)
		value, err := selector.Compute(ws)
		if err != nil {
			// TODO(pascal): panic here, this should have failed earlier when binding Args
			return nil, err
		}
		values[i] = value
	}
	return e.computedBy.Compute(values...), nil
}
