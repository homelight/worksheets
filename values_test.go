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
	ws.MustSet("age", NewNumberFromInt(73))

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
			&Number{1, &NumberType{0}},
			&Number{1, &NumberType{0}},
			&Number{10, &NumberType{1}},
			&Number{1000, &NumberType{3}},
		},
		{
			&Number{0, &NumberType{0}},
			&Number{0, &NumberType{1}},
			&Number{0, &NumberType{2}},
			&Number{0, &NumberType{3}},
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
		left, right *Number
		expected    string
	}{
		{
			left:     NewNumberFromInt(2),
			right:    NewNumberFromInt(3),
			expected: "5",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3),
			expected: "5.0",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3).Round(ModeHalf, 1),
			expected: "5.0",
		},
	}
	for _, ex := range cases {
		actual := ex.left.Plus(ex.right)
		assert.Equal(s.T(), ex.expected, actual.String(), "%s + %s", ex.left, ex.right)

		actual = ex.right.Plus(ex.left)
		assert.Equal(s.T(), ex.expected, actual.String(), "%s + %s", ex.right, ex.left)
	}
}

func (s *Zuite) TestNumber_Minus() {
	cases := []struct {
		left, right *Number
		expected    string
	}{
		{
			left:     NewNumberFromInt(2),
			right:    NewNumberFromInt(3),
			expected: "-1",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3),
			expected: "-1.0",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3).Round(ModeHalf, 1),
			expected: "-1.0",
		},
	}
	for _, ex := range cases {
		actual := ex.left.Minus(ex.right)
		assert.Equal(s.T(), ex.expected, actual.String(), "%s + %s", ex.left, ex.right)
	}
}

func (s *Zuite) TestNumber_Mult() {
	cases := []struct {
		left, right *Number
		expected    string
	}{
		{
			left:     NewNumberFromInt(2),
			right:    NewNumberFromInt(3),
			expected: "6",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3),
			expected: "6.0",
		},
		{
			left:     NewNumberFromInt(2).Round(ModeHalf, 1),
			right:    NewNumberFromInt(3).Round(ModeHalf, 1),
			expected: "6.00",
		},
	}
	for _, ex := range cases {
		actual := ex.left.Mult(ex.right)
		assert.Equal(s.T(), ex.expected, actual.String(), "%s + %s", ex.left, ex.right)

		actual = ex.right.Mult(ex.left)
		assert.Equal(s.T(), ex.expected, actual.String(), "%s + %s", ex.right, ex.left)
	}
}

func (s *Zuite) TestNumber_Round() {
	cases := []struct {
		value    *Number
		round    *tRound
		expected string
	}{
		// down
		{
			value:    NewNumberFromFloat64(2.34),
			round:    &tRound{"down", 2},
			expected: "2.34",
		},
		{
			value:    NewNumberFromFloat64(2.34),
			round:    &tRound{"down", 3},
			expected: "2.340",
		},
		{
			value:    NewNumberFromFloat64(2.34),
			round:    &tRound{"down", 1},
			expected: "2.3",
		},

		// up
		{
			value:    NewNumberFromFloat64(2.34),
			round:    &tRound{"up", 1},
			expected: "2.4",
		},
		{
			value:    &Number{2, &NumberType{0}},
			round:    &tRound{"up", 1},
			expected: "2.0",
		},
		{
			value:    &Number{200, &NumberType{2}},
			round:    &tRound{"up", 1},
			expected: "2.0",
		},

		// half
		{
			value:    NewNumberFromFloat64(2.34),
			round:    &tRound{"half", 1},
			expected: "2.3",
		},
		{
			value:    NewNumberFromFloat64(2.35),
			round:    &tRound{"half", 1},
			expected: "2.4",
		},
		{
			value:    NewNumberFromFloat64(-2.34),
			round:    &tRound{"half", 1},
			expected: "-2.3",
		},
		{
			value:    NewNumberFromFloat64(-2.35),
			round:    &tRound{"half", 1},
			expected: "-2.4",
		},
		{
			value:    NewNumberFromFloat64(2.304),
			round:    &tRound{"half", 2},
			expected: "2.30",
		},
		{
			value:    NewNumberFromFloat64(2.305),
			round:    &tRound{"half", 2},
			expected: "2.31",
		},
		{
			value:    NewNumberFromFloat64(-2.304),
			round:    &tRound{"half", 2},
			expected: "-2.30",
		},
		{
			value:    NewNumberFromFloat64(-2.305),
			round:    &tRound{"half", 2},
			expected: "-2.31",
		},
	}
	for _, ex := range cases {
		actual := ex.value.Round(ex.round.mode, ex.round.scale)
		assert.Equal(s.T(), ex.expected, actual.String(),
			"%s round %s %d should equal %s",
			ex.value, ex.round.mode, ex.round.scale, ex.expected)
	}
}

