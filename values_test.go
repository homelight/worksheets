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
		&tUndefined{}: "undefined",

		&tText{`Hello, "World"!`}: `"Hello, \"World\"!"`,

		&tBool{true}: "true",

		&tNumber{1, &tNumberType{0}}:     "1",
		&tNumber{10000, &tNumberType{4}}: "1.0000",
		&tNumber{123, &tNumberType{1}}:   "12.3",
		&tNumber{123, &tNumberType{2}}:   "1.23",
		&tNumber{123, &tNumberType{3}}:   "0.123",
		&tNumber{123, &tNumberType{4}}:   "0.0123",
	}
	for value, expected := range cases {
		assert.Equal(s.T(), expected, value.String())
	}
}

func (s *Zuite) TestValueEquals() {
	// a.k.a. congruence classes
	buckets := [][]Value{
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
				assert.True(s.T(), this.Equals(that), "%s == %s", this, that)
			}
		}
	}

	// across buckets, all values must not be equal
	// TODO(pascal): todo
}
