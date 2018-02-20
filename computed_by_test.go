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
	"fmt"
	"strconv"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestComputedBy_externalComputedBy() {
	cases := []struct {
		def         string
		opt         *Options
		expectedErr string
	}{
		{
			`worksheet simple {
				1:hello_name text computed_by { external }
			}`,
			nil,
			"simple.hello_name: missing plugin for external computed_by",
		},
		{
			`worksheet simple {}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"not_so_simple": map[string]ComputedBy{
						"unknown_name": nil,
					},
				},
			},
			"plugins: unknown worksheet(not_so_simple)",
		},
		{
			`worksheet simple {}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"unknown_name": nil,
					},
				},
			},
			"plugins: unknown field simple.unknown_name",
		},
		{
			`worksheet simple {
				1:name text
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": nil,
					},
				},
			},
			"plugins: field simple.name not externally defined",
		},
		{
			`worksheet simple {
				1:name text computed_by { external }
				2:age number[0]
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": sayAlice([]string{}),
					},
				},
			},
			"simple.name has no dependencies",
		},
		{
			`worksheet simple {
				1:name text computed_by { external }
				2:age number[0]
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": sayAlice([]string{"agee"}),
					},
				},
			},
			"simple.name references unknown arg agee",
		},
		{
			`worksheet parent {
				1:child child
				2:name text computed_by { external }
			}
			worksheet child {
				3:field text
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"parent": map[string]ComputedBy{
						"name": sayAlice([]string{"child.not_field"}),
					},
				},
			},
			"parent.name references unknown arg child.not_field",
		},
	}
	for _, ex := range cases {
		var opts []Options
		if ex.opt != nil {
			opts = append(opts, *ex.opt)
		}
		_, err := NewDefinitions(strings.NewReader(ex.def), opts...)
		if assert.Error(s.T(), err) {
			require.Equal(s.T(), ex.expectedErr, err.Error())
		}
	}

}

func (s *Zuite) TestComputedBy_externalComputedByPlugin() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{"age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	err = ws.Set("name", NewText("Alex"))
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "cannot assign to computed field name", err.Error())
	}
}

type sayAlice []string

var _ ComputedBy = sayAlice([]string{})

func (sa sayAlice) Args() []string {
	return sa
}

func (sa sayAlice) Compute(...Value) Value {
	return NewText("Alice")
}

type fullName []string

var _ ComputedBy = fullName([]string{})

func (fn fullName) Args() []string {
	return fn
}

func (fn fullName) Compute(values ...Value) Value {
	var firstName, lastName string
	switch t := values[0].(type) {
	case *Text:
		firstName = t.value
	case *Undefined:
		firstName = ""
	}
	switch t := values[1].(type) {
	case *Text:
		lastName = t.value
	case *Undefined:
		lastName = ""
	}
	return NewText(fmt.Sprintf("%s %s", firstName, lastName))
}

type age []string

var _ ComputedBy = age([]string{})

func (fn age) Args() []string {
	return fn
}

func (fn age) Compute(values ...Value) Value {
	// TODO(pascal): we need to figure out how to make Values useful, e.g. having an AsString()
	birthYear := values[0].(*Number).value
	value, _ := NewValue(strconv.FormatInt(2018-birthYear, 10))
	return value
}

type bio []string

var _ ComputedBy = bio([]string{})

func (fn bio) Args() []string {
	return fn
}

func (fn bio) Compute(values ...Value) Value {
	var fullName string
	var birthYear, age int64
	switch t := values[0].(type) {
	case *Text:
		fullName = t.value
	case *Undefined:
		fullName = ""
	}
	switch t := values[1].(type) {
	case *Number:
		birthYear = t.value
	case *Undefined:
		birthYear = 0
	}
	switch t := values[2].(type) {
	case *Number:
		age = t.value
	case *Undefined:
		age = 0
	}

	return NewText(fmt.Sprintf("%s, age %d, born in %d", fullName, age, birthYear))
}