func (s *Zuite) TestNumber_Div() {
	cases := []struct {
		left, right *Number
		round       *tRound
		expected    string
	}{
		{
			left:     NewNumberFromInt(8),
			right:    NewNumberFromInt(2),
			expected: "4.0",
			round:    &tRound{"up", 1},
		},
		{
			left:     NewNumberFromInt(8),
			right:    NewNumberFromInt(2),
			expected: "4.00",
			round:    &tRound{"up", 2},
		},
		{
			left:     NewNumberFromInt(1),
			right:    NewNumberFromInt(7),
			expected: "0.1",
			round:    &tRound{"down", 1},
		},
		{
			left:     NewNumberFromInt(1),
			right:    NewNumberFromInt(7),
			expected: "0.2",
			round:    &tRound{"up", 1},
		},
		{
			left:     NewNumberFromInt(1),
			right:    NewNumberFromInt(7),
			expected: "0.14",
			round:    &tRound{"down", 2},
		},
		{
			left:     NewNumberFromInt(1),
			right:    NewNumberFromInt(7),
			expected: "0.15",
			round:    &tRound{"up", 2},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.691",
			round:    &tRound{"down", 3},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.691",
			round:    &tRound{"up", 3},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.6910",
			round:    &tRound{"down", 4},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.6911",
			round:    &tRound{"up", 4},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.69105",
			round:    &tRound{"down", 5},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(1.23),
			expected: "5.69106",
			round:    &tRound{"up", 5},
		},
		{
			left:     NewNumberFromFloat64(7.777),
			right:    NewNumberFromFloat64(1.6),
			expected: "4",
			round:    &tRound{"down", 0},
		},
		{
			left:     NewNumberFromFloat64(7.777),
			right:    NewNumberFromFloat64(1.6),
			expected: "5",
			round:    &tRound{"up", 0},
		},
		{
			left:     NewNumberFromFloat64(7.777),
			right:    NewNumberFromFloat64(1.6),
			expected: "4.8",
			round:    &tRound{"down", 1},
		},
		{
			left:     NewNumberFromFloat64(7.777),
			right:    NewNumberFromFloat64(1.6),
			expected: "4.9",
			round:    &tRound{"up", 1},
		},
		{
			left:     NewNumberFromFloat64(9.999999),
			right:    NewNumberFromInt(1),
			expected: "9",
			round:    &tRound{"down", 0},
		},
		{
			left:     NewNumberFromFloat64(9.999999),
			right:    NewNumberFromInt(1),
			expected: "10",
			round:    &tRound{"up", 0},
		},
		{
			left:     NewNumberFromInt(7),
			right:    NewNumberFromFloat64(2.22),
			expected: "3.1532",
			round:    &tRound{"half", 4},
		},
		{
			left:     NewNumberFromInt(-7),
			right:    NewNumberFromFloat64(2.22),
			expected: "-3.1532",
			round:    &tRound{"half", 4},
		},
	}
	for _, ex := range cases {
		actual := ex.left.Div(ex.right, ex.round.mode, ex.round.scale)
		assert.Equal(s.T(), ex.expected, actual.String(),
			"%s / %s round %s %d should equal %s",
			ex.left, ex.right, ex.round.mode, ex.round.scale, ex.expected)
	}
}

func (s *Zuite) TestValue_assignableTo() {
	cases := []struct {
		value Value
		typ   Type
	}{
		{vUndefined, &TextType{}},
		{vUndefined, &BoolType{}},
		{vUndefined, &NumberType{0}},
		{vUndefined, &NumberType{1}},

		{NewText(""), &TextType{}},
		{NewText("a"), &EnumType{"", map[string]bool{"a": true}}},

		{NewBool(true), &BoolType{}},

		{NewNumberFromInt(5), &NumberType{0}},
		{NewNumberFromFloat64(0.5), &NumberType{1}},
	}
	for _, ex := range cases {
		assert.True(s.T(), ex.value.assignableTo(ex.typ),
			"%s should be assignable to %s", ex.value, ex.typ)
	}
}

func (s *Zuite) TestValue_notAssignableTo() {
	cases := []struct {
		value Value
		typ   Type
	}{
		{NewText(""), &UndefinedType{}},
		{NewBool(true), &UndefinedType{}},
		{NewNumberFromInt(5), &UndefinedType{}},
		{NewNumberFromFloat64(0.5), &UndefinedType{}},

		{NewBool(true), &TextType{}},
		{NewNumberFromFloat64(0.5), &TextType{}},

		{NewText(""), &BoolType{}},
		{NewNumberFromFloat64(0.5), &BoolType{}},

		{NewText(""), &NumberType{1}},
		{NewNumberFromFloat64(0.55), &NumberType{1}},

		{NewNumberFromFloat64(5), &EnumType{"", map[string]bool{"a": true}}},
		{NewText("b"), &EnumType{"", map[string]bool{"a": true}}},
	}
	for _, ex := range cases {
		assert.False(s.T(), ex.value.assignableTo(ex.typ),
			"%s should not be assignable to %s", ex.value, ex.typ)
	}
}
