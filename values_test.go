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
	"github.com/stretchr/testify/assert"
)

func (s *Zuite) TestValueString() {
	ws := s.defs.MustNewWorksheet("simple")
	ws.MustSet("name", alice)
	ws.MustSet("age", MustNewValue("73"))

	cases := map[Value]string{
		vUndefined: "undefined",

		&Text{`Hello, "World"!`}: `"Hello, \"World\"!"`,

		&Bool{true}: "true",

		&Number{1, &NumberType{0}}:     "1",
		&Number{10000, &NumberType{4}}: "1.0000",
		&Number{123, &NumberType{1}}:   "12.3",
		&Number{123, &NumberType{2}}:   "1.23",
		&Number{123, &NumberType{3}}:   "0.123",
		&Number{123, &NumberType{4}}:   "0.0123",
		&Number{-4, &NumberType{0}}:    "-4",
		&Number{-4, &NumberType{2}}:    "-0.04",

		&Slice{elements: []sliceElement{
			{value: &Number{123, &NumberType{1}}},
		}}: "[12.3]",
		&Slice{elements: []sliceElement{
			{value: &Bool{true}},
			{value: &Bool{false}},
		}}: "[true false]",

		ws: `worksheet[age:73 name:"Alice"]`,
	}
	for value, expected := range cases {
		assert.Equal(s.T(), expected, value.String())
	}
}

func (s *Zuite) TestValueEqual() {
	// a.k.a. congruence classes
	buckets := [][]Value{
		{
			NewUndefined(),
			NewUndefined(),
		},
		{
			MustNewValue("1"),
			MustNewValue("1"),
			MustNewValue("1.0"),
			MustNewValue("1.000"),
		},
		{
			MustNewValue("0"),
			MustNewValue("0"),
			MustNewValue("-0"),
			MustNewValue("0.000"),
		},
		{
			NewText("Alice"),
			NewText("Alice"),
		},
		{
			NewText("Bob"),
			NewText("Bob"),
		},
		{
			NewBool(true),
			NewBool(true),
		},
		{
			NewBool(false),
			NewBool(false),
		},
	}

	// all values must be equal within a bucket
	var (
		i, j int
	)
	for _, bucket := range buckets {
		for i = 0; i < len(bucket); i++ {
			for j = i + 1; j < len(bucket); j++ {
				this := bucket[i]
				that := bucket[j]
				assert.True(s.T(), this.Equal(that), "%s == %s", this, that)
			}
		}
	}

	// across buckets, all values must not be equal
	for i = 0; i < len(buckets); i++ {
		for j = i + 1; j < len(buckets); j++ {
			thisBucket := buckets[i]
			thatBucket := buckets[j]
			for _, this := range thisBucket {
				for _, that := range thatBucket {
					assert.True(s.T(), !this.Equal(that), "%s != %s", this, that)
				}
			}
		}
	}
}

func (s *Zuite) TestNumber_Plus() {
	cases := []struct {
		left, right, expected *Number
	}{
		{
			left:     MustNewValue("2").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("5").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("5.0").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3.0").(*Number),
			expected: MustNewValue("5.0").(*Number),
		},
	}
	for _, ex := range cases {
		actual := ex.left.Plus(ex.right)
		assert.Equal(s.T(), ex.expected, actual, "%s + %s", ex.left, ex.right)

		actual = ex.right.Plus(ex.left)
		assert.Equal(s.T(), ex.expected, actual, "%s + %s", ex.right, ex.left)
	}
}

func (s *Zuite) TestNumber_Minus() {
	cases := []struct {
		left, right, expected *Number
	}{
		{
			left:     MustNewValue("2").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("-1").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("-1.0").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3.0").(*Number),
			expected: MustNewValue("-1.0").(*Number),
		},
	}
	for _, ex := range cases {
		actual := ex.left.Minus(ex.right)
		assert.Equal(s.T(), ex.expected, actual, "%s + %s", ex.left, ex.right)
	}
}

func (s *Zuite) TestNumber_Mult() {
	cases := []struct {
		left, right, expected *Number
	}{
		{
			left:     MustNewValue("2").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("6").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3").(*Number),
			expected: MustNewValue("6.0").(*Number),
		},
		{
			left:     MustNewValue("2.0").(*Number),
			right:    MustNewValue("3.0").(*Number),
			expected: MustNewValue("6.00").(*Number),
		},
	}
	for _, ex := range cases {
		actual := ex.left.Mult(ex.right)
		assert.Equal(s.T(), ex.expected, actual, "%s + %s", ex.left, ex.right)

		actual = ex.right.Mult(ex.left)
		assert.Equal(s.T(), ex.expected, actual, "%s + %s", ex.right, ex.left)
	}
}

