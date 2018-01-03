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

		`3 + 4`: &tBinop{opPlus, &Number{3, &tNumberType{0}}, &Number{4, &tNumberType{0}}, nil},
		`!foo`:  &tUnop{opNot, &tVar{"foo"}},

		`(true)`:          &Bool{true},
		`(3 + 4)`:         &tBinop{opPlus, &Number{3, &tNumberType{0}}, &Number{4, &tNumberType{0}}, nil},
		`(3) + (4)`:       &tBinop{opPlus, &Number{3, &tNumberType{0}}, &Number{4, &tNumberType{0}}, nil},
		`((((3)) + (4)))`: &tBinop{opPlus, &Number{3, &tNumberType{0}}, &Number{4, &tNumberType{0}}, nil},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseExpressionOrExternal()
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), "", p.next(), "%s should have reached eof", input)
		assert.Equal(s.T(), expected, actual, input)
	}
}

func (s *Zuite) TestParser_parseExpressionAllAboutRounding() {
	cases := map[string]expression{
		// single expressions being rounded
		`3.00 round down 1`:     &tBinop{opPlus, &Number{300, &tNumberType{2}}, &Number{0, &tNumberType{0}}, &tRound{"down", 1}},
		`3.00 * 4 round down 5`: &tBinop{opMult, &Number{300, &tNumberType{2}}, &Number{4, &tNumberType{0}}, &tRound{"down", 5}},
		`3.00 round down 5 * 4`: &tBinop{
			opMult,
			&tBinop{opPlus, &Number{300, &tNumberType{2}}, &Number{0, &tNumberType{0}}, &tRound{"down", 5}},
			&Number{4, &tNumberType{0}},
			nil,
		},

		// rounding closest to the operator it applies
		`1 * 2 round up 4 * 3 round half 5`: &tBinop{
			opMult,
			&tBinop{opMult, &Number{1, &tNumberType{0}}, &Number{2, &tNumberType{0}}, &tRound{"up", 4}},
			&Number{3, &tNumberType{0}},
			&tRound{"half", 5},
		},
		// same way to write the above, because 1 * 2 is the first operator to
		// be folded, it associates with the first rounding mode
		`1 * 2 * 3 round up 4 round half 5`: &tBinop{
			opMult,
			&tBinop{opMult, &Number{1, &tNumberType{0}}, &Number{2, &tNumberType{0}}, &tRound{"up", 4}},
			&Number{3, &tNumberType{0}},
			&tRound{"half", 5},
		},
		// here, because 2 / 3 is the first operator to be folded, the rounding
		// mode applies to this first
		`1 * 2 / 3 round up 4 round half 5`: &tBinop{
			opMult,
			&Number{1, &tNumberType{0}},
			&tBinop{opDiv, &Number{2, &tNumberType{0}}, &Number{3, &tNumberType{0}}, &tRound{"up", 4}},
			&tRound{"half", 5},
		},
		// we move round up 4 closer to the 1 * 2 group, but since the division
		// has precedence, this really means that 2 is first rounded (i.e. it
		// has no bearings on the * binop)
		`1 * 2 round up 4 / 3 round half 5`: &tBinop{
			opMult,
			&Number{1, &tNumberType{0}},
			&tBinop{
				opDiv,
				&tBinop{opPlus, &Number{2, &tNumberType{0}}, vZero, &tRound{"up", 4}},
				&Number{3, &tNumberType{0}},
				&tRound{"half", 5},
			},
			nil,
		},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseExpressionOrExternal()
		require.NoError(s.T(), err, input)
		assert.Equal(s.T(), expected, actual, input)
	}
}

