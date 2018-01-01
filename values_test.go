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
	cases := map[Value]string{
		&Undefined{}: "undefined",

		&Text{`Hello, "World"!`}: `"Hello, \"World\"!"`,

		&Bool{true}: "true",

		&Number{1, &tNumberType{0}}:     "1",
		&Number{10000, &tNumberType{4}}: "1.0000",
		&Number{123, &tNumberType{1}}:   "12.3",
		&Number{123, &tNumberType{2}}:   "1.23",
		&Number{123, &tNumberType{3}}:   "0.123",
		&Number{123, &tNumberType{4}}:   "0.0123",
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
		},
		{
			MustNewValue("1.0"),
			MustNewValue("1.0"),
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