func (s *Zuite) TestComputedBy_externalGood() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{"age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	require.False(s.T(), ws.MustIsSet("name"))

	ws.MustSet("age", MustNewValue("73"))
	require.Equal(s.T(), `"Alice"`, ws.MustGet("name").String())
}

func (s *Zuite) TestComputedBy_externalGoodComplicated() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"complicated": map[string]ComputedBy{
				"full_name": fullName([]string{"first_name", "last_name"}),
				"age":       age([]string{"birth_year"}),
				"bio":       bio([]string{"full_name", "birth_year", "age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`worksheet complicated {
		1:first_name text
		2:last_name text
		3:full_name text computed_by { external }
		4:birth_year number[0]
		5:age number[0] computed_by { external }
		6:bio text computed_by { external }
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("complicated")

	require.False(s.T(), ws.MustIsSet("full_name"))

	ws.MustSet("first_name", NewText("Alice"))
	ws.MustSet("last_name", NewText("Maters"))
	ws.MustSet("birth_year", MustNewValue("1945"))
	require.Equal(s.T(), `"Alice Maters"`, ws.MustGet("full_name").String())
	require.Equal(s.T(), `73`, ws.MustGet("age").String())
	require.Equal(s.T(), `"Alice Maters, age 73, born in 1945"`, ws.MustGet("bio").String())
}

func (s *Zuite) TestComputedBy_simpleExpressionsInWorksheet() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:age number[0]
		2:age_plus_two number[0] computed_by { return age + 2 }
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	ws.MustSet("age", MustNewValue("73"))
	require.Equal(s.T(), "75", ws.MustGet("age_plus_two").String())
}

func (s *Zuite) TestComputedBy_cyclicEditsIfNoIdentCheck() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet cyclic_edits {
		1:right bool
		2:a bool computed_by {
			return b || right
		}
		3:b bool computed_by {
			return a || !right
		}
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("cyclic_edits")

	ws.MustSet("right", MustNewValue("true"))
	require.Equal(s.T(), "true", ws.MustGet("right").String(), "right")
	require.Equal(s.T(), "undefined", ws.MustGet("a").String(), "a")
	require.Equal(s.T(), "undefined", ws.MustGet("b").String(), "b")
}

var defsCrossWs = MustNewDefinitions(strings.NewReader(`
worksheet parent {
	1:child_amount number[2] computed_by {
		return child.amount
	}
	2:child child
}

worksheet child {
	5:amount number[2]
}`))

func (s *Zuite) TestComputedBy_simpleCrossWsParentPointers() {
	parent := defsCrossWs.MustNewWorksheet("parent")
	child := defsCrossWs.MustNewWorksheet("child")
	forciblySetId(parent, "parent-id")

	require.Len(s.T(), child.parents, 0)
	require.Len(s.T(), child.parents["parent"], 0)
	require.Len(s.T(), child.parents["parent"][2], 0)

	parent.MustSet("child", child)
	require.Len(s.T(), child.parents, 1)
	require.Len(s.T(), child.parents["parent"], 1)
	require.Len(s.T(), child.parents["parent"][2], 1)
	require.True(s.T(), child.parents["parent"][2]["parent-id"] == parent)

	parent.MustUnset("child")
	require.Len(s.T(), child.parents, 1)
	require.Len(s.T(), child.parents["parent"], 1)
	require.Len(s.T(), child.parents["parent"][2], 0)
}

func (s *Zuite) TestComputedBy_simpleCrossWsExample() {
	parent := defsCrossWs.MustNewWorksheet("parent")

	child := defsCrossWs.MustNewWorksheet("child")
	child.MustSet("amount", MustNewValue("1.11"))
	parent.MustSet("child", child)
	require.Equal(s.T(), "1.11", parent.MustGet("child_amount").String())

	child.MustSet("amount", MustNewValue("2.22"))
	require.Equal(s.T(), "2.22", parent.MustGet("child_amount").String())

	parent.MustUnset("child")
	require.Equal(s.T(), "undefined", parent.MustGet("child_amount").String())
}

type sumPlugin string

// Assert that sumPlugin implements the ComputedBy interface.
var _ ComputedBy = sumPlugin("")

func (p sumPlugin) Args() []string {
	return []string{string(p)}
}

func (p sumPlugin) Compute(values ...Value) Value {
	slice := values[0].(*Slice)
	sum := MustNewValue("0").(*Number)
	for _, elem := range slice.Elements() {
		if num, ok := elem.(*Number); ok {
			sum = sum.Plus(num)
		} else {
			return &Undefined{}
		}
	}
	return sum
}

var defsCrossWsThroughSlice = MustNewDefinitions(strings.NewReader(`
worksheet parent {
	1:sum_child_amount number[2] computed_by {
		external
	}
	2:children []child
}

worksheet child {
	5:amount number[2]
}`), Options{
	Plugins: map[string]map[string]ComputedBy{
		"parent": {
			"sum_child_amount": sumPlugin("children.amount"),
		},
	},
})

func (s *Zuite) TestComputedBy_crossWsThroughSliceParentPointers() {
	parent := defsCrossWsThroughSlice.MustNewWorksheet("parent")
	child1 := defsCrossWsThroughSlice.MustNewWorksheet("child")
	child2 := defsCrossWsThroughSlice.MustNewWorksheet("child")
	forciblySetId(parent, "parent-id")

	require.Len(s.T(), child1.parents, 0)
	require.Len(s.T(), child1.parents["parent"], 0)
	require.Len(s.T(), child1.parents["parent"][2], 0)
	require.Len(s.T(), child2.parents, 0)
	require.Len(s.T(), child2.parents["parent"], 0)
	require.Len(s.T(), child2.parents["parent"][2], 0)

	parent.MustAppend("children", child1)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][2], 1)
	require.True(s.T(), child1.parents["parent"][2]["parent-id"] == parent)
	require.Len(s.T(), child2.parents, 0)
	require.Len(s.T(), child2.parents["parent"], 0)
	require.Len(s.T(), child2.parents["parent"][2], 0)

	parent.MustAppend("children", child2)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][2], 1)
	require.True(s.T(), child1.parents["parent"][2]["parent-id"] == parent)
	require.Len(s.T(), child2.parents, 1)
	require.Len(s.T(), child2.parents["parent"], 1)
	require.Len(s.T(), child2.parents["parent"][2], 1)
	require.True(s.T(), child2.parents["parent"][2]["parent-id"] == parent)

	parent.Del("children", 0)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][2], 0)
	require.Len(s.T(), child2.parents, 1)
	require.Len(s.T(), child2.parents["parent"], 1)
	require.Len(s.T(), child2.parents["parent"][2], 1)
	require.True(s.T(), child2.parents["parent"][2]["parent-id"] == parent)

	parent.Del("children", 0)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][2], 0)
	require.Len(s.T(), child2.parents, 1)
	require.Len(s.T(), child2.parents["parent"], 1)
	require.Len(s.T(), child2.parents["parent"][2], 0)
}

func (s *Zuite) TestComputedBy_crossWsThroughSliceExample() {
	parent := defsCrossWsThroughSlice.MustNewWorksheet("parent")

	require.Equal(s.T(), "undefined", parent.MustGet("sum_child_amount").String())

	child1 := defsCrossWsThroughSlice.MustNewWorksheet("child")
	child1.MustSet("amount", MustNewValue("1.11"))
	parent.MustAppend("children", child1)
	require.Equal(s.T(), "1.11", parent.MustGet("sum_child_amount").String())

	child2 := defsCrossWsThroughSlice.MustNewWorksheet("child")
	child2.MustSet("amount", MustNewValue("2.22"))
	parent.MustAppend("children", child2)
	require.Equal(s.T(), "3.33", parent.MustGet("sum_child_amount").String())

	parent.Del("children", 0)
	require.Equal(s.T(), "2.22", parent.MustGet("sum_child_amount").String())

	parent.Del("children", 0)
	require.Equal(s.T(), "0", parent.MustGet("sum_child_amount").String())
}