func (s *Zuite) TestNumber_Round() {
	cases := []struct {
		value, expected *Number
		round           *tRound
	}{
		// down
		{
			value:    MustNewValue("2.34").(*Number),
			round:    &tRound{"down", 2},
			expected: MustNewValue("2.34").(*Number),
		},
		{
			value:    MustNewValue("2.34").(*Number),
			round:    &tRound{"down", 3},
			expected: MustNewValue("2.340").(*Number),
		},
		{
			value:    MustNewValue("2.34").(*Number),
			round:    &tRound{"down", 1},
			expected: MustNewValue("2.3").(*Number),
		},

		// up
		{
			value:    MustNewValue("2.34").(*Number),
			round:    &tRound{"up", 1},
			expected: MustNewValue("2.4").(*Number),
		},
		{
			value:    MustNewValue("2.00").(*Number),
			round:    &tRound{"up", 1},
			expected: MustNewValue("2.0").(*Number),
		},

		// half
		{
			value:    MustNewValue("2.34").(*Number),
			round:    &tRound{"half", 1},
			expected: MustNewValue("2.3").(*Number),
		},
		{
			value:    MustNewValue("2.35").(*Number),
			round:    &tRound{"half", 1},
			expected: MustNewValue("2.4").(*Number),
		},
		{
			value:    MustNewValue("-2.34").(*Number),
			round:    &tRound{"half", 1},
			expected: MustNewValue("-2.3").(*Number),
		},
		{
			value:    MustNewValue("-2.35").(*Number),
			round:    &tRound{"half", 1},
			expected: MustNewValue("-2.4").(*Number),
		},
		{
			value:    MustNewValue("2.304").(*Number),
			round:    &tRound{"half", 2},
			expected: MustNewValue("2.30").(*Number),
		},
		{
			value:    MustNewValue("2.305").(*Number),
			round:    &tRound{"half", 2},
			expected: MustNewValue("2.31").(*Number),
		},
		{
			value:    MustNewValue("-2.304").(*Number),
			round:    &tRound{"half", 2},
			expected: MustNewValue("-2.30").(*Number),
		},
		{
			value:    MustNewValue("-2.305").(*Number),
			round:    &tRound{"half", 2},
			expected: MustNewValue("-2.31").(*Number),
		},
	}
	for _, ex := range cases {
		actual := ex.value.Round(ex.round.mode, ex.round.scale)
		assert.Equal(s.T(), ex.expected, actual,
			"%s round %s %d should equal %s",
			ex.value, ex.round.mode, ex.round.scale, ex.expected)
	}
}

func (s *Zuite) TestNumber_Div() {
	cases := []struct {
		left, right, expected *Number
		round                 *tRound
	}{
		{
			left:     MustNewValue("8").(*Number),
			right:    MustNewValue("2").(*Number),
			expected: MustNewValue("4.0").(*Number),
			round:    &tRound{"up", 1},
		},
		{
			left:     MustNewValue("8").(*Number),
			right:    MustNewValue("2").(*Number),
			expected: MustNewValue("4.00").(*Number),
			round:    &tRound{"up", 2},
		},
		{
			left:     MustNewValue("1").(*Number),
			right:    MustNewValue("7").(*Number),
			expected: MustNewValue("0.1").(*Number),
			round:    &tRound{"down", 1},
		},
		{
			left:     MustNewValue("1").(*Number),
			right:    MustNewValue("7").(*Number),
			expected: MustNewValue("0.2").(*Number),
			round:    &tRound{"up", 1},
		},
		{
			left:     MustNewValue("1").(*Number),
			right:    MustNewValue("7").(*Number),
			expected: MustNewValue("0.14").(*Number),
			round:    &tRound{"down", 2},
		},
		{
			left:     MustNewValue("1").(*Number),
			right:    MustNewValue("7").(*Number),
			expected: MustNewValue("0.15").(*Number),
			round:    &tRound{"up", 2},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.691").(*Number),
			round:    &tRound{"down", 3},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.691").(*Number),
			round:    &tRound{"up", 3},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.6910").(*Number),
			round:    &tRound{"down", 4},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.6911").(*Number),
			round:    &tRound{"up", 4},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.69105").(*Number),
			round:    &tRound{"down", 5},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("1.23").(*Number),
			expected: MustNewValue("5.69106").(*Number),
			round:    &tRound{"up", 5},
		},
		{
			left:     MustNewValue("7.777").(*Number),
			right:    MustNewValue("1.6").(*Number),
			expected: MustNewValue("4").(*Number),
			round:    &tRound{"down", 0},
		},
		{
			left:     MustNewValue("7.777").(*Number),
			right:    MustNewValue("1.6").(*Number),
			expected: MustNewValue("5").(*Number),
			round:    &tRound{"up", 0},
		},
		{
			left:     MustNewValue("7.777").(*Number),
			right:    MustNewValue("1.6").(*Number),
			expected: MustNewValue("4.8").(*Number),
			round:    &tRound{"down", 1},
		},
		{
			left:     MustNewValue("7.777").(*Number),
			right:    MustNewValue("1.6").(*Number),
			expected: MustNewValue("4.9").(*Number),
			round:    &tRound{"up", 1},
		},
		{
			left:     MustNewValue("9.999999").(*Number),
			right:    MustNewValue("1").(*Number),
			expected: MustNewValue("9").(*Number),
			round:    &tRound{"down", 0},
		},
		{
			left:     MustNewValue("9.999999").(*Number),
			right:    MustNewValue("1").(*Number),
			expected: MustNewValue("10").(*Number),
			round:    &tRound{"up", 0},
		},
		{
			left:     MustNewValue("7").(*Number),
			right:    MustNewValue("2.22").(*Number),
			expected: MustNewValue("3.1532").(*Number),
			round:    &tRound{"half", 4},
		},
		{
			left:     MustNewValue("-7").(*Number),
			right:    MustNewValue("2.22").(*Number),
			expected: MustNewValue("-3.1532").(*Number),
			round:    &tRound{"half", 4},
		},
	}
	for _, ex := range cases {
		actual := ex.left.Div(ex.right, ex.round.mode, ex.round.scale)
		assert.Equal(s.T(), ex.expected, actual,
			"%s / %s round %s %d should equal %s",
			ex.left, ex.right, ex.round.mode, ex.round.scale, ex.expected)
	}
}
