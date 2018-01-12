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
	defs map[string]*Definition
}

// Worksheet is ... TODO(pascal): documentation binge
type Worksheet struct {
	// def holds the definition of this worksheet.
	def *Definition

	// orig holds the worksheet data as it was when it was initially loaded.
	orig map[int]Value

	// data holds all the worksheet data.
	data map[int]Value
}

const (
	// IndexId is the reserved index to store a worksheet's identifier.
	IndexId = -2

	// IndexVersion is the reserved index to store a worksheet's version.
	IndexVersion = -1
)

type ComputedBy interface {
	Args() []string
	Compute(...Value) Value
}

type Options struct {
	// Plugins is a map of workshet names, to field names, to plugins for
	// externally computed fields.
	Plugins map[string]map[string]ComputedBy
}

func MustNewDefinitions(reader io.Reader, opts ...Options) *Definitions {
	defs, err := NewDefinitions(reader, opts...)
	if err != nil {
		panic(err)
	}
	return defs
}

// NewDefinitions parses one or more worksheet definitions, and creates worksheet
// models from them.
func NewDefinitions(reader io.Reader, opts ...Options) (*Definitions, error) {
	p := newParser(reader)
	defs, err := p.parseWorksheets()
	if err != nil {
		return nil, err
	} else if p.next() != "" || len(defs) == 0 {
		return nil, fmt.Errorf("expecting worksheet")
	}

	err = processOptions(defs, opts...)
	if err != nil {
		return nil, err
	}

	for _, def := range defs {
		var (
			indexesUsed = make(map[int]bool)
			namesUsed   = make(map[string]bool)
		)
		for _, field := range def.fields {
			// Any bad index?
			if field.index == 0 {
				return nil, fmt.Errorf("%s.%s: index cannot be zero", def.name, field.name)
			}

			// Any index reused?
			if _, ok := indexesUsed[field.index]; ok {
				return nil, fmt.Errorf("%s.%s: index %d cannot be reused", def.name, field.name, field.index)
			}
			indexesUsed[field.index] = true

			// Any names reused?
			if _, ok := namesUsed[field.name]; ok {
				return nil, fmt.Errorf("%s.%s: multiple fields named %s", def.name, field.name, field.name)
			}
			namesUsed[field.name] = true

			// Any unresolved externals?
			if _, ok := field.computedBy.(*tExternal); ok {
				return nil, fmt.Errorf("%s.%s: missing plugin for external computed_by", def.name, field.name)
			}

			// Any unknown refs types?
			if err := resolveRefTypes(fmt.Sprintf("%s.%s", def.name, field.name), defs, field); err != nil {
				return nil, err
			}
		}
	}

	// Resolve computed_by dependencies
	for _, def := range defs {
		def.dependents = make(map[int][]int)
		for _, field := range def.fields {
			if field.computedBy != nil {
				fieldName := field.name
				args := field.computedBy.Args()
				if len(args) == 0 {
					return nil, fmt.Errorf("%s.%s has no dependencies", def.name, fieldName)
				}
				for _, argName := range args {
					dependent, ok := def.fieldsByName[argName]
					if !ok {
						return nil, fmt.Errorf("%s.%s references unknown arg %s", def.name, fieldName, argName)
					}
					def.dependents[dependent.index] = append(def.dependents[dependent.index], field.index)
				}
			}
		}
	}

	return &Definitions{
		defs: defs,
	}, nil
}

func resolveRefTypes(niceFieldName string, defs map[string]*Definition, locus interface{}) error {
	switch locus.(type) {
	case *Field:
		field := locus.(*Field)
		if refTyp, ok := field.typ.(*Definition); ok {
			refDef, ok := defs[refTyp.name]
			if !ok {
				return fmt.Errorf("%s: unknown worksheet %s referenced", niceFieldName, refTyp.name)
			}
			field.typ = refDef
		}
		if _, ok := field.typ.(*SliceType); ok {
			return resolveRefTypes(niceFieldName, defs, field.typ)
		}
	case *SliceType:
		sliceType := locus.(*SliceType)
		if refTyp, ok := sliceType.elementType.(*Definition); ok {
			refDef, ok := defs[refTyp.name]
			if !ok {
				return fmt.Errorf("%s: unknown worksheet %s referenced", niceFieldName, refTyp.name)
			}
			sliceType.elementType = refDef
		}
		return resolveRefTypes(niceFieldName, defs, sliceType.elementType)
	}

	return nil
}