func (s *Zuite) TestParser_parseExpressionsAndCheckCompute() {
	cases := map[string]string{
		`3`:           `3`,
		`3 + 4`:       `7`,
		`3 + 4 + 5`:   `12`,
		`3 - 4 + 5`:   `4`,
		`3 + 4 - 5`:   `2`,
		`3 + 4 * 5`:   `23`,
		`3 * 4 + 5`:   `17`,
		`3 * (4 + 5)`: `27`,

		`1.2345 round down 0`: `1`,
		`1.2345 round down 1`: `1.2`,
		`1.2345 round down 2`: `1.23`,
		`1.2345 round down 3`: `1.234`,
		`1.2345 round down 4`: `1.2345`,
		`1.2345 round down 5`: `1.23450`,
		`1.2345 round up 0`:   `2`,
		`1.2345 round up 1`:   `1.3`,
		`1.2345 round up 2`:   `1.24`,
		`1.2345 round up 3`:   `1.235`,
		`1.2345 round up 4`:   `1.2345`,
		`1.2345 round up 5`:   `1.23450`,

		` 3 * 5  / 4 round down 0`:             `3`,
		`(3 * 5) / 4 round down 0`:             `3`,
		` 3 * 5  / 4 round up 0`:               `6`,
		`(3 * 5) / 4 round up 0`:               `4`,
		`29 / 2 round down 0 / 7 round down 0`: `2`,
		`29 / 2 round down 0 / 7 round up 0`:   `2`,
		`29 / 2 round up 0 / 7 round down 0`:   `2`,
		`29 / 2 round up 0 / 7 round up 0`:     `3`,

		`!undefined`:                       `undefined`,
		`!true`:                            `false`,
		`3 == 4`:                           `false`,
		`3 + 1 == 4`:                       `true`,
		`4 / 1 round down 0 == 2 * 2`:      `true`,
		`5 - 1 == 2 * 2 round down 0`:      `true`,
		`3 + 1 == 4 && true`:               `true`,
		`"foo" == "foo" && "bar" == "bar"`: `true`,
		`3 + 1 != 4 || true`:               `true`,
		`3 + 1 != 4 || false`:              `false`,
		`"foo" != "foo" || "bar" == "baz"`: `false`,

		`true || undefined`:                `true`,
		`true || 6 / 0 round down 7 == 6`:  `true`,
		`false && undefined`:               `false`,
		`false && 6 / 0 round down 7 == 6`: `false`,

		// TODO(pascal): work on convoluted examples below
		// `5 - 1 == 2 * 2 round down 2 round down 0`: `true`,
	}
	for input, output := range cases {
		expected := MustNewValue(output)
		p := newParser(strings.NewReader(input))
		expr, err := p.parseExpressionOrExternal()
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), "", p.next(), "%s should have reached eof", input)
		actual, err := expr.Compute(nil)
		require.NoError(s.T(), err, input)
		assert.Equal(s.T(), expected, actual, "%s should equal %s was %s", input, output, actual)
	}
}

func (s *Zuite) TestParser_parseExpressionErrors() {
	cases := map[string]string{
		`_1_234`:    `number cannot start with underscore`,
		`1_234_`:    `number cannot terminate with underscore`,
		`1_234.`:    `number cannot terminate with dot`,
		`1_234._67`: `number fraction cannot start with underscore`,
		`1_234.+7`:  `number cannot terminate with dot`,
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		_, err := p.parseExpression(true)
		assert.EqualError(s.T(), err, expected, input)
	}
}

func (s *Zuite) TestParser_parseLiteral() {
	cases := map[string]Value{
		`undefined`: &Undefined{},

		`1`:                  &Number{1, &tNumberType{0}},
		`-123.67`:            &Number{-12367, &tNumberType{2}},
		`1.000`:              &Number{1000, &tNumberType{3}},
		`1_234.000_000_008`:  &Number{1234000000008, &tNumberType{9}},
		`-1_234.000_000_008`: &Number{-1234000000008, &tNumberType{9}},

		`"foo"`: &Text{"foo"},
		`"456"`: &Text{"456"},

		`true`: &Bool{true},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseLiteral()
		require.NoError(s.T(), err)
		assert.Equal(s.T(), expected, actual, input)
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

func (s *Zuite) TestTokenizer() {
	cases := map[string][]string{
		`worksheet simple {1:full_name text}`: []string{
			"worksheet",
			"simple",
			"{",
			"1",
			":",
			"full_name",
			"text",
			"}",
		},
		`1_2___4.6_78___+_1_2`: []string{
			"1",
			"_2___4",
			".6",
			"_78___",
			"+",
			"_1_2",
		},
		`1_2__6+7`: []string{
			"1",
			"_2__6",
			"+",
			"7",
		},
		`1!=2!3! =4==5=6= =7&&8&9& &0||1|2| |done`: []string{
			"1", "!=",
			"2", "!",
			"3", "!", "=",
			"4", "==",
			"5", "=",
			"6", "=", "=",
			"7", "&&",
			"8", "&",
			"9", "&", "&",
			"0", "||",
			"1", "|",
			"2", "|", "|",
			"done",
		},
	}
	for input, toks := range cases {
		p := newParser(strings.NewReader(input))

		for _, tok := range toks {
			require.Equal(s.T(), tok, p.next(), input)
		}
		require.Equal(s.T(), "", p.next(), input)
		require.Equal(s.T(), "", p.next(), input)
	}
}
