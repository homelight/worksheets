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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestParser_parseWorksheet() {
	cases := map[string]func(*Definition){
		`worksheet simple {}`: func(ws *Definition) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+0, len(ws.fields))
			require.Equal(s.T(), 2+0, len(ws.fieldsByName))
			require.Equal(s.T(), 2+0, len(ws.fieldsByIndex))
		},
		`worksheet simple {42:full_name text}`: func(ws *Definition) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+1, len(ws.fields))
			require.Equal(s.T(), 2+1, len(ws.fieldsByName))
			require.Equal(s.T(), 2+1, len(ws.fieldsByIndex))

			field := ws.fieldsByName["full_name"]
			require.Equal(s.T(), 42, field.index)
			require.Equal(s.T(), "full_name", field.name)
			require.Equal(s.T(), &TextType{}, field.typ)
			require.Equal(s.T(), ws.fieldsByName["full_name"], field)
			require.Equal(s.T(), ws.fieldsByIndex[42], field)
		},
		`  worksheet simple {42:full_name text 45:happy bool}`: func(ws *Definition) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2+2, len(ws.fields))

			field1 := ws.fieldsByName["full_name"]
			require.Equal(s.T(), 42, field1.index)
			require.Equal(s.T(), "full_name", field1.name)
			require.Equal(s.T(), &TextType{}, field1.typ)
			require.Equal(s.T(), ws.fieldsByName["full_name"], field1)
			require.Equal(s.T(), ws.fieldsByIndex[42], field1)

			field2 := ws.fieldsByName["happy"]
			require.Equal(s.T(), 45, field2.index)
			require.Equal(s.T(), "happy", field2.name)
			require.Equal(s.T(), &BoolType{}, field2.typ)
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

func (s *Zuite) TestParser_parseStatement() {
	cases := map[string]expression{
		`external`:    &tExternal{},
		`return true`: &tReturn{&Bool{true}},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseStatement()
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), "", p.next(), "%s should have reached eof", input)
		assert.Equal(s.T(), expected, actual, input)
	}
}

func (s *Zuite) TestParser_parseExpression() {
	cases := map[string]expression{
		// literals
		`3`:         &Number{3, &NumberType{0}},
		`-5.12`:     &Number{-512, &NumberType{2}},
		`undefined`: &Undefined{},
		`"Alice"`:   &Text{"Alice"},
		`true`:      &Bool{true},

		// var
		`foo`: &tVar{"foo"},

		// unop and binop
		`3 + 4`: &tBinop{opPlus, &Number{3, &NumberType{0}}, &Number{4, &NumberType{0}}, nil},
		`!foo`:  &tUnop{opNot, &tVar{"foo"}},

		// parentheses
		`(true)`:          &Bool{true},
		`(3 + 4)`:         &tBinop{opPlus, &Number{3, &NumberType{0}}, &Number{4, &NumberType{0}}, nil},
		`(3) + (4)`:       &tBinop{opPlus, &Number{3, &NumberType{0}}, &Number{4, &NumberType{0}}, nil},
		`((((3)) + (4)))`: &tBinop{opPlus, &Number{3, &NumberType{0}}, &Number{4, &NumberType{0}}, nil},

		// single expressions being rounded
		`3.00 round down 1`:     &tBinop{opPlus, &Number{300, &NumberType{2}}, &Number{0, &NumberType{0}}, &tRound{"down", 1}},
		`3.00 * 4 round down 5`: &tBinop{opMult, &Number{300, &NumberType{2}}, &Number{4, &NumberType{0}}, &tRound{"down", 5}},
		`3.00 round down 5 * 4`: &tBinop{
			opMult,
			&tBinop{opPlus, &Number{300, &NumberType{2}}, &Number{0, &NumberType{0}}, &tRound{"down", 5}},
			&Number{4, &NumberType{0}},
			nil,
		},

		// rounding closest to the operator it applies
		`1 * 2 round up 4 * 3 round half 5`: &tBinop{
			opMult,
			&tBinop{opMult, &Number{1, &NumberType{0}}, &Number{2, &NumberType{0}}, &tRound{"up", 4}},
			&Number{3, &NumberType{0}},
			&tRound{"half", 5},
		},
		// same way to write the above, because 1 * 2 is the first operator to
		// be folded, it associates with the first rounding mode
		`1 * 2 * 3 round up 4 round half 5`: &tBinop{
			opMult,
			&tBinop{opMult, &Number{1, &NumberType{0}}, &Number{2, &NumberType{0}}, &tRound{"up", 4}},
			&Number{3, &NumberType{0}},
			&tRound{"half", 5},
		},
		// here, because 2 / 3 is the first operator to be folded, the rounding
		// mode applies to this first
		`1 * 2 / 3 round up 4 round half 5`: &tBinop{
			opMult,
			&Number{1, &NumberType{0}},
			&tBinop{opDiv, &Number{2, &NumberType{0}}, &Number{3, &NumberType{0}}, &tRound{"up", 4}},
			&tRound{"half", 5},
		},
		// we move round up 4 closer to the 1 * 2 group, but since the division
		// has precedence, this really means that 2 is first rounded (i.e. it
		// has no bearings on the * binop)
		`1 * 2 round up 4 / 3 round half 5`: &tBinop{
			opMult,
			&Number{1, &NumberType{0}},
			&tBinop{
				opDiv,
				&tBinop{opPlus, &Number{2, &NumberType{0}}, vZero, &tRound{"up", 4}},
				&Number{3, &NumberType{0}},
				&tRound{"half", 5},
			},
			nil,
		},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseExpression(true)
		require.NoError(s.T(), err, input)
		assert.Equal(s.T(), expected, actual, input)
	}
}