func processOptions(defs map[string]*Definition, opts ...Options) error {
	if len(opts) == 0 {
		return nil
	} else if len(opts) != 1 {
		return fmt.Errorf("too many options provided")
	}

	opt := opts[0]

	for name, plugins := range opt.Plugins {
		def, ok := defs[name]
		if !ok {
			return fmt.Errorf("plugins: unknown worksheet(%s)", name)
		}
		err := attachPluginsToFields(def, plugins)
		if err != nil {
			return err
		}
	}
	return nil
}

func attachPluginsToFields(def *Definition, plugins map[string]ComputedBy) error {
	for fieldName, plugin := range plugins {
		field, ok := def.fieldsByName[fieldName]
		if !ok {
			return fmt.Errorf("plugins: unknown field %s.%s", def.name, fieldName)
		}
		if _, ok := field.computedBy.(*tExternal); !ok {
			return fmt.Errorf("plugins: field %s.%s not externally defined", def.name, fieldName)
		}
		field.computedBy = &ePlugin{plugin}
	}
	return nil
}

func (defs *Definitions) MustNewWorksheet(name string) *Worksheet {
	ws, err := defs.NewWorksheet(name)
	if err != nil {
		panic(err)
	}
	return ws
}

func (defs *Definitions) NewWorksheet(name string) (*Worksheet, error) {
	ws, err := defs.newUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	// uuid
	id := uuid.NewV4()
	if err := ws.Set("id", NewText(id.String())); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	// version
	if err := ws.Set("version", MustNewValue(strconv.Itoa(1))); err != nil {
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
		orig: make(map[int]Value),
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
	return ws.data[IndexId].(*Text).value
}

func (ws *Worksheet) Version() int {
	return int(ws.data[IndexVersion].(*Number).value)
}

func (ws *Worksheet) Name() string {
	// TODO(pascal): consider having ws.Type().Name() instead
	return ws.def.name
}

func (ws *Worksheet) MustSet(name string, value Value) {
	if err := ws.Set(name, value); err != nil {
		panic(err)
	}
}

func (ws *Worksheet) Set(name string, value Value) error {
	// TODO(pascal): create a 'change', and then commit that change, garantee
	// that commits are atomic, and either win or lose the race by using
	// optimistic concurrency. Change must be a a Definition level, since it
	// could span multiple worksheets at once.

	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}

	if field.computedBy != nil {
		return fmt.Errorf("cannot assign to computed field %s", name)
	}

	if _, ok := field.typ.(*SliceType); ok {
		return fmt.Errorf("Set on slice field %s, use Append, or Del", name)
	}

	err := ws.set(field, value)
	return err
}

func (ws *Worksheet) set(field *Field, value Value) error {
	index := field.index

	// ident
	if oldValue, ok := ws.data[index]; !ok {
		if _, ok := value.(*Undefined); ok {
			return nil
		}
	} else if oldValue.Equal(value) {
		return nil
	}

	// type check
	litType := value.Type()
	if ok := litType.AssignableTo(field.typ); !ok {
		return fmt.Errorf("cannot assign value of type %s to field of type %s", litType, field.typ)
	}

	// store
	if value.Type().AssignableTo(&tUndefinedType{}) {
		delete(ws.data, index)
	} else {
		ws.data[index] = value
	}

	// if this field is an ascendant to any other, recompute them
	for _, dependentIndex := range ws.def.dependents[index] {
		dependent := ws.def.fieldsByIndex[dependentIndex]
		updatedValue, err := dependent.computedBy.Compute(ws)
		if err != nil {
			return err
		}
		if err := ws.set(dependent, updatedValue); err != nil {
			return err
		}
	}

	return nil
}

func (ws *Worksheet) MustUnset(name string) {
	if err := ws.Unset(name); err != nil {
		panic(err)
	}
}

func (ws *Worksheet) Unset(name string) error {
	if field, ok := ws.def.fieldsByName[name]; ok {
		if _, ok := field.typ.(*SliceType); ok {
			return fmt.Errorf("Unset on slice field names, must use Del")
		}
	}
	return ws.Set(name, NewUndefined())
}

