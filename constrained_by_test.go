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
	"math"
	"strconv"
	"strings"

	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestWorksheet_constrainedBy() {
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {
		1:name text constrained_by { return name == "Alex" || name == "Wilson" }
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	require.False(s.T(), ws.MustIsSet("name"))
	err = ws.Set("name", NewText("Alice"))
	require.Equal(s.T(), `"Alice" not a valid value for constrained field name`, err.Error())
	require.False(s.T(), ws.MustIsSet("name"))

	err = ws.Set("name", NewText("Alex"))
	require.NoError(s.T(), err)
	require.True(s.T(), ws.MustIsSet("name"))
	require.Equal(s.T(), `"Alex"`, ws.MustGet("name").String())
}

func (s *Zuite) TestWorksheet_constrainedByNonBoolExpression() {
	defs, err := NewDefinitions(strings.NewReader(`type constrained_non_bool_constrained_expression worksheet {
			69:some_field number[0] constrained_by { return some_field + 2 }
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("constrained_non_bool_constrained_expression")

	require.False(s.T(), ws.MustIsSet("some_field"))
	err = ws.Set("some_field", MustNewValue("7"))
	require.Equal(s.T(), "7 not a valid value for constrained field some_field", err.Error())
	require.False(s.T(), ws.MustIsSet("some_field"))
}

type perimeterAndAreaConstraints []string

var _ ComputedBy = perimeterAndAreaConstraints([]string{})

func (fn perimeterAndAreaConstraints) Args() []string {
	return fn
}

func (fn perimeterAndAreaConstraints) Compute(values ...Value) Value {
	var field_a, field_b, field_c int64
	switch t := values[0].(type) {
	case *Number:
		field_a = t.value
	case *Undefined:
		field_a = 0
	}
	switch t := values[1].(type) {
	case *Number:
		field_b = t.value
	case *Undefined:
		field_b = 0
	}
	switch t := values[2].(type) {
	case *Number:
		field_c = t.value
	case *Undefined:
		field_c = 0
	}
	withinLengthConstraint := 100-field_a-field_b-field_c > 0
	withinAreaConstraint := false

	switch t := values[3].(type) {
	case *Number:
		withinAreaConstraint = t.value >= 200
	case *Undefined:
		withinAreaConstraint = true
	}

	return NewBool(withinLengthConstraint && withinAreaConstraint)
}

type hypotenuse []string

var _ ComputedBy = hypotenuse([]string{})

func (fn hypotenuse) Args() []string {
	return fn
}

func (fn hypotenuse) Compute(values ...Value) Value {
	var field_a, field_b int64
	switch t := values[0].(type) {
	case *Number:
		field_a = t.value
	case *Undefined:
		field_a = 0
	}
	switch t := values[1].(type) {
	case *Number:
		field_b = t.value
	case *Undefined:
		field_b = 0
	}
	c := math.Sqrt(float64(field_a*field_a + field_b*field_b))
	hypotVal, _ := NewValue(strconv.FormatFloat(c, 'f', 0, 64))
	return hypotVal
}

type area []string

var _ ComputedBy = area([]string{})

func (fn area) Args() []string {
	return fn
}

func (fn area) Compute(values ...Value) Value {
	var field_a, field_b int64
	switch t := values[0].(type) {
	case *Number:
		field_a = t.value
	case *Undefined:
		return vUndefined
	}
	switch t := values[1].(type) {
	case *Number:
		field_b = t.value
	case *Undefined:
		return vUndefined
	}
	a := float64(field_a * field_b / 2)
	areaVal, _ := NewValue(strconv.FormatFloat(a, 'f', 0, 64))
	return areaVal
}

func (s *Zuite) TestWorksheet_constrainedByAndComputedBy() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"constrained_and_computed": map[string]ComputedBy{
				"field_a": perimeterAndAreaConstraints([]string{"field_a", "field_b", "field_c", "area"}),
				"field_b": perimeterAndAreaConstraints([]string{"field_a", "field_b", "field_c", "area"}),
				"field_c": hypotenuse([]string{"field_a", "field_b"}),
				"area":    area([]string{"field_a", "field_b"}),
			},
		},
	}
	// a worksheet to calculate a constrived math problem, where you have 100 ft of fence
	// and have to build a right triangle shaped enclosure that's at least 200 square feet in area.
	// The constrained fields check both area and perimeter
	defs, err := NewDefinitions(strings.NewReader(`type constrained_and_computed worksheet {
			50:field_a number[0] constrained_by { external }
			99:field_b number[0] constrained_by { external }
			12:field_c number[2] computed_by { external }
			30:area number[2] computed_by { external }
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("constrained_and_computed")

	require.False(s.T(), ws.MustIsSet("field_a"))
	err = ws.Set("field_a", MustNewValue("28"))
	require.NoError(s.T(), err)

	err = ws.Set("field_b", MustNewValue("90"))
	require.Equal(s.T(), "90 not a valid value for constrained field field_b", err.Error())

	err = ws.Set("field_b", MustNewValue("21"))
	require.NoError(s.T(), err)

	err = ws.Set("field_b", MustNewValue("1")) // too small in area
	require.Equal(s.T(), "1 not a valid value for constrained field field_b", err.Error())

	err = ws.Set("field_b", MustNewValue("50")) // too long of perimiter
	require.Equal(s.T(), "50 not a valid value for constrained field field_b", err.Error())
}
