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

func (s *Zuite) TestParser_parseWorksheet() {
	cases := map[string]func(*tWorksheet){
		`worksheet simple {}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+0, len(ws.fields))
			require.Equal(s.T(), 2+0, len(ws.fieldsByName))
			require.Equal(s.T(), 2+0, len(ws.fieldsByIndex))
		},
		`worksheet simple {42:full_name text}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+1, len(ws.fields))
			require.Equal(s.T(), 2+1, len(ws.fieldsByName))
			require.Equal(s.T(), 2+1, len(ws.fieldsByIndex))

			field := ws.fieldsByName["full_name"]
			require.Equal(s.T(), 42, field.index)
			require.Equal(s.T(), "full_name", field.name)
			require.Equal(s.T(), &tTextType{}, field.typ)
			require.Equal(s.T(), ws.fieldsByName["full_name"], field)
			require.Equal(s.T(), ws.fieldsByIndex[42], field)
		},
		`  worksheet simple {42:full_name text 45:happy bool}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+2, len(ws.fields))

			field1 := ws.fieldsByName["full_name"]
			require.Equal(s.T(), 42, field1.index)
			require.Equal(s.T(), "full_name", field1.name)
			require.Equal(s.T(), &tTextType{}, field1.typ)
			require.Equal(s.T(), ws.fieldsByName["full_name"], field1)
			require.Equal(s.T(), ws.fieldsByIndex[42], field1)

			field2 := ws.fieldsByName["happy"]
			require.Equal(s.T(), 45, field2.index)
			require.Equal(s.T(), "happy", field2.name)
			require.Equal(s.T(), &tBoolType{}, field2.typ)
			require.Equal(s.T(), ws.fieldsByName["happy"], field2)
			require.Equal(s.T(), ws.fieldsByIndex[45], field2)
		},
	}
	for input, checks := range cases {
		p := newParser(strings.NewReader(input))
		ws, err := p.parseWorksheet()
		require.NoError(s.T(), err)
		checks(ws)
	}
}

func (s *Zuite) TestParser_parseWorksheetErrors() {
	cases := []string{
		"worksheet simple {\n\t42:full_name\n\ttext 42:happy bool\n}",
		"worksheet simple {\n\t42:same_name\n\ttext 43:same_name bool\n}",
	}
	for _, input := range cases {
		p := newParser(strings.NewReader(input))
		_, err := p.parseWorksheet()
		require.NotNil(s.T(), err)
		// TODO(pascal): verify error messages are nice
	}
}

func (s *Zuite) TestParser_parseExpressionOrExternal() {
	cases := map[string]expression{
		`external`: &tExternal{},

		`3`:         &Number{3, &tNumberType{0}},
		`-5.12`:     &Number{-512, &tNumberType{2}},
		`undefined`: &Undefined{},
		`"Alice"`:   &Text{"Alice"},
		`true`:      &Bool{true},

		`foo`: &tVar{"foo"},

		`3 + 4`: &tBinop{opPlus, &Number{3, &tNumberType{0}}, &Number{4, &tNumberType{0}}},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseExpressionOrExternal()
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), expected, actual, input)
	}
}

func (s *Zuite) TestParser_parseLiteral() {
	cases := map[string]Value{
		`undefined`: &Undefined{},

		`1`:       &Number{1, &tNumberType{0}},
		`-123.67`: &Number{-12367, &tNumberType{2}},
		`1.000`:   &Number{1000, &tNumberType{3}},

		`"foo"`: &Text{"foo"},
		`"456"`: &Text{"456"},

		`true`: &Bool{true},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseLiteral()
		require.NoError(s.T(), err)
		require.Equal(s.T(), expected, actual)
	}
}

func (s *Zuite) TestParser_parseType() {
	cases := map[string]Type{
		`undefined`: &tUndefinedType{},
		`text`:      &tTextType{},
		`bool`:      &tBoolType{},
		`number[5]`: &tNumberType{5},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseType()
		require.NoError(s.T(), err)
		require.Equal(s.T(), expected, actual)
	}
}

func (s *Zuite) TestTokenizer_Simple() {
	input := `worksheet simple {1:full_name text}`
	p := newParser(strings.NewReader(input))

	toks := []string{
		"worksheet",
		"simple",
		"{",
		"1",
		":",
		"full_name",
		"text",
		"}",
	}
	for _, tok := range toks {
		require.Equal(s.T(), tok, p.next())
	}
	require.Equal(s.T(), "", p.next())
	require.Equal(s.T(), "", p.next())
}
