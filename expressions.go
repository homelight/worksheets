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
	selectors() []tSelector
	compute(ws *Worksheet) (Value, error)
}

// Assert that all expressions implement the expression interface
var _ = []expression{
	vUndefined,
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

func (e *tExternal) selectors() []tSelector {
	panic(fmt.Sprintf("unresolved plugin in worksheet"))
}

func (e *tExternal) compute(ws *Worksheet) (Value, error) {
	panic(fmt.Sprintf("unresolved plugin in worksheet(%s)", ws.def.name))
}

func (e *Undefined) selectors() []tSelector {
	return nil
}

func (e *Undefined) compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Number) selectors() []tSelector {
	return nil
}

func (e *Number) compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Text) selectors() []tSelector {
	return nil
}

func (e *Text) compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e *Bool) selectors() []tSelector {
	return nil
}

func (e *Bool) compute(ws *Worksheet) (Value, error) {
	return e, nil
}

func (e tSelector) selectors() []tSelector {
	return []tSelector{e}
}

func (e tSelector) compute(ws *Worksheet) (Value, error) {
	// TODO(pascal): raw get for internal use?
	value, ok := ws.data[ws.def.fieldsByName[e[0]].index]
	if !ok {
		value = vUndefined
	}

	// base case
	if len(e) == 1 {
		return value, nil
	}

	// recursive case
	if _, ok := value.(*Undefined); ok {
		return value, nil
	} else if selectedWs, ok := value.(*Worksheet); ok {
		return tSelector(e[1:]).compute(selectedWs)
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
			subValue, err := tSelector(e[1:]).compute(subWs)
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

func (e *tUnop) selectors() []tSelector {
	return e.expr.selectors()
}

func (e *tUnop) compute(ws *Worksheet) (Value, error) {
	result, err := e.expr.compute(ws)
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

func (e *tBinop) selectors() []tSelector {
	left := e.left.selectors()
	right := e.right.selectors()
	return append(left, right...)
}

func (e *tBinop) compute(ws *Worksheet) (Value, error) {
	left, err := e.left.compute(ws)
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

		right, err := e.right.compute(ws)
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

	right, err := e.right.compute(ws)
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

func (e *tReturn) selectors() []tSelector {
	return e.expr.selectors()
}

func (e *tReturn) compute(ws *Worksheet) (Value, error) {
	return e.expr.compute(ws)
}

func (e *tCall) selectors() []tSelector {
	var args []tSelector
	for _, expr := range e.args {
		args = append(args, expr.selectors()...)
	}
	return args
}

// fnArgs captures the expressions passed into pre-defined functions, as well
// as the context (i.e. the worksheet) which those expressions should be
// evaluated in. It futher guarantees that expressions are computed at most
// once, ad the time where they are first accessed. This makes it possible
// to have expressions flowing through into functions thus avoiding early
// evaluation which may not be well-formed, such as `if(false, x / 0, y)`
// where evaluating early would result in a division by 0.
type fnArgs struct {
	ws     *Worksheet
	exprs  []expression
	values []Value
	errs   []error
}

func newFnArgs(ws *Worksheet, exprs []expression) *fnArgs {
	args := fnArgs{
		ws:     ws,
		exprs:  make([]expression, len(exprs)),
		values: make([]Value, len(exprs)),
		errs:   make([]error, len(exprs)),
	}
	copy(args.exprs, exprs)
	return &args
}

func (args *fnArgs) get(index int) (Value, error) {
	// compute?
	if expr := args.exprs[index]; expr != nil {
		args.values[index], args.errs[index] = expr.compute(args.ws)
		args.exprs[index] = nil
	}

	// get
	return args.values[index], args.errs[index]
}

var functions = map[string]struct {
	argsNum int
	fn      func(args *fnArgs) (Value, error)
}{
	"len": {1, func(args *fnArgs) (Value, error) {
		arg, err := args.get(0)
		if err != nil {
			return nil, err
		}
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
	"sum": {1, func(args *fnArgs) (Value, error) {
		arg, err := args.get(0)
		if err != nil {
			return nil, err
		}
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
					return vUndefined, nil
				}
			}
			return sum, nil
		default:
			return nil, fmt.Errorf("sum expects argument #1 to be slice of numbers")
		}
	}},
	"sumiftrue": {2, func(args *fnArgs) (Value, error) {
		arg0, err := args.get(0)
		if err != nil {
			return nil, err
		}
		arg1, err := args.get(1)
		if err != nil {
			return nil, err
		}

		if _, ok := arg0.(*Undefined); ok {
			return vUndefined, nil
		}
		values, ok := arg0.(*Slice)
		if !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #1 to be slice of numbers")
		} else if _, ok := values.typ.elementType.(*NumberType); !ok {
			return nil, fmt.Errorf("sumiftrue expects argument #1 to be slice of numbers")
		}

		if _, ok := arg1.(*Undefined); ok {
			return &Undefined{}, nil
		}
		conditions, ok := arg1.(*Slice)
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
	"if": {3, func(args *fnArgs) (Value, error) {
		cond, err := args.get(0)
		if err != nil {
			return nil, err
		}

		if _, ok := cond.(*Undefined); ok {
			return vUndefined, nil
		}

		if _, ok := cond.(*Bool); !ok {
			return nil, fmt.Errorf("if expects argument #1 to be bool")
		}

		if cond.(*Bool).value {
			// if-branch
			return args.get(1)
		} else {
			// else-branch
			return args.get(2)
		}
	}},
}

func (e *tCall) compute(ws *Worksheet) (Value, error) {
	fn, ok := functions[e.name[0]]
	if len(e.name) != 1 || !ok {
		return nil, fmt.Errorf("unknown function %s", e.name)
	}

	if len(e.args) != fn.argsNum {
		return nil, fmt.Errorf("%s expects %d argument(s)", e.name, fn.argsNum)
	}

	return fn.fn(newFnArgs(ws, e.args))
}

type ePlugin struct {
	computedBy ComputedBy
}

func (e *ePlugin) selectors() []tSelector {
	var args []tSelector
	for _, arg := range e.computedBy.Args() {
		args = append(args, tSelector(strings.Split(arg, ".")))
	}
	return args
}

func (e *ePlugin) compute(ws *Worksheet) (Value, error) {
	args := e.selectors()
	values := make([]Value, len(args), len(args))
	for i, arg := range args {
		value, err := arg.compute(ws)
		if err != nil {
			// TODO(pascal): panic here, this should have failed earlier when binding Args
			return nil, err
		}
		values[i] = value
	}
	return e.computedBy.Compute(values...), nil
}
