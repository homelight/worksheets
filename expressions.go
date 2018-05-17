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

func (ws *Worksheet) selectors() []tSelector {
	return nil
}

func (ws *Worksheet) compute(_ *Worksheet) (Value, error) {
	return ws, nil
}

func (slice *Slice) selectors() []tSelector {
	return nil
}

func (slice *Slice) compute(_ *Worksheet) (Value, error) {
	return slice, nil
}

func (e tSelector) selectors() []tSelector {
	return []tSelector{e}
}

func (e tSelector) compute(ws *Worksheet) (Value, error) {
	_, value, err := ws.get(e[0])
	if err != nil {
		return nil, err
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
		var elementType = subWsDef.fieldsByName[e[1]].Type()
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
// once, at the time when they are first accessed. This makes it possible
// to have expressions flowing through into functions thus avoiding early
// evaluation which may not be well-formed, such as `if(false, x / 0, y)`
// where evaluating early would result in a division by 0.
type fnArgs struct {
	ws     *Worksheet
	round  *tRound
	exprs  []expression
	values []Value
	errs   []error
}

func newFnArgs(ws *Worksheet, round *tRound, values []Value) *fnArgs {
	return &fnArgs{
		ws:     ws,
		round:  round,
		exprs:  make([]expression, len(values)),
		values: values,
		errs:   make([]error, len(values)),
	}
}

func newLazyFnArgs(ws *Worksheet, round *tRound, exprs []expression) *fnArgs {
	args := fnArgs{
		ws:     ws,
		round:  round,
		exprs:  make([]expression, len(exprs)),
		values: make([]Value, len(exprs)),
		errs:   make([]error, len(exprs)),
	}
	copy(args.exprs, exprs)
	return &args
}

func (args *fnArgs) checkArgsNum(nums ...int) error {
	actual := args.num()
	if len(nums) == 1 {
		// exact
		num := nums[0]
		if actual != num {
			return fmt.Errorf("%d argument(s) expected but %d found", num, actual)
		}
	} else {
		// min - max
		min, max := nums[0], nums[1]
		if err := args.checkMinArgsNum(min); err != nil {
			return err
		}
		if max < actual {
			return fmt.Errorf("at most %d argument(s) expected but %d found", max, actual)
		}
	}
	return nil
}

func (args *fnArgs) checkMinArgsNum(min int) error {
	actual := args.num()
	if actual < min {
		if actual == 0 {
			return fmt.Errorf("at least %d argument(s) expected but none found", min)
		} else {
			return fmt.Errorf("at least %d argument(s) expected but only %d found", min, actual)
		}
	}
	return nil
}

func (args *fnArgs) num() int {
	return len(args.exprs)
}

func (args *fnArgs) has(index int) bool {
	return index < args.num()
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

var functions = map[string]func(args *fnArgs) (Value, error){
	"len": func(args *fnArgs) (Value, error) {
		if err := args.checkArgsNum(1); err != nil {
			return nil, err
		}
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
			return nil, fmt.Errorf("argument #1 expected to be text, or slice")
		}
	},
	"sum": rSum,
	"sumiftrue": func(args *fnArgs) (Value, error) {
		if err := args.checkArgsNum(2); err != nil {
			return nil, err
		}
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
			return nil, fmt.Errorf("argument #1 expected to be slice of numbers")
		} else if _, ok := values.typ.elementType.(*NumberType); !ok {
			return nil, fmt.Errorf("argument #1 expected to be slice of numbers")
		}

		if _, ok := arg1.(*Undefined); ok {
			return vUndefined, nil
		}
		conditions, ok := arg1.(*Slice)
		if !ok {
			return nil, fmt.Errorf("argument #2 expected to be slice of bools")
		} else if _, ok := conditions.typ.elementType.(*BoolType); !ok {
			return nil, fmt.Errorf("argument #2 expected to be slice of bools")
		}

		if len(values.Elements()) != len(conditions.Elements()) {
			return nil, fmt.Errorf("argument #1 and argument #2 expected to be the same length")
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
					return vUndefined, nil
				}
			} else {
				return vUndefined, nil
			}
		}
		return sum, nil
	},
	"if": func(args *fnArgs) (Value, error) {
		if err := args.checkArgsNum(2, 3); err != nil {
			return nil, err
		}
		cond, err := args.get(0)
		if err != nil {
			return nil, err
		}

		if _, ok := cond.(*Undefined); ok {
			return vUndefined, nil
		}

		if _, ok := cond.(*Bool); !ok {
			return nil, fmt.Errorf("argument #1 expected to be bool")
		}

		if cond.(*Bool).value {
			// if-branch
			return args.get(1)
		} else {
			// else-branch
			if args.has(2) {
				return args.get(2)
			} else {
				return vUndefined, nil
			}
		}
	},
	"first_of": rFirstOf,
	"min":      rMin,
	"max":      rMax,
	"slice":    rSlice,
	"avg":      rAvg,
}

