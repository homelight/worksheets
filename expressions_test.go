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

var defsForSelectors = MustNewDefinitions(strings.NewReader(`
worksheet child {
	1:name text
}

worksheet parent {
	2:ref_to_child     child
	3:refs_to_children []child
}`))

func (s *Zuite) TestSelectors() {
	// single field
	child := defsForSelectors.MustNewWorksheet("child")
	child.MustSet("name", alice)
	{
		actual, err := tSelector([]string{"name"}).Compute(child)
		require.NoError(s.T(), err)
		require.Equal(s.T(), alice, actual)
	}

	// path to field
	parent := defsForSelectors.MustNewWorksheet("parent")
	parent.MustSet("ref_to_child", child)
	{
		actual, err := tSelector([]string{"ref_to_child", "name"}).Compute(parent)
		require.NoError(s.T(), err)
		require.Equal(s.T(), alice, actual)
	}

	// slice expression
	parent.MustAppend("refs_to_children", child)
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).Compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Equal(s.T(), []Value{alice}, slice.Elements())
		require.Equal(s.T(), &SliceType{alice.Type()}, slice.Type())
	}
}

func (s *Zuite) TestSelectorSliceTypes() {
	child_1 := defsForSelectors.MustNewWorksheet("child")
	child_2 := defsForSelectors.MustNewWorksheet("child")
	parent := defsForSelectors.MustNewWorksheet("parent")
	parent.MustAppend("refs_to_children", child_1)
	parent.MustAppend("refs_to_children", child_2)
	// all slice values undefined, slice type is also undefined
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).Compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Equal(s.T(), []Value{NewUndefined(), NewUndefined()}, slice.Elements())
		require.Equal(s.T(), &SliceType{&UndefinedType{}}, slice.Type())
	}

	// at least one value set - slice type should match value type
	child_1.MustSet("name", alice)
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).Compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Equal(s.T(), []Value{alice, NewUndefined()}, slice.Elements())
		require.Equal(s.T(), &SliceType{alice.Type()}, slice.Type())
	}
}
