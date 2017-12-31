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
	&tVar{},
	&tBinop{},
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

func (e *tVar) Args() []string {
	return []string{e.name}
}

func (e *tVar) Compute(ws *Worksheet) (Value, error) {
	return ws.Get(e.name)
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
	if _, ok := left.(*Undefined); ok {
		return left, nil
	}

	right, err := e.right.Compute(ws)
	if err != nil {
		return nil, err
	}
	if _, ok := right.(*Undefined); ok {
		return right, nil
	}

	nLeft, ok := left.(*Number)
	if !ok {
		return nil, fmt.Errorf("op on non-number")
	}

	nRight, ok := right.(*Number)
	if !ok {
		return nil, fmt.Errorf("op on non-number")
	}

	// TODO(pascal): implement for other ops
	switch e.op {
	case opPlus:
		return nLeft.Plus(nRight), nil
	default:
		panic("not implemented")
	}
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
		value := ws.MustGet(arg)
		values[i] = value
	}
	return e.computedBy.Compute(values...), nil
}