func (ws *Worksheet) MustIsSet(name string) bool {
	isSet, err := ws.IsSet(name)
	if err != nil {
		panic(err)
	}
	return isSet
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

func (ws *Worksheet) MustGetSlice(name string) []Value {
	slice, err := ws.GetSlice(name)
	if err != nil {
		panic(err)
	}
	return slice
}

func (ws *Worksheet) GetSlice(name string) ([]Value, error) {
	_, slice, err := ws.getSlice(name)
	if err != nil {
		return nil, err
	} else if slice == nil {
		return nil, nil
	}

	var values []Value
	for _, element := range slice.elements {
		values = append(values, element.value)
	}
	return values, nil
}

func (ws *Worksheet) getSlice(name string) (*Field, *slice, error) {
	field, value, err := ws.get(name)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := field.typ.(*SliceType); !ok {
		return field, nil, fmt.Errorf("GetSlice on non-slice field %s, use Get", name)
	}

	if _, ok := value.(*Undefined); ok {
		return field, nil, nil
	}

	return field, value.(*slice), nil
}

// Get gets a value for base types, e.g. text, number, or bool.
// For other kinds of values, use specific getters such as `GetSlice`.
func (ws *Worksheet) Get(name string) (Value, error) {
	field, value, err := ws.get(name)
	if err != nil {
		return nil, err
	}

	if _, ok := field.typ.(*SliceType); ok {
		return nil, fmt.Errorf("Get on slice field %s, use GetSlice", name)
	}

	return value, err
}

func (ws *Worksheet) get(name string) (*Field, Value, error) {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return nil, nil, fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// is a value set for this field?
	value, ok := ws.data[index]
	if !ok {
		return field, &Undefined{}, nil
	}

	return field, value, nil
}

func (ws *Worksheet) MustAppend(name string, value Value) {
	if err := ws.Append(name, value); err != nil {
		panic(err)
	}
}

func (ws *Worksheet) Append(name string, element Value) error {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	sliceType, ok := field.typ.(*SliceType)
	if !ok {
		return fmt.Errorf("Append on non-slice field %s", name)
	}

	// is a value set for this field?
	value, ok := ws.data[index]
	if !ok {
		value = newSlice(sliceType)
		ws.data[index] = value
	}

	// append
	slice := value.(*slice)
	slice, err := slice.doAppend(element)
	if err != nil {
		return err
	}
	ws.data[index] = slice

	return nil
}

func (ws *Worksheet) MustDel(name string, index int) {
	if err := ws.Del(name, index); err != nil {
		panic(err)
	}
}

func (ws *Worksheet) Del(name string, index int) error {
	field, slice, err := ws.getSlice(name)
	if err != nil {
		if field != nil {
			if _, ok := field.typ.(*SliceType); !ok {
				return fmt.Errorf("Del on non-slice field %s", name)
			}
		}
		return err
	}

	slice, err = slice.doDel(index)
	if err != nil {
		return err
	}

	ws.data[field.index] = slice

	return nil
}

type change struct {
	before, after Value
}

func (ws *Worksheet) diff() map[int]change {
	allIndexes := make(map[int]bool)
	for index := range ws.orig {
		allIndexes[index] = true
	}
	for index := range ws.data {
		allIndexes[index] = true
	}

	diff := make(map[int]change)
	for index := range allIndexes {
		orig, hasOrig := ws.orig[index]
		data, hasData := ws.data[index]
		if hasOrig && !hasData {
			diff[index] = change{
				before: orig,
				after:  &Undefined{},
			}
		} else if !hasOrig && hasData {
			diff[index] = change{
				before: &Undefined{},
				after:  data,
			}
		} else if !orig.Equal(data) {
			diff[index] = change{
				before: orig,
				after:  data,
			}
		}
	}

	return diff
}

func diffSlices(before, after *slice) ([]int, []sliceElement) {
	var (
		b, a          int
		ranksOfDels   []int
		elementsAdded []sliceElement
	)
	for b < len(before.elements) && a < len(after.elements) {
		bElement, aElement := before.elements[b], after.elements[a]
		if bElement.rank == aElement.rank {
			if !bElement.value.Equal(aElement.value) {
				// we've replaced the value at this rank
				// represent as a delete and an add
				ranksOfDels = append(ranksOfDels, bElement.rank)
				elementsAdded = append(elementsAdded, aElement)
			}
			b++
			a++
		} else if bElement.rank < aElement.rank {
			ranksOfDels = append(ranksOfDels, bElement.rank)
			b++
		} else if aElement.rank < bElement.rank {
			elementsAdded = append(elementsAdded, aElement)
			a++
		}
	}
	for ; b < len(before.elements); b++ {
		ranksOfDels = append(ranksOfDels, before.elements[b].rank)
	}
	for ; a < len(after.elements); a++ {
		elementsAdded = append(elementsAdded, after.elements[a])
	}
	return ranksOfDels, elementsAdded
}
