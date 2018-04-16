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
	"strings"

	"github.com/stretchr/testify/require"
)

var cloneDefs = MustNewDefinitions(strings.NewReader(`
worksheet dup_me {
	1: value   text
	2: v_slice []number[0]
	3: r_slice []dup_me
	4: ref1    dup_me
	5: ref2    dup_me
}
`))

func (s *Zuite) TestClone_simple() {
	ws := cloneDefs.MustNewWorksheet("dup_me")
	ws.MustSet("value", NewText("Mary had a little lamb"))

	dup := ws.Clone()
	require.True(s.T(), ws != dup, "dup must be a different instance than ws")
	require.NotEqual(s.T(), ws.Id(), dup.Id())
	require.Len(s.T(), dup.orig, 0)
	require.Equal(s.T(), map[int]Value{
		indexId:      NewText(dup.Id()),
		indexVersion: MustNewValue("1"),
		1:            NewText("Mary had a little lamb"),
	}, dup.data)
	require.Len(s.T(), dup.parents, 0)
}

func (s *Zuite) TestClone_withOneWsRef() {
	ws1 := cloneDefs.MustNewWorksheet("dup_me")
	ws2 := cloneDefs.MustNewWorksheet("dup_me")
	ws1.MustSet("ref1", ws2)

	dup1 := ws1.Clone()
	dup2 := dup1.MustGet("ref1").(*Worksheet)

	require.True(s.T(), ws1 != dup1, "dup1 must be a different instance than ws1")
	require.True(s.T(), ws2 != dup2, "dup2 must be a different instance than ws2")

	require.NotEqual(s.T(), ws1.Id(), dup1.Id())
	require.NotEqual(s.T(), ws2.Id(), dup2.Id())

	require.Len(s.T(), dup1.orig, 0)
	require.Len(s.T(), dup2.orig, 0)

	require.Equal(s.T(), map[int]Value{
		indexId:      NewText(dup1.Id()),
		indexVersion: MustNewValue("1"),
		4:            dup2,
	}, dup1.data)
	require.Equal(s.T(), map[int]Value{
		indexId:      NewText(dup2.Id()),
		indexVersion: MustNewValue("1"),
	}, dup2.data)

	require.Len(s.T(), dup1.parents, 0)
	require.Equal(s.T(), parentsRefs(map[string]map[int]map[string]*Worksheet{
		"dup_me": {
			4: {
				dup1.Id(): dup1,
			},
		},
	}), dup2.parents)
}

func (s *Zuite) TestClone_withTwoWsRefToSameWs() {
	ws1 := cloneDefs.MustNewWorksheet("dup_me")
	ws2 := cloneDefs.MustNewWorksheet("dup_me")
	ws1.MustSet("ref1", ws2)
	ws1.MustSet("ref2", ws2)

	dup1 := ws1.Clone()
	dup2a := dup1.MustGet("ref1").(*Worksheet)
	dup2b := dup1.MustGet("ref2").(*Worksheet)

	require.True(s.T(), ws1 != dup1, "dup1 must be a different instance than ws1")
	require.True(s.T(), ws2 != dup2a, "dup2a must be a different instance than ws2")
	require.True(s.T(), ws2 != dup2b, "dup2b must be a different instance than ws2")
	require.True(s.T(), dup2a == dup2b, "dup2a must be the same instance as dup2b")

	dup2 := dup2a // == dup2b

	require.NotEqual(s.T(), ws1.Id(), dup1.Id())
	require.NotEqual(s.T(), ws2.Id(), dup2.Id())

	require.Len(s.T(), dup1.orig, 0)
	require.Len(s.T(), dup2a.orig, 0)

	require.Equal(s.T(), map[int]Value{
		indexId:      NewText(dup1.Id()),
		indexVersion: MustNewValue("1"),
		4:            dup2,
		5:            dup2,
	}, dup1.data)
	require.Equal(s.T(), map[int]Value{
		indexId:      NewText(dup2.Id()),
		indexVersion: MustNewValue("1"),
	}, dup2.data)

	require.Len(s.T(), dup1.parents, 0)
	require.Equal(s.T(), parentsRefs(map[string]map[int]map[string]*Worksheet{
		"dup_me": {
			4: {
				dup1.Id(): dup1,
			},
			5: {
				dup1.Id(): dup1,
			},
		},
	}), dup2.parents)
}

