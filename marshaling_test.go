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
	"encoding/json"

	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestMarshaling_simple() {
	ws := defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustSet("text", NewText(`some text with " and stuff`))
	ws.MustSet("bool", NewBool(true))
	ws.MustSet("num_0", MustNewValue("123"))
	ws.MustSet("num_2", MustNewValue("123.45"))
	ws.MustSet("undefined", &Undefined{})

	expected := `{"the-id":{
		"text": "some text with \" and stuff",
		"bool": true,
		"num_0": "123",
		"num_2": "123.45",
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfText() {
	ws := defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustAppend("slice_t", alice)
	ws.MustAppend("slice_t", bob)

	expected := `{"the-id":{
		"slice_t": ["Alice", "Bob"],
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceWithUndefined() {
	ws := defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustAppend("slice_t", &Undefined{})
	ws.MustAppend("slice_t", bob)

	expected := `{"the-id":{
		"slice_t": [null, "Bob"],
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_wsRef() {
	parent := defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	child := defs.MustNewWorksheet("all_types")
	forciblySetId(child, "the-child")

	parent.MustSet("ws", child)

	expected := `{
	"the-parent":{
		"ws": "the-child",
		"id": "the-parent",
		"version":"1"
	},
	"the-child":{
		"id": "the-child",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_wsRefToItself() {
	parent := defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent-and-child")

	parent.MustSet("ws", parent)

	expected := `{"the-parent-and-child":{
		"ws": "the-parent-and-child",
		"id": "the-parent-and-child",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfRefs() {
	parent := defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	child1 := defs.MustNewWorksheet("all_types")
	forciblySetId(child1, "the-child1")

	child2 := defs.MustNewWorksheet("all_types")
	forciblySetId(child2, "the-child2")

	parent.MustAppend("slice_ws", child1)
	parent.MustAppend("slice_ws", child2)

	expected := `{
	"the-parent":{
		"slice_ws": ["the-child1", "the-child2"],
		"id": "the-parent",
		"version":"1"
	},
	"the-child1":{
		"id": "the-child1",
		"version":"1"
	},
	"the-child2":{
		"id": "the-child2",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfRefsToItself() {
	parent := defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	parent.MustAppend("slice_ws", parent)
	parent.MustAppend("slice_ws", parent)

	expected := `{"the-parent":{
		"slice_ws": ["the-parent", "the-parent"],
		"id": "the-parent",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) requireSameJson(expected string, actual []byte) {
	var e, a interface{}

	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		require.Fail(s.T(), "bad expected JSON", expected)
	}
	if err := json.Unmarshal(actual, &a); err != nil {
		require.Fail(s.T(), "bad actual JSON", actual)
	}
	require.Equal(s.T(), e, a)
}
