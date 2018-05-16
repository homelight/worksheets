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
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {1:name text}`))
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

func (s *Zuite) TestNewDefinitionsErrors() {
	cases := map[string]string{
		// crap input
		`some text`:       `syntax error: non-type declaration`,
		`not a worksheet`: `syntax error: non-type declaration`,
		`work sheet`:      `syntax error: non-type declaration`,
		`type {`:          `expected name, found {`,
		`type simple {`:   `expected worksheet, or enum`,

		// worksheet semantics
		`type simple worksheet {
			65536:index_too_large bool
		}`: `simple.index_too_large: index cannot be greater than 65535`,

		`type simple worksheet {
			9999999999999999999999999999999999999999999999999:index_too_large bool
		}`: `simple.index_too_large: index cannot be greater than 65535`,

		`type simple worksheet {
			0:no_can_do_with_zero bool
		}`: `simple.no_can_do_with_zero: index cannot be zero`,

		`type simple worksheet {
			42:full_name text
			42:happy bool
		}`: `simple.happy: index 42 cannot be reused`,

		`type simple worksheet {
			42:same_name text
			43:same_name text
		}`: `simple.same_name: name same_name cannot be reused`,

		`type ref_to_worksheet worksheet {
			89:ref_here some_other_worksheet
		}`: `ref_to_worksheet.ref_here: unknown type some_other_worksheet`,

		`type refs_to_worksheet worksheet {
			89:refs_here []some_other_worksheet
		}`: `refs_to_worksheet.refs_here: unknown type some_other_worksheet`,

		`type refs_to_worksheet worksheet {
			89:refs_here [][]some_other_worksheet
		}`: `refs_to_worksheet.refs_here: unknown type some_other_worksheet`,

		`type refs_to_enum worksheet {
			89:refs_here some_enum
		}`: `refs_to_enum.refs_here: unknown type some_enum`,

		`type refs_to_enum worksheet {
			89:refs_here []some_enum
		}`: `refs_to_enum.refs_here: unknown type some_enum`,

		`type constrained_and_computed worksheet {
			1:age number[0]
			69:some_field text constrained_by { return true } computed_by { return age + 2 }
		}`: `expected index, found computed_by`,

		`type computed_and_constrained worksheet {
			1:age number[0]
			69:some_field text computed_by { return age + 2 } constrained_by { return true }
		}`: `expected index, found constrained_by`,

		`type constrained_invalid_arg worksheet {
			69:some_field text constrained_by { return not_a_field == "Alex" }
		}`: `constrained_invalid_arg.some_field references unknown arg not_a_field`,

		`type constrained_no_arg worksheet {
			69:some_field text constrained_by { return true }
		}`: `constrained_no_arg.some_field has no dependencies`,

		`
		type name_reused worksheet {}
		type name_reused worksheet {}
		`: `multiple types name_reused`,

		`
		type name_reused enum {}
		type name_reused worksheet {}
		`: `multiple types name_reused`,

		`
		type name_reused enum {}
		type name_reused enum {}
		`: `multiple types name_reused`,
	}
	for input, msg := range cases {
		_, err := NewDefinitions(strings.NewReader(input))
		assert.EqualErrorf(s.T(), err, msg, "'%s' expecting: %s ", input, msg)
	}
}

func (s *Zuite) TestWorksheetNew_empty() {
	defs, err := NewDefinitions(strings.NewReader(``))
	require.NoError(s.T(), err)
	require.Empty(s.T(), defs.defs)
}

func (s *Zuite) TestWorksheetNew_fieldAtMaxIndex() {
	defs, err := NewDefinitions(strings.NewReader(`
		type simple worksheet {
			65535:index_at_max bool
		}
		`))
	require.NoError(s.T(), err)
	require.Len(s.T(), defs.defs, 1)
	def, ok := defs.defs["simple"]
	require.True(s.T(), ok)
	simple := def.(*Definition)
	require.Len(s.T(), simple.fieldsByIndex, 3)
	require.NotNil(s.T(), simple.fieldsByIndex[65535])
	require.Len(s.T(), simple.fieldsByName, 3)
	require.NotNil(s.T(), simple.fieldsByName["index_at_max"])
}

func (s *Zuite) TestWorksheetNew_multipleDefs() {
	wsDefs := `type one worksheet {1:name text} type two worksheet {1:occupation text}`
	defs, err := NewDefinitions(strings.NewReader(wsDefs))
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(defs.defs))

	for _, wsName := range []string{"one", "two"} {
		_, ok := defs.defs[wsName]
		require.True(s.T(), ok)
	}
}

func (s *Zuite) TestWorksheetNew_origEmpty() {
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// We need to ensure orig is empty since this is a fresh worksheet, and
	// even the special values (e.g. version, id) must be taken into
	// consideration upon save.
	require.Empty(s.T(), ws.orig)
}

func (s *Zuite) TestWorksheetNew_resolveRefTypes() {
	defs := MustNewDefinitions(strings.NewReader(`
		type simple worksheet {
			1:me     simple
			2:myself simple
			3:and_i  simple
			4:not_me even_simpler
		}

		type simple_enum enum {
		}

		type even_simpler worksheet {
			5:not_it simple
			6:enum_field simple_enum
		}

		type refs_in_slices worksheet {
			6:many_simples  []simple
			7:many_simplers [][]even_simpler
			8:many_enums    []simple_enum
		}`))
	var (
		simpleDef       = defs.defs["simple"].(*Definition)
		simpleEnumDef   = defs.defs["simple_enum"].(*EnumType)
		evenSimplerDef  = defs.defs["even_simpler"].(*Definition)
		refsInSlicesDef = defs.defs["refs_in_slices"].(*Definition)
	)

	// refs
	cases := []struct {
		parent *Definition
		field  string
		child  NamedType
	}{
		{simpleDef, "me", simpleDef},
		{simpleDef, "myself", simpleDef},
		{simpleDef, "and_i", simpleDef},
		{simpleDef, "not_me", evenSimplerDef},

		{evenSimplerDef, "enum_field", simpleEnumDef},

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

	manyEnumsTyp := refsInSlicesDef.fieldsByName["many_enums"].typ.(*SliceType)
	assert.True(s.T(), manyEnumsTyp.elementType == simpleEnumDef)
}

func (s *Zuite) TestWorksheetGet_undefinedIfNoValue() {
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	value := ws.MustGet("name")
	require.Equal(s.T(), "undefined", value.String())
}

func (s *Zuite) TestWorksheet_idAndVersion() {
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {1:name text}`))
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
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// initial diff
	require.Equal(s.T(), map[int]change{
		indexId: {
			before: vUndefined,
			after:  NewText(ws.Id()),
		},
		indexVersion: {
			before: vUndefined,
			after:  NewNumberFromInt(1),
		},
	}, ws.diff())

	// set name to Alice
	err = ws.Set("name", alice)
	require.NoError(s.T(), err)

	// now, also expecting Alice
	require.Equal(s.T(), map[int]change{
		indexId: {
			before: vUndefined,
			after:  NewText(ws.Id()),
		},
		indexVersion: {
			before: vUndefined,
			after:  NewNumberFromInt(1),
		},
		1: {
			before: vUndefined,
			after:  alice,
		},
	}, ws.diff())

	// Alice is now Bob
	err = ws.Set("name", bob)
	require.NoError(s.T(), err)

	require.Equal(s.T(), map[int]change{
		indexId: {
			before: vUndefined,
			after:  NewText(ws.Id()),
		},
		indexVersion: {
			before: vUndefined,
			after:  NewNumberFromInt(1),
		},
		1: {
			before: vUndefined,
			after:  bob,
		},
	}, ws.diff())

	// let's fake Bob being there before, and not anymore
	ws.orig[1] = ws.data[1]
	err = ws.Unset("name")
	require.NoError(s.T(), err)

	// now, name should go to an explicit undefine
	require.Equal(s.T(), map[int]change{
		indexId: {
			before: vUndefined,
			after:  NewText(ws.Id()),
		},
		indexVersion: {
			before: vUndefined,
			after:  NewNumberFromInt(1),
		},
		1: {
			before: bob,
			after:  vUndefined,
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
