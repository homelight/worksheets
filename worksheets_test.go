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

func (s *Zuite) TestExample() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {1:name text}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	isSet, err := ws.IsSet("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), false, isSet)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), true, isSet)

	name, err := ws.Get("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), `"Alice"`, name.String())

	err = ws.Unset("name")
	require.NoError(s.T(), err)

	isSet, err = ws.IsSet("name")
	require.NoError(s.T(), err)
	require.Equal(s.T(), false, isSet)
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

func (s *Zuite) TestExample_externalComputedBy() {
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
		2:hello_name text computed_by { external }
	}`))
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: missing plugin for simple.hello_name", err.Error())
	}
}

func (s *Zuite) TestExample_externalComputedBy2() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"not_so_simple": map[string]ComputedBy{},
		},
	}
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
	}`), opt)
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: unknown worksheet(not_so_simple)", err.Error())
	}
}

func (s *Zuite) TestExample_externalComputedBy3() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"unknown_name": nil,
			},
		},
	}
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
	}`), opt)
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: unknown field simple.unknown_name", err.Error())
	}
}

func (s *Zuite) TestExample_externalComputedBy4() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": nil,
			},
		},
	}
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text
	}`), opt)
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: field simple.name not externally defined", err.Error())
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

func (s *Zuite) TestExample_externalComputedBy4point5() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{}),
			},
		},
	}
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: simple.name plugin has no dependencies", err.Error())
	}
}

func (s *Zuite) TestExample_externalComputedBy5() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{"agee"}),
			},
		},
	}
	_, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "plugins: simple.name plugin has incorrect arg agee", err.Error())
	}
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
	require.Equal(s.T(), "Alice", ws.MustGet("name").String())
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
