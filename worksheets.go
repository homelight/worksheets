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
	"io"
	"strconv"

	"github.com/satori/go.uuid"
)

// Definitions encapsulate one or many worksheet definitions, and is the
// overall entry point into the worksheet framework.
//
// TODO(pascal) make sure Definitions are concurrent access safe!
type Definitions struct {
	// defs holds all worksheet definitions
	defs map[string]*tWorksheet
}

// Worksheet is an instance of a worksheet, which can be manipulated, as well
// as saved, and restored from a permanent storage.
type Worksheet struct {
	// dfn holds the definition of this worksheet
	def *tWorksheet

	// data holds all the worksheet data
	data map[int]Value
}

const (
	// IndexId is the reserved index to store a worksheet's identifier.
	IndexId = -1

	// IndexVersion is the reserved index to store a worksheet's version.
	IndexVersion = -2
)

// NewDefinitions parses a worksheet definition, and creates a worksheet
// model from it.
func NewDefinitions(src io.Reader) (*Definitions, error) {
	// TODO(pascal): support reading multiple worksheet definitions in one file
	p := newParser(src)
	def, err := p.parseWorksheet()
	if err != nil {
		return nil, err
	}
	return &Definitions{
		defs: map[string]*tWorksheet{
			def.name: def,
		},
	}, nil
}

func (defs *Definitions) NewWorksheet(name string) (*Worksheet, error) {
	ws, err := defs.newUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	// uuid
	id := uuid.NewV4()
	if err := ws.Set("id", fmt.Sprintf(`"%s"`, id)); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	// version
	if err := ws.Set("version", strconv.Itoa(1)); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	// validate
	if err := ws.validate(); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	return ws, nil
}

func (defs *Definitions) newUninitializedWorksheet(name string) (*Worksheet, error) {
	def, ok := defs.defs[name]
	if !ok {
		return nil, fmt.Errorf("unknown worksheet %s", name)
	}

	ws := &Worksheet{
		def:  def,
		data: make(map[int]Value),
	}

	return ws, nil
}

func (ws *Worksheet) validate() error {
	// ensure we have an id and a version
	if _, ok := ws.data[IndexId]; !ok {
		return fmt.Errorf("missing id")
	}
	if _, ok := ws.data[IndexVersion]; !ok {
		return fmt.Errorf("missing version")
	}

	// ensure all values are of the proper type
	for index, value := range ws.data {
		field, ok := ws.def.fieldsByIndex[index]
		if !ok {
			return fmt.Errorf("value present for unknown field index %d", index)
		}
		if ok := value.Type().AssignableTo(field.typ); !ok {
			return fmt.Errorf("value present with unassignable type for field index %d", index)
		}
	}

	return nil
}

func (ws *Worksheet) Id() string {
	return ws.data[IndexId].(*tText).value
}

func (ws *Worksheet) Version() int {
	return int(ws.data[IndexVersion].(*tNumber).value)
}

func (ws *Worksheet) Name() string {
	// TODO(pascal): consider having ws.Type().Name() instead
	return ws.def.name
}

func (ws *Worksheet) Set(name string, value string) error {
	// TODO(pascal): create a 'change', and then commit that change, garantee
	// that commits are atomic, and either win or lose the race by using
	// optimistic concurrency. Change must be a a Definition level, since it
	// could span multiple worksheets at once.

	// parse literal
	lit, err := NewValue(value)
	if err != nil {
		return err
	}

	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// type check
	litType := lit.Type()
	if ok := litType.AssignableTo(field.typ); !ok {
		return fmt.Errorf("cannot assign %s to %s", lit, field.typ)
	}

	// store
	ws.setAtIndex(index, lit)

	return nil
}

func (ws *Worksheet) setAtIndex(index int, value Value) {
	if value.Type().AssignableTo(&tUndefinedType{}) {
		delete(ws.data, index)
	} else {
		ws.data[index] = value
	}
}

func (ws *Worksheet) Unset(name string) error {
	return ws.Set(name, "undefined")
}

func (ws *Worksheet) IsSet(name string) (bool, error) {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return false, fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// check presence of value
	_, isSet := ws.data[index]

	return isSet, nil
}

func (ws *Worksheet) MustGet(name string) Value {
	value, err := ws.Get(name)
	if err != nil {
		panic(err)
	}
	return value
}

// TODO(pascal): need to think about proper return type here, should be consistent with Set
func (ws *Worksheet) Get(name string) (Value, error) {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return nil, fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// is a value set for this field?
	value, ok := ws.data[index]
	if !ok {
		return &tUndefined{}, nil
	}

	// type check
	if ok := value.Type().AssignableTo(field.typ); !ok {
		return nil, fmt.Errorf("cannot assign %s to %s", value, field.typ)
	}

	return value, nil
}