func (s *Zuite) TestClone_withSliceOfValues() {
	ws := cloneDefs.MustNewWorksheet("dup_me")
	ws.MustAppend("v_slice", MustNewValue("2"))
	ws.MustAppend("v_slice", MustNewValue("3"))
	ws.MustAppend("v_slice", MustNewValue("3"))
	ws.MustAppend("v_slice", MustNewValue("5"))
	ws.MustAppend("v_slice", MustNewValue("8"))
	ws.MustAppend("v_slice", MustNewValue("13"))
	ws.MustDel("v_slice", 2)
	wsSlice := ws.data[2].(*Slice)

	dup := ws.Clone()
	require.True(s.T(), ws != dup, "dup must be a different instance than ws")
	require.NotEqual(s.T(), ws.Id(), dup.Id())
	require.Len(s.T(), dup.orig, 0)
	require.Len(s.T(), dup.data, 3)
	require.Len(s.T(), dup.parents, 0)

	// Highlighting that dupSlice is a fresh new slice, where elements have been
	// set to exactly what is in wsSlice at the time of doing the cloning, i.e.
	// the lastRank is not preserved, because we are doing a clean slate copy.
	// For wsSlice lastRank is 6, since we've added 6 elements, and removed one,
	// but for dupSlice, the lastRank is 5, since we're only copying the 5
	// remaining elements when cloning.
	require.Equal(s.T(), 6, wsSlice.lastRank)

	dupSlice := dup.data[2].(*Slice)
	require.NotEqual(s.T(), wsSlice.id, dupSlice.id)
	require.Equal(s.T(), wsSlice.typ, dupSlice.typ)
	require.Equal(s.T(), 5, dupSlice.lastRank)
	require.Equal(s.T(), []sliceElement{
		{1, MustNewValue("2")},
		{2, MustNewValue("3")},
		{3, MustNewValue("5")},
		{4, MustNewValue("8")},
		{5, MustNewValue("13")},
	}, dupSlice.elements)
}

func (s *Zuite) TestClone_withSliceOfRefs() {
	ws := cloneDefs.MustNewWorksheet("dup_me")
	child1 := cloneDefs.MustNewWorksheet("dup_me")
	child2 := cloneDefs.MustNewWorksheet("dup_me")
	child3 := cloneDefs.MustNewWorksheet("dup_me")
	ws.MustAppend("r_slice", child1)
	ws.MustAppend("r_slice", child2)
	ws.MustAppend("r_slice", child3)
	ws.MustSet("ref1", child1)
	ws.MustSet("ref2", child2)

	dup := ws.Clone()

	// Since we're testing cloning behavior in other tests, we're only checking
	// that all pointers are set properly here.

	dupSlice := dup.data[3].(*Slice)
	dupChild1 := dupSlice.elements[0].value.(*Worksheet)
	dupChild2 := dupSlice.elements[1].value.(*Worksheet)
	dupChild3 := dupSlice.elements[2].value.(*Worksheet)

	require.True(s.T(), ws != dup, "dup must be a different instance than ws")
	require.True(s.T(), child1 != dupChild1, "dupChild1 must be a different instance than child1")
	require.True(s.T(), child2 != dupChild2, "dupChild2 must be a different instance than child2")
	require.True(s.T(), child3 != dupChild3, "dupChild3 must be a different instance than child3")

	require.True(s.T(), dup.data[4] == dupChild1, "r_slice[0] should point to ref1")
	require.True(s.T(), dup.data[5] == dupChild2, "r_slice[1] should point to ref2")

	// Parents.

	require.Len(s.T(), dup.parents, 0)
	require.Equal(s.T(), parentsRefs(map[string]map[int]map[string]*Worksheet{
		"dup_me": {
			3: {
				dup.Id(): dup,
			},
			4: {
				dup.Id(): dup,
			},
		},
	}), dupChild1.parents)
	require.Equal(s.T(), parentsRefs(map[string]map[int]map[string]*Worksheet{
		"dup_me": {
			3: {
				dup.Id(): dup,
			},
			5: {
				dup.Id(): dup,
			},
		},
	}), dupChild2.parents)
	require.Equal(s.T(), parentsRefs(map[string]map[int]map[string]*Worksheet{
		"dup_me": {
			3: {
				dup.Id(): dup,
			},
		},
	}), dupChild3.parents)
}
