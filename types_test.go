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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestTypeAssignableTo() {
	cases := []struct {
		left, right Type
	}{
		{&UndefinedType{}, &TextType{}},
		{&UndefinedType{}, &BoolType{}},
		{&UndefinedType{}, &NumberType{0}},
		{&UndefinedType{}, &NumberType{1}},

		{&TextType{}, &TextType{}},

		{&BoolType{}, &BoolType{}},

		{&NumberType{0}, &NumberType{0}},
		{&NumberType{1}, &NumberType{1}},
	}
	for _, ex := range cases {
		require.True(s.T(), ex.left.AssignableTo(ex.right), "%s should be assignable to %s", ex.left, ex.right)
	}
}

func (s *Zuite) TestTypeNotAssignableTo() {
	cases := []struct {
		left, right Type
	}{
		{&TextType{}, &UndefinedType{}},
		{&BoolType{}, &UndefinedType{}},
		{&NumberType{0}, &UndefinedType{}},
		{&NumberType{1}, &UndefinedType{}},

		{&BoolType{}, &TextType{}},
		{&NumberType{9}, &TextType{}},

		{&TextType{}, &BoolType{}},
		{&NumberType{9}, &BoolType{}},

		{&TextType{}, &NumberType{1}},
		{&NumberType{2}, &NumberType{1}},
	}
	for _, ex := range cases {
		assert.False(s.T(), ex.left.AssignableTo(ex.right), "%s should not be assignable to %s", ex.left, ex.right)
	}
}

func (s *Zuite) TestTypeString() {
	cases := map[Type]string{
		&UndefinedType{}:            "undefined",
		&TextType{}:                 "text",
		&BoolType{}:                 "bool",
		&NumberType{1}:              "number[1]",
		&SliceType{&BoolType{}}:     "[]bool",
		&Definition{name: "simple"}: "simple",
	}
	for typ, expected := range cases {
		assert.Equal(s.T(), expected, typ.String(), expected)
	}
}

func (s *Zuite) TestWorksheetDefinition_Fields() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	fields := ws.Type().(*Definition).Fields()
	require.Len(s.T(), fields, 3)

	expectedFields := []*Field{
		{
			index: 1,
			name:  "name",
			typ:   &TextType{},
		},
		{
			index: -2,
			name:  "id",
			typ:   &TextType{},
		},
		{
			index: -1,
			name:  "version",
			typ:   &NumberType{},
		},
	}
	for _, field := range expectedFields {
		require.Contains(s.T(), fields, field)
	}
}
