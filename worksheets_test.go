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
	"sort"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestExample() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	isSet, err := ws.IsSet("name")
	require.NoError(s.T(), err)
	require.False(s.T(), isSet)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.True(s.T(), isSet)

	name, err := ws.Get("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), `"Alice"`, name.String())

	err = ws.Unset("name")
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.False(s.T(), isSet)
}

func (s *Zuite) TestNewDefinitionsGood() {
	cases := []string{
		`worksheet simple {
			65535:index_at_max bool
		}`,
	}
	for _, ex := range cases {
		_, err := NewDefinitions(strings.NewReader(ex))
		assert.NoError(s.T(), err, ex)
	}
}

func (s *Zuite) TestNewDefinitionsErrors() {
	cases := map[string]string{
		// crap input
		``:                `expecting worksheet`,
		` `:               `expecting worksheet`,
		`some text`:       `expecting worksheet`,
		`not a worksheet`: `expecting worksheet`,
		`work sheet`:      `expecting worksheet`,

		// worksheet semantics
		`worksheet simple {
			65536:index_too_large bool
		}`: `simple.index_too_large: index cannot be greater than 65535`,

		`worksheet simple {
			9999999999999999999999999999999999999999999999999:index_too_large bool
		}`: `simple.index_too_large: index cannot be greater than 65535`,

		`worksheet simple {
			0:no_can_do_with_zero bool
		}`: `simple.no_can_do_with_zero: index cannot be zero`,

		`worksheet simple {
			42:full_name text
			42:happy bool
		}`: `simple.happy: index 42 cannot be reused`,

		`worksheet simple {
			42:same_name text
			43:same_name text
		}`: `simple.same_name: name same_name cannot be reused`,

		`worksheet ref_to_worksheet {
			89:ref_here some_other_worksheet
		}`: `ref_to_worksheet.ref_here: unknown worksheet some_other_worksheet referenced`,

		`worksheet refs_to_worksheet {
			89:refs_here []some_other_worksheet
		}`: `refs_to_worksheet.refs_here: unknown worksheet some_other_worksheet referenced`,

		`worksheet refs_to_worksheet {
			89:refs_here [][]some_other_worksheet
		}`: `refs_to_worksheet.refs_here: unknown worksheet some_other_worksheet referenced`,

		`worksheet constrained_and_computed {
			1:age number[0]
			69:some_field text constrained_by { return true } computed_by { return age + 2 }
		}`: `expected index, found computed_by`,

		`worksheet computed_and_constrained {
			1:age number[0]
			69:some_field text computed_by { return age + 2 } constrained_by { return true }
		}`: `expected index, found constrained_by`,

		`worksheet constrained_invalid_arg {
			69:some_field text constrained_by { return not_a_field == "Alex" }
		}`: `constrained_invalid_arg.some_field references unknown arg not_a_field`,

		`worksheet constrained_no_arg {
			69:some_field text constrained_by { return true }
		}`: `constrained_no_arg.some_field has no dependencies`,
	}
	for input, msg := range cases {
		_, err := NewDefinitions(strings.NewReader(input))
		assert.EqualError(s.T(), err, msg, input+" expecting "+msg)
	}
}

func (s *Zuite) TestWorksheetNew_multipleDefs() {
	wsDefs := `worksheet one {1:name text} worksheet two {1:occupation text}`
	defs, err := NewDefinitions(strings.NewReader(wsDefs))
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(defs.defs))

	for _, wsName := range []string{"one", "two"} {
		_, ok := defs.defs[wsName]
		require.True(s.T(), ok)
	}
}

func (s *Zuite) TestWorksheetNew_multipleDefsSameName() {
	wsDefs := `worksheet simple {1:name text} worksheet simple {1:occupation text}`
	_, err := NewDefinitions(strings.NewReader(wsDefs))
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "multiple worksheets with name simple", err.Error())
	}
}

func (s *Zuite) TestWorksheetNew_origEmpty() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// We need to ensure orig is empty since this is a fresh worksheet, and
	// even the special values (e.g. version, id) must be taken into
	// consideration upon save.
	require.Empty(s.T(), ws.orig)
}

func (s *Zuite) TestWorksheetNew_refTypesMustBeResolved() {
	defs := MustNewDefinitions(strings.NewReader(`
		worksheet simple {
			1:me     simple
			2:myself simple
			3:and_i  simple
			4:not_me even_simpler
		}

		worksheet even_simpler {
			5:not_it simple
		}

		worksheet refs_in_slices {
			6:many_simples  []simple
			7:many_simplers [][]even_simpler
		}`))
	var (
		simpleDef       = defs.defs["simple"]
		evenSimplerDef  = defs.defs["even_simpler"]
		refsInSlicesDef = defs.defs["refs_in_slices"]
	)

	// refs
	cases := []struct {
		parent *Definition
		field  string
		child  *Definition
	}{
		{simpleDef, "me", simpleDef},
		{simpleDef, "myself", simpleDef},
		{simpleDef, "and_i", simpleDef},
		{simpleDef, "not_me", evenSimplerDef},

		{evenSimplerDef, "not_it", simpleDef},
	}
	for _, ex := range cases {
		assert.True(s.T(), ex.parent.fieldsByName[ex.field].typ == ex.child,
			"type of field %s.%s should resolve to def of %s",
			ex.parent, ex.field, ex.child)
	}

	// slices
	manySimplesTyp := refsInSlicesDef.fieldsByName["many_simples"].typ.(*SliceType)
	assert.True(s.T(), manySimplesTyp.elementType == simpleDef)

	manySimplersTyp := refsInSlicesDef.fieldsByName["many_simplers"].typ.(*SliceType)
	manySimplersElemTyp := manySimplersTyp.elementType.(*SliceType)
	assert.True(s.T(), manySimplersElemTyp.elementType == evenSimplerDef)
}