func rFirstOf(args *fnArgs) (Value, error) {
	if err := args.checkMinArgsNum(1); err != nil {
		return nil, err
	}
	for i := 0; i < args.num(); i++ {
		arg, err := args.get(i)
		if err != nil {
			return nil, err
		}
		switch a := arg.(type) {
		case *Slice:
			return rFirstOf(newFnArgs(args.ws, args.round, a.Elements()))
		case *Undefined:
			continue
		default:
			return a, nil
		}
	}
	return vUndefined, nil
}

type foldNumbers interface {
	update(value *Number)
	result() Value
}

func rFoldNumbers(f foldNumbers, args *fnArgs, minArgs int) (Value, error) {
	if err := args.checkMinArgsNum(minArgs); err != nil {
		return nil, err
	}
	for i := 0; i < args.num(); i++ {
		arg, err := args.get(i)
		if err != nil {
			return nil, err
		}
		switch value := arg.(type) {
		case *Undefined:
			return vUndefined, nil
		case *Number:
			f.update(value)
		case *Slice:
			if value.Len() != 0 {
				result, err := rFoldNumbers(f, newFnArgs(args.ws, args.round, value.Elements()), 0)
				if err != nil {
					return nil, err
				}
				if result == vUndefined {
					return vUndefined, nil
				}
			}
		default:
			return nil, fmt.Errorf("encountered non-numerical argument")
		}
	}
	return f.result(), nil
}

type sumFolder struct {
	sum *Number
}

func (f *sumFolder) update(value *Number) {
	f.sum = f.sum.Plus(value)
}

func (f *sumFolder) result() Value {
	return f.sum
}

func rSum(args *fnArgs) (Value, error) {
	return rFoldNumbers(&sumFolder{sum: vZero}, args, 1)
}

type minFolder struct {
	min *Number
}

func (f *minFolder) update(value *Number) {
	if f.min == nil || value.LessThan(f.min) {
		f.min = value
	}
}

func (f *minFolder) result() Value {
	return f.min
}

func rMin(args *fnArgs) (Value, error) {
	return rFoldNumbers(&minFolder{}, args, 1)
}

type maxFolder struct {
	max *Number
}

func (f *maxFolder) update(value *Number) {
	if f.max == nil || value.GreaterThan(f.max) {
		f.max = value
	}
}

func (f *maxFolder) result() Value {
	return f.max
}

func rMax(args *fnArgs) (Value, error) {
	return rFoldNumbers(&maxFolder{}, args, 1)
}

func rSlice(args *fnArgs) (Value, error) {
	if err := args.checkMinArgsNum(1); err != nil {
		return nil, err
	}
	var (
		values      []Value
		elementType Type
	)
	for i := 0; i < args.num(); i++ {
		arg, err := args.get(i)
		if err != nil {
			return nil, err
		}
		if _, ok := arg.Type().(*UndefinedType); !ok {
			if elementType == nil {
				elementType = arg.Type()
			} else if nElementType, ok := elementType.(*NumberType); ok {
				if nArgType, ok := arg.Type().(*NumberType); ok {
					if nElementType.scale < nArgType.scale {
						elementType = nArgType
					}
				} else {
					return nil, fmt.Errorf("cannot mix incompatible types %s and %s in slice", elementType, arg.Type())
				}
			} else if !arg.assignableTo(elementType) {
				return nil, fmt.Errorf("cannot mix incompatible types %s and %s in slice", elementType, arg.Type())
			}
		}
		values = append(values, arg)
	}
	if elementType == nil {
		return nil, fmt.Errorf("unable to infer slice type, only undefined values encountered")
	}
	return newSlice(&SliceType{elementType}, values...), nil
}

type avgFolder struct {
	sum   *Number
	count int
	round *tRound
}

func (f *avgFolder) update(value *Number) {
	f.sum = f.sum.Plus(value)
	f.count++
}

func (f *avgFolder) result() Value {
	return f.sum.Div(NewNumberFromInt(f.count), f.round.mode, f.round.scale)
}

func rAvg(args *fnArgs) (Value, error) {
	if args.round == nil {
		return nil, fmt.Errorf("missing rounding mode")
	}
	return rFoldNumbers(&avgFolder{
		sum:   vZero,
		round: args.round,
	}, args, 1)
}

func (e *tCall) compute(ws *Worksheet) (Value, error) {
	fn, ok := functions[e.name[0]]
	if len(e.name) != 1 || !ok {
		return nil, fmt.Errorf("unknown function %s", e.name)
	}

	value, err := fn(newLazyFnArgs(ws, e.round, e.args))
	if err != nil {
		return nil, fmt.Errorf("%s: %s", e.name, err)
	}

	if e.round != nil {
		nValue, ok := value.(*Number)
		if !ok {
			return nil, fmt.Errorf("unable to round non-numerical value")
		}
		value = nValue.Round(e.round.mode, e.round.scale)
	}

	return value, nil
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
