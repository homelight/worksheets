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

func (s *Zuite) TestExample() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	isSet, err := ws.IsSet("name")
	require.NoError(s.T(), err)
	require.False(s.T(), isSet)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.True(s.T(), isSet)

	name, err := ws.Get("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), `"Alice"`, name.String())

	err = ws.Unset("name")
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.False(s.T(), isSet)
}

func (s *Zuite) TestWorksheetNew_zeroDefs() {
	wsDefs := []string{
		``,
		` `,
		`some text`,
		`not a worksheet`,
		`work sheet`,
	}
	for _, def := range wsDefs {
		_, err := NewDefinitions(strings.NewReader(def))
		if assert.Error(s.T(), err) {
			require.Equal(s.T(), "no worksheets defined", err.Error())
		}
	}
}

func (s *Zuite) TestWorksheetNew_multipleDefs() {
	wsDefs := `worksheet one {1:name text} worksheet two {1:occupation text}`
	defs, err := NewDefinitions(strings.NewReader(wsDefs))
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(defs.defs))

	for _, wsName := range []string{"one", "two"} {
		_, ok := defs.defs[wsName]
		require.True(s.T(), ok)
	}
}

func (s *Zuite) TestWorksheetNew_multipleDefsSameName() {
	wsDefs := `worksheet simple {1:name text} worksheet simple {1:occupation text}`
	_, err := NewDefinitions(strings.NewReader(wsDefs))
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "multiple worksheets with name simple", err.Error())
	}
}

func (s *Zuite) TestWorksheet_externalComputedBy() {
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
			"plugins: missing plugin for simple.hello_name",
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
			"plugins: simple.name plugin has no dependencies",
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
			"plugins: simple.name plugin has incorrect arg agee",
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

func (s *Zuite) TestWorksheet_externalComputedByPlugin() {
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
	case *tText:
		firstName = t.value
	case *tUndefined:
		firstName = ""
	}
	switch t := values[1].(type) {
	case *tText:
		lastName = t.value
	case *tUndefined:
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
	birthYear := values[0].(*tNumber).value
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
	case *tText:
		fullName = t.value
	case *tUndefined:
		fullName = ""
	}
	switch t := values[1].(type) {
	case *tNumber:
		birthYear = t.value
	case *tUndefined:
		birthYear = 0
	}
	switch t := values[2].(type) {
	case *tNumber:
		age = t.value
	case *tUndefined:
		age = 0
	}

	return NewText(fmt.Sprintf("%s, age %d, born in %d", fullName, age, birthYear))
}

func (s *Zuite) TestExternalComputedBy_good() {
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

func (s *Zuite) TestExternalComputedBy_goodComplicated() {
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

func (s *Zuite) TestWorksheetNew_origEmpty() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// We need to ensure orig is empty since this is a fresh worksheet, and
	// even the special values (e.g. version, id) must be taken into
	// consideration upon save.
	require.Empty(s.T(), ws.orig)
}

func (s *Zuite) TestWorksheetGet_undefinedIfNoValue() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	value := ws.MustGet("name")
	require.Equal(s.T(), "undefined", value.String())
}

func (s *Zuite) TestWorksheet_idAndVersion() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	id, err := ws.Get("id")
	require.NoError(s.T(), err)
	require.Equal(s.T(), 36+2, len(id.String()))

	version, err := ws.Get("version")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "1", version.String())
}

func (s *Zuite) TestWorksheet_diff() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws, err := defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	// initial diff
	require.Equal(s.T(), map[int]Value{
		IndexId:      NewText(ws.Id()),
		IndexVersion: MustNewValue("1"),
	}, ws.diff())

	// set name to Alice
	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	// now, also expecting Alice
	require.Equal(s.T(), map[int]Value{
		IndexId:      NewText(ws.Id()),
		IndexVersion: MustNewValue("1"),
		1:            NewText("Alice"),
	}, ws.diff())

	// Alice is now Bob
	err = ws.Set("name", NewText("Bob"))
	require.NoError(s.T(), err)

	require.Equal(s.T(), map[int]Value{
		IndexId:      NewText(ws.Id()),
		IndexVersion: MustNewValue("1"),
		1:            NewText("Bob"),
	}, ws.diff())

	// let's fake Bob being there before, and not anymore
	ws.orig[1] = ws.data[1]
	err = ws.Unset("name")
	require.NoError(s.T(), err)

	// now, name should go to an explicit undefine
	require.Equal(s.T(), map[int]Value{
		IndexId:      NewText(ws.Id()),
		IndexVersion: MustNewValue("1"),
		1:            MustNewValue("undefined"),
	}, ws.diff())
}