func (s *Zuite) TestWorksheetGet_undefinedIfNoValue() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	value := ws.MustGet("name")
	require.Equal(s.T(), "undefined", value.String())
}

func (s *Zuite) TestWorksheet_idAndVersion() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	id, err := ws.Get("id")
	require.NoError(s.T(), err)
	require.Equal(s.T(), 36+2, len(id.String()))

	version, err := ws.Get("version")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "1", version.String())
}

func (s *Zuite) TestWorksheet_diff() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// initial diff
	require.Equal(s.T(), map[int]change{
		indexId: change{
			before: &Undefined{},
			after:  NewText(ws.Id()),
		},
		indexVersion: change{
			before: &Undefined{},
			after:  MustNewValue("1"),
		},
	}, ws.diff())

	// set name to Alice
	err = ws.Set("name", alice)
	require.NoError(s.T(), err)

	// now, also expecting Alice
	require.Equal(s.T(), map[int]change{
		indexId: change{
			before: &Undefined{},
			after:  NewText(ws.Id()),
		},
		indexVersion: change{
			before: &Undefined{},
			after:  MustNewValue("1"),
		},
		1: change{
			before: &Undefined{},
			after:  alice,
		},
	}, ws.diff())

	// Alice is now Bob
	err = ws.Set("name", bob)
	require.NoError(s.T(), err)

	require.Equal(s.T(), map[int]change{
		indexId: change{
			before: &Undefined{},
			after:  NewText(ws.Id()),
		},
		indexVersion: change{
			before: &Undefined{},
			after:  MustNewValue("1"),
		},
		1: change{
			before: &Undefined{},
			after:  bob,
		},
	}, ws.diff())

	// let's fake Bob being there before, and not anymore
	ws.orig[1] = ws.data[1]
	err = ws.Unset("name")
	require.NoError(s.T(), err)

	// now, name should go to an explicit undefine
	require.Equal(s.T(), map[int]change{
		indexId: change{
			before: &Undefined{},
			after:  NewText(ws.Id()),
		},
		indexVersion: change{
			before: &Undefined{},
			after:  MustNewValue("1"),
		},
		1: change{
			before: bob,
			after:  &Undefined{},
		},
	}, ws.diff())
}

func (s *Zuite) TestWorksheet_diffSlices() {
	cases := []struct {
		before, after   map[int]Value
		elementsDeleted []sliceElement
		elementsAdded   []sliceElement
	}{
		{
			before:          map[int]Value{},
			after:           map[int]Value{17: alice},
			elementsDeleted: nil,
			elementsAdded:   []sliceElement{{rank: 17, value: alice}},
		},
		{
			before:          map[int]Value{17: alice},
			after:           map[int]Value{17: bob},
			elementsDeleted: []sliceElement{{rank: 17, value: alice}},
			elementsAdded:   []sliceElement{{rank: 17, value: bob}},
		},
		{
			before:          map[int]Value{17: alice},
			after:           map[int]Value{},
			elementsDeleted: []sliceElement{{rank: 17, value: alice}},
			elementsAdded:   nil,
		},
		{
			before:          map[int]Value{17: alice, 67: bob},
			after:           map[int]Value{2: carol, 67: bob},
			elementsDeleted: []sliceElement{{rank: 17, value: alice}},
			elementsAdded:   []sliceElement{{2, carol}},
		},
		{
			before:          map[int]Value{1: alice, 3: bob, 5: carol},
			after:           map[int]Value{2: carol},
			elementsDeleted: []sliceElement{{rank: 1, value: alice}, {rank: 3, value: bob}, {rank: 5, value: carol}},
			elementsAdded:   []sliceElement{{2, carol}},
		},
	}
	for _, ex := range cases {
		sliceChange := diffSlices(toSlice(ex.before), toSlice(ex.after))
		assert.Equal(s.T(), ex.elementsDeleted, sliceChange.deleted, "dels: %v to %v", ex.before, ex.after)
		assert.Equal(s.T(), ex.elementsAdded, sliceChange.added, "adds: %v to %v", ex.before, ex.after)
	}
}

func toSlice(data map[int]Value) *Slice {
	ranks := make([]int, 0, len(data))
	for rank := range data {
		ranks = append(ranks, rank)
	}
	sort.Ints(ranks)

	slice := &Slice{}
	for _, rank := range ranks {
		slice.elements = append(slice.elements, sliceElement{
			rank:  rank,
			value: data[rank],
		})
	}
	return slice
}
