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

func (s *Zuite) TestRuntime_parseAndEvalExpr() {
	cases := map[string]string{
		// equal
		`4 == 4`:           `true`,
		`2 == 9`:           `false`,
		`2.4 == 2.400`:     `true`,
		`3.0 == 3.001`:     `false`,
		`-10 == -10`:       `true`,
		`-12 == -80`:       `false`,
		`-1.9 == -1.900`:   `true`,
		`-7.600 == -7.601`: `false`,

		// not equal
		`7 != 3`:           `true`,
		`8 != 8`:           `false`,
		`9.01 != 9.1`:      `true`,
		`3.30 != 3.3`:      `false`,
		`-98 != -14`:       `true`,
		`-3 != -3`:         `false`,
		`-8.69 != -8.7`:    `true`,
		`-2.00000 != -2.0`: `false`,

		// greater than
		`3 > 2`:           `true`,
		`7 > 7`:           `false`,
		`62 > 100`:        `false`,
		`2.01 > 2.001`:    `true`,
		`17.6 > 17.60`:    `false`,
		`18.1 > 109.0004`: `false`,
		`-4 > -10`:        `true`,
		`-10 > -10`:       `false`,
		`-99 > -2`:        `false`,
		`-6.01 > -11.0`:   `true`,
		`-9.8 > -9.80`:    `false`,
		`-100.1 > -3.99`:  `false`,
		`0 > 0`:           `false`,
		`0.000 > 0.00`:    `false`,
		`0.0 > 0.00000`:   `false`,
		`500 > -500`:      `true`,
		`-500 > 500`:      `false`,
		`-500 > -500`:     `false`,

		// greater than or equal
		`3 >= 2`:           `true`,
		`7 >= 7`:           `true`,
		`62 >= 100`:        `false`,
		`2.01 >= 2.001`:    `true`,
		`17.6 >= 17.60`:    `true`,
		`18.1 >= 109.0004`: `false`,
		`-4 >= -10`:        `true`,
		`-10 >= -10`:       `true`,
		`-99 >= -2`:        `false`,
		`-6.01 >= -11.0`:   `true`,
		`-9.8 >= -9.80`:    `true`,
		`-100.1 >= -3.99`:  `false`,
		`0 >= 0`:           `true`,
		`0.000 >= 0.00`:    `true`,
		`0.0 >= 0.00000`:   `true`,
		`500 >= -500`:      `true`,
		`-500 >= 500`:      `false`,
		`-500 >= -500`:     `true`,

		// less than
		`7 < 99`:         `true`,
		`13 < 13`:        `false`,
		`11 < 8`:         `false`,
		`145.6 < 145.85`: `true`,
		`83.3 < 83.30`:   `false`,
		`123.443 < 90.6`: `false`,
		`-9 < 5`:         `true`,
		`-10 < -10`:      `false`,
		`-3 < -7`:        `false`,
		`-5.3 < -1.99`:   `true`,
		`-6.0 < -6`:      `false`,
		`-2.5 < -7.667`:  `false`,
		`0 < 0`:          `false`,
		`0.000 < 0.00`:   `false`,
		`0.0 < 0.00000`:  `false`,
		`-100 < 100`:     `true`,
		`100 < -100`:     `false`,
		`-100 < -100`:    `false`,

		// less than or equal
		`7 <= 99`:         `true`,
		`13 <= 13`:        `true`,
		`11 <= 8`:         `false`,
		`145.6 <= 145.85`: `true`,
		`83.3 <= 83.30`:   `true`,
		`123.443 <= 90.6`: `false`,
		`-9 <= 5`:         `true`,
		`-10 <= -10`:      `true`,
		`-3 <= -7`:        `false`,
		`-5.3 <= -1.99`:   `true`,
		`-6.0 <= -6`:      `true`,
		`-2.5 <= -7.667`:  `false`,
		`0 <= 0`:          `true`,
		`0.000 <= 0.00`:   `true`,
		`0.0 <= 0.00000`:  `true`,
		`-100 <= 100`:     `true`,
		`100 <= -100`:     `false`,
		`-100 <= -100`:    `true`,

		// undefined gt/gte/lt/lte expressions
		`54 < undefined`:         `undefined`,
		`undefined < 83`:         `undefined`,
		`undefined < undefined`:  `undefined`,
		`5 <= undefined`:         `undefined`,
		`undefined <= 7`:         `undefined`,
		`undefined <= undefined`: `undefined`,
		`31 > undefined`:         `undefined`,
		`undefined > 26`:         `undefined`,
		`undefined > undefined`:  `undefined`,
		`45 >= undefined`:        `undefined`,
		`undefined >= 86`:        `undefined`,
		`undefined >= undefined`: `undefined`,

		// len
		`len("Bob")`:     `3`,
		`len(undefined)`: `undefined`,
		`len(slice_t)`:   `2`,
		`len(text)`:      `5`,

		// sum
		`sum(slice_n0)`: `10`,
		`sum(slice_n2)`: `11.10`,
		`sum(slice_nu)`: `undefined`,

		// sumiftrue
		`sumiftrue(slice_n0, slice_b)`:   `7`,
		`sumiftrue(slice_n2, slice_b)`:   `7.77`,
		`sumiftrue(slice_nu, slice_b)`:   `undefined`,
		`sumiftrue(slice_n0, slice_bu)`:  `undefined`,
		`sumiftrue(slice_nu, slice_bu)`:  `undefined`,
		`sumiftrue(undefined, slice_b)`:  `undefined`,
		`sumiftrue(slice_n0, undefined)`: `undefined`,

		// if
		`if(true, 1, 3)`:                                                   `1`,
		`if(false, 1, 3)`:                                                  `3`,
		`if(undefined, 1, 3)`:                                              `undefined`,
		`if(true, 1, 3 / 0 round down 0)`:                                  `1`,
		`if(false, 1 / 0 round down 0, 3)`:                                 `3`,
		`if(0 < -1, "unused", if("a" == "a", "good", 1 / 0 round down 0))`: `"good"`,
		`if(true, 1)`:  `1`,
		`if(false, 1)`: `undefined`,
	}
	for input, output := range cases {
		// fixture
		ws := s.defs.MustNewWorksheet("all_types")
		ws.MustSet("text", alice)
		ws.MustAppend("slice_t", alice)
		ws.MustAppend("slice_t", bob)
		ws.MustAppend("slice_n0", NewNumberFromInt(2))
		ws.MustAppend("slice_n0", NewNumberFromInt(3))
		ws.MustAppend("slice_n0", NewNumberFromInt(5))
		ws.MustAppend("slice_n2", NewNumberFromFloat64(2.22))
		ws.MustAppend("slice_n2", NewNumberFromFloat64(3.33))
		ws.MustAppend("slice_n2", NewNumberFromFloat64(5.55))
		ws.MustAppend("slice_nu", NewUndefined())
		ws.MustAppend("slice_nu", NewNumberFromInt(3))
		ws.MustAppend("slice_nu", NewNumberFromInt(5))
		ws.MustAppend("slice_b", NewBool(true))
		ws.MustAppend("slice_b", NewBool(false))
		ws.MustAppend("slice_b", NewBool(true))
		ws.MustAppend("slice_bu", NewUndefined())
		ws.MustAppend("slice_bu", NewBool(false))
		ws.MustAppend("slice_bu", NewBool(true))

		// test
		expected := MustNewValue(output)
		p := newParser(strings.NewReader(input))

		expr, err := p.parseExpression(true)
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), "", p.next(), "%s should have reached eof", input)

		actual, err := expr.compute(ws)
		if assert.NoError(s.T(), err, input) {
			assert.Equal(s.T(), expected, actual, "%s should equal %s was %s", input, output, actual)
		}
	}
}

func (s *Zuite) TestRuntime_parseAndEvalExprExpectingFailure() {
	cases := map[string]string{
		`no_such_func()`: `unknown function no_such_func`,
		`no.such.func()`: `unknown function no.such.func`,
		`len(1, 2)`:      `len: 1 argument(s) expected but 2 found`,
		`len(1)`:         `len: argument #1 expected to be text, or slice`,
		`sum(1, 2)`:      `sum: 1 argument(s) expected but 2 found`,
		`sum(1)`:         `sum: argument #1 expected to be slice of numbers`,
		`sum(slice_t)`:   `sum: argument #1 expected to be slice of numbers`,
		`if(1)`:          `if: at least 2 argument(s) expected but only 1 found`,
		`if(1,2,3,4)`:    `if: at most 3 argument(s) expected but 4 found`,
	}
	for input, output := range cases {
		// fixture
		ws := s.defs.MustNewWorksheet("all_types")

		// test
		p := newParser(strings.NewReader(input))

		expr, err := p.parseExpression(true)
		require.NoError(s.T(), err, input)
		require.Equal(s.T(), "", p.next(), "%s should have reached eof", input)

		_, err = expr.compute(ws)
		assert.EqualError(s.T(), err, output, input)
	}
}
