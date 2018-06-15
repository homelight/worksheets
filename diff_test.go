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

func (s *Zuite) TestWsRefAtVersion_diffCompare() {
	ws := s.defs.MustNewWorksheet("simple")
	wsRefAtOne := &wsRefAtVersion{
		ws:      ws,
		version: 1,
	}
	wsRefAtFive := &wsRefAtVersion{
		ws:      ws,
		version: 5,
	}

	// refs should be equal to themselves
	s.True(wsRefAtOne.diffCompare(wsRefAtOne))
	s.True(wsRefAtFive.diffCompare(wsRefAtFive))

	// ws == ws ref @ 1 since ws is currently at version 1
	s.True(wsRefAtOne.diffCompare(ws))
	s.True(ws.diffCompare(wsRefAtOne))

	// ws != ws ref @ 5 however since ws is currently at version 1
	s.False(wsRefAtFive.diffCompare(ws))
	s.False(ws.diffCompare(wsRefAtFive))
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
