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
	"github.com/stretchr/testify/require"
)

var defsForSelectors = `
type child worksheet {
	1:name text
}

type parent worksheet {
	2:ref_to_child     child
	3:refs_to_children []child
}`

func (s *Zuite) TestSelectors() {
	// single field
	child := s.defsForSelectors.MustNewWorksheet("child")
	child.MustSet("name", alice)
	{
		actual, err := tSelector([]string{"name"}).compute(child)
		require.NoError(s.T(), err)
		require.Equal(s.T(), alice, actual)
	}

	// path to field
	parent := s.defsForSelectors.MustNewWorksheet("parent")
	parent.MustSet("ref_to_child", child)
	{
		actual, err := tSelector([]string{"ref_to_child", "name"}).compute(parent)
		require.NoError(s.T(), err)
		require.Equal(s.T(), alice, actual)
	}

	// slice expression
	parent.MustAppend("refs_to_children", child)
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Equal(s.T(), []Value{alice}, slice.Elements())
		require.Equal(s.T(), &SliceType{&TextType{}}, slice.Type())
	}
}

func (s *Zuite) TestSelectorSliceTypes() {
	child := s.defsForSelectors.MustNewWorksheet("child")
	parent := s.defsForSelectors.MustNewWorksheet("parent")
	parent.MustAppend("refs_to_children", child)
	// even with an undefined value, slice type should match field def type
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Equal(s.T(), []Value{NewUndefined()}, slice.Elements())
		require.Equal(s.T(), &SliceType{&TextType{}}, slice.Type())
	}
	// a selected empty slice should still be of the correct type
	parent.MustDel("refs_to_children", 0)
	{
		actual, err := tSelector([]string{"refs_to_children", "name"}).compute(parent)
		require.NoError(s.T(), err)
		slice, ok := actual.(*Slice)
		require.True(s.T(), ok)
		require.Empty(s.T(), slice.Elements())
		require.Equal(s.T(), &SliceType{&TextType{}}, slice.Type())
	}
}

func (s *Zuite) TestFnArgs() {
	args := newFnArgs(nil, []expression{vZero})
	s.Len(args.exprs, 1)
	s.NotNil(args.exprs[0])
	s.Len(args.values, 1)
	s.Nil(args.values[0])
	s.Len(args.errs, 1)
	s.Nil(args.errs[0])

	v, err := args.get(0)
	require.NoError(s.T(), err)
	require.Equal(s.T(), vZero, v)

	s.Len(args.exprs, 1)
	s.Nil(args.exprs[0])
	s.Len(args.values, 1)
	s.Equal(vZero, args.values[0])
	s.Len(args.errs, 1)
	s.Nil(args.errs[0])
}