func (s *Zuite) TestParser_parseExpressionsAndCheckCompute() {
	// Parsing and evaluating expressions is an easier way to write tests for
	// operator precedence rules. It's great when things are green... And when
	// they are not, it's key to look at the AST to debug.
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

		// gt
		`3 > 2`:           `true`,
		`7 > 7`:           `false`,
		`62 > 100`:        `false`,
		`1 + 1 > 1`:       `true`,
		`7 + 8 > 15`:      `false`,
		`2 + 2 > 5`:       `false`,
		`100 > 90 + 4`:    `true`,
		`18 > 8 + 10`:     `false`,
		`9 > 5 + 4`:       `false`,
		`!(17 > 9)`:       `false`,
		`!(8 > 8)`:        `true`,
		`!(9 > 10)`:       `true`,
		`2.01 > 2.001`:    `true`,
		`17.6 > 17.6`:     `false`,
		`18.1 > 109.0004`: `false`,
		`-4 > -10`:        `true`,
		`-99 > -2`:        `false`,
		`-10 > -10`:       `false`,
		`1.005 > -10`:     `true`,
		`0 > -1000`:       `true`,
		`-5 > -3.01`:      `false`,
		`0 > 5.003`:       `false`,
		`0 > 0`:           `false`,
		`0.000 > 0.000`:   `false`,
		`0 > 0.000000`:    `false`,
		`0.000000 > 0`:    `false`,
		`0.5 > 0.5000`:    `false`,
		`0.5000 > 0.5`:    `false`,
		`-0.120 > -0.12`:  `false`,
		`-0.23 > -0.2300`: `false`,

		// gte
		`6 >= 4`:          `true`,
		`80 >= 80`:        `true`,
		`5 >= 44`:         `false`,
		`8 + 13 >= 9`:     `true`,
		`17 + 17 >= 34`:   `true`,
		`4 + 20 >= 109`:   `false`,
		`872 >= 800 + 10`: `true`,
		`99 >= 19 + 80`:   `true`,
		`45 >= 60 + 17`:   `false`,
		`!(87 >= 10)`:     `false`,
		`!(5 >= 5)`:       `false`,
		`!(17 >= 56)`:     `true`,
		`86.123 >= 55`:    `true`,
		`38.22 >= 38.22`:  `true`,
		`105.7 >= 105.75`: `false`,
		`-4 >= -107`:      `true`,
		`-99 >= -99`:      `true`,
		`-5 >= -3`:        `false`,
		`-3.667 >= -4.01`: `true`,
		`0 >= -1.00`:      `true`,
		`-5 >= -2.9458`:   `false`,
		`0 >= 84.2`:       `false`,
		`0 >= 0`:          `true`,
		`0 >= 0.0000000`:  `true`,
		`0.0000 >= 0`:     `true`,
		`0.000 >= 0.000`:  `true`,
		`0.100 >= 0.1`:    `true`,
		`0.1 >= 0.1000`:   `true`,
		`-0.34 >= -0.340`: `true`,
		`-2.2 >= -2.200`:  `true`,

		// lt
		`7 < 99`:          `true`,
		`13 < 13`:         `false`,
		`11 < 8`:          `false`,
		`18 + 1 < 20`:     `true`,
		`99 + 1 < 100`:    `false`,
		`100 + 47 < 10`:   `false`,
		`999 < 998 + 2`:   `true`,
		`107 < 7 + 100`:   `false`,
		`546 < 200 + 107`: `false`,
		`!(6 < 10)`:       `false`,
		`!(50 < 50)`:      `true`,
		`!(79 < 68)`:      `true`,
		`145.6 < 145.8`:   `true`,
		`14 < 834.34`:     `true`,
		`123.3 < 100.6`:   `false`,
		`-9 < 5`:          `true`,
		`-10 < -10`:       `false`,
		`-3 < -7`:         `false`,
		`0 < 7.00`:        `true`,
		`-34.8 < -20.1`:   `true`,
		`-4.2 < -4.8`:     `false`,
		`0.00000001 < 0`:  `false`,
		`0 < 0`:           `false`,
		`0.000 < 0.000`:   `false`,
		`0.00 < 0`:        `false`,
		`0 < 0.0000000`:   `false`,
		`0.4300 < 0.43`:   `false`,
		`0.3 < 0.30000`:   `false`,
		`-5.61 < -5.6100`: `false`,
		`-4.5 < -4.5000`:  `false`,

		// lte
		`32 <= 71`:        `true`,
		`45 <= 45`:        `true`,
		`120 <= 6`:        `false`,
		`4 + 9 <= 45`:     `true`,
		`13 + 5 <= 18`:    `true`,
		`99 + 1 <= 26`:    `false`,
		`67 <= 88 + 34`:   `true`,
		`45 <= 10 + 35`:   `true`,
		`99 <= 9 + 9`:     `false`,
		`!(32 <= 45)`:     `false`,
		`!(78 <= 78)`:     `false`,
		`!(17 <= 8)`:      `true`,
		`19.5 <= 19.99`:   `true`,
		`20.34 <= 20.34`:  `true`,
		`5.22 <= 5.01`:    `false`,
		`-10 <= -8`:       `true`,
		`-123 <= -123`:    `true`,
		`-89 <= -99`:      `false`,
		`0 <= 10`:         `true`,
		`-4.67 <= -4.67`:  `true`,
		`-22.2 <= -22.9`:  `false`,
		`-10.0 <= -10.43`: `false`,
		`0 <= 0`:          `true`,
		`0.000 <= 0.000`:  `true`,
		`0 <= 0.000000`:   `true`,
		`0.00000 <= 0.0`:  `true`,
		`1.56 <= 1.5600`:  `true`,
		`4.5000 <= 4.5`:   `true`,
		`-98.2 <= -98.20`: `true`,
		`-1.5000 <= -1.5`: `true`,

		// more complicated gt/gte/lt/lte expressions
		`15.899 > 15 + 0.899 round up 0`:        `false`,
		`5999 / 12 round half 2 >= 499.9199999`: `true`,
		`900 - 900.111 < -0.111`:                `false`,
		`17.5 * 13 round down 0 <= 227.0`:       `true`,

		// TODO(pascal): work on convoluted examples below
		// `5 - 1 == 2 * 2 round down 2 round down 0`: `true`,
	}
	for input, output := range cases {
		expected := MustNewValue(output)
		p := newParser(strings.NewReader(input))
		expr, err := p.parseExpression(true)
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

		`1`:                  &Number{1, &NumberType{0}},
		`-123.67`:            &Number{-12367, &NumberType{2}},
		`1.000`:              &Number{1000, &NumberType{3}},
		`1_234.000_000_008`:  &Number{1234000000008, &NumberType{9}},
		`-1_234.000_000_008`: &Number{-1234000000008, &NumberType{9}},

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
		`undefined`: &UndefinedType{},
		`text`:      &TextType{},
		`bool`:      &BoolType{},
		`number[5]`: &NumberType{5},
		`[]bool`:    &SliceType{&BoolType{}},
		`foobar`:    &Definition{name: "foobar"},
	}
	for input, expected := range cases {
		p := newParser(strings.NewReader(input))
		actual, err := p.parseType()
		require.NoError(s.T(), err)
		require.Equal(s.T(), expected, actual)
	}
}

func (s *Zuite) TestTokenPatterns() {
	cases := []struct {
		pattern *tokenPattern
		yes     []string
		no      []string
	}{
		{
			pName,
			[]string{"a", "a_a", "a_0"},
			[]string{"0", "_a", "a_"},
		},
	}
	for _, ex := range cases {
		s.T().Run(ex.pattern.name, func(t *testing.T) {
			for _, y := range ex.yes {
				assert.True(t, ex.pattern.re.MatchString(y))
			}
			for _, n := range ex.no {
				assert.False(t, ex.pattern.re.MatchString(n))
			}
		})
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
		"1// ignore my comment\n4": []string{
			"1",
			"4",
		},
		`1/* this one too */4`: []string{
			"1",
			"4",
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
