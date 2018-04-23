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

package wstesting

import (
	"strings"
	"testing"

	"github.com/cucumber/gherkin-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/helloeave/worksheets"
)

type Zuite struct {
	suite.Suite
}

func (s *Zuite) TestStepToCommand() {
	cases := []struct {
		step     *gherkin.Step
		expected command
	}{
		// load
		{
			step(`load "some_file.ws"`),
			cLoad{
				filename: "some_file.ws",
			},
		},

		// create
		{
			step(`create some_ws "some_name"`),
			cCreate{
				ws:   "some_ws",
				name: "some_name",
			},
		},

		// set
		{
			step(`set some_ws.some_field 5`),
			cSet{
				ws: "some_ws",
				values: map[string]worksheets.Value{
					"some_field": worksheets.NewNumberFromInt(5),
				},
			},
		},
		{
			step(`set some_ws`,
				[]string{"some_field1", "true"},
				[]string{"some_field2", `"hello"`},
				[]string{"some_field3", "9.99"},
			),
			cSet{
				ws: "some_ws",
				values: map[string]worksheets.Value{
					"some_field1": worksheets.NewBool(true),
					"some_field2": worksheets.NewText("hello"),
					"some_field3": worksheets.NewNumberFromFloat64(9.99),
				},
			},
		},

		// unset
		{
			step(`unset some_ws.some_field`),
			cSet{
				ws: "some_ws",
				values: map[string]worksheets.Value{
					"some_field": worksheets.NewUndefined(),
				},
			},
		},
		{
			step(`unset some_ws`,
				[]string{"some_field1"},
				[]string{"some_field2"},
				[]string{"some_field3"},
			),
			cSet{
				ws: "some_ws",
				values: map[string]worksheets.Value{
					"some_field1": worksheets.NewUndefined(),
					"some_field2": worksheets.NewUndefined(),
					"some_field3": worksheets.NewUndefined(),
				},
			},
		},

		// append
		{
			step(`append some_ws.some_field 666`),
			cAppend{
				ws:    "some_ws",
				field: "some_field",
				values: []worksheets.Value{
					worksheets.NewNumberFromInt(666),
				},
			},
		},
		{
			step(`append some_ws.some_field`,
				[]string{"1"},
				[]string{"2"},
				[]string{"3"},
			),
			cAppend{
				ws:    "some_ws",
				field: "some_field",
				values: []worksheets.Value{
					worksheets.NewNumberFromInt(1),
					worksheets.NewNumberFromInt(2),
					worksheets.NewNumberFromInt(3),
				},
			},
		},

		// del
		{
			step(`del some_ws.some_field 78`),
			cDel{
				ws:    "some_ws",
				field: "some_field",
				indexes: []int{
					78,
				},
			},
		},
		{
			step(`del some_ws.some_field`,
				[]string{"25"},
				[]string{"67"},
			),
			cDel{
				ws:    "some_ws",
				field: "some_field",
				indexes: []int{
					25,
					67,
				},
			},
		},

		// assert
		{
			step(`assert some_ws.some_field 6`),
			cAssert{
				ws:      "some_ws",
				partial: true,
				expected: map[string]worksheets.Value{
					"some_field": worksheets.NewNumberFromInt(6),
				},
			},
		},
		{
			step(`assert some_ws`,
				[]string{"some_field1", "true"},
				[]string{"some_field2", `"hello"`},
				[]string{"some_field3", "9.99"},
			),
			cAssert{
				ws:      "some_ws",
				partial: false,
				expected: map[string]worksheets.Value{
					"some_field1": worksheets.NewBool(true),
					"some_field2": worksheets.NewText("hello"),
					"some_field3": worksheets.NewNumberFromFloat64(9.99),
				},
			},
		},
		{
			step(`assert some_ws`,
				[]string{"some_field1", "true"},
				[]string{"some_field2", `"hello"`},
				[]string{"-", ""},
			),
			cAssert{
				ws:      "some_ws",
				partial: true,
				expected: map[string]worksheets.Value{
					"some_field1": worksheets.NewBool(true),
					"some_field2": worksheets.NewText("hello"),
				},
			},
		},
	}
	for _, ex := range cases {
		actual, err := stepToCommand(ex.step)
		if assert.NoError(s.T(), err, ex.step.Text) {
			assert.Equal(s.T(), ex.expected, actual, ex.step.Text)
		}
	}
}

func (s *Zuite) TestStepToCommand_errors() {
	cases := []struct {
		step        *gherkin.Step
		expectedErr string
	}{
		// misc
		{
			step(``),
			"no verb: expecting verb load, create, set, unset, append, del, or assert",
		},
		{
			step(`foo`),
			"wrong verb 'foo': expecting verb load, create, set, unset, append, del, or assert",
		},

		// load
		{
			step(`load too many`),
			`load too many: expecting load "<filename>"`,
		},
		{
			step(`load not_quoted`),
			`load not_quoted: expecting quoted filename, e.g. "my_definitions.ws"`,
		},

		// create
		{
			step(`create too_little`),
			`create too_little: expecting create <ws> "<name>"`,
		},
		{
			step(`create too many here`),
			`create too many here: expecting create <ws> "<name>"`,
		},
		{
			step(`create ws not_quoted`),
			`create ws not_quoted: expecting quoted name, e.g. "my_name"`,
		},

		// set
		{
			step(`set`),
			`set: expecting <ws> with data table or <ws.field> with value`,
		},
		{
			step(`set some_ws`),
			`set some_ws: must provide a data table`,
		},
		{
			step(`set some_ws.some_field`),
			`set some_ws.some_field: missing value`,
		},
		{
			step(`set some_ws 6`),
			`set some_ws 6: expecting <ws>.<field>`,
		},
		{
			step(`set some_ws.some_field bad`),
			`set some_ws.some_field bad: unknown literal, found bad`,
		},
		{
			step(`set some_ws`,
				[]string{"some_field", "bad"},
			),
			`set some_ws: unknown literal, found bad`,
		},
		{
			step(`set some_ws`,
				[]string{"some_field", "5"},
				[]string{"-", ""},
			),
			`set some_ws: partial not allowed`,
		},
		{
			step(`set some_ws.some_field 5 too_many`),
			`set some_ws.some_field 5 too_many: expecting <ws> with data table or <ws.field> with value`,
		},

		// unset
		{
			step(`unset`),
			`unset: expecting <ws> with field table or <ws.field>`,
		},
		{
			step(`unset some_ws`),
			`unset some_ws: expecting <ws>.<field>`,
		},
		{
			step(`unset some_ws.some_field 5 too_many`),
			`unset some_ws.some_field 5 too_many: expecting <ws> with field table or <ws.field>`,
		},

		// append
		{
			step(`append`),
			`append: expecting <ws>.<field> with value or value table`,
		},
		{
			step(`append some_ws`),
			`append some_ws: expecting <ws>.<field>`,
		},
		{
			step(`append some_ws.some_field`),
			`append some_ws.some_field: must provide a value table`,
		},
		{
			step(`append some_ws.some_field bad`),
			`append some_ws.some_field bad: unknown literal, found bad`,
		},
		{
			step(`append some_ws.some_field`,
				[]string{"bad"},
			),
			`append some_ws.some_field: unknown literal, found bad`,
		},
		{
			step(`append some_ws.some_field 5 too_many`),
			`append some_ws.some_field 5 too_many: expecting <ws>.<field> with value or value table`,
		},

		// del
		{
			step(`del`),
			`del: expecting <ws>.<field> with index or index table`,
		},
		{
			step(`del ws`),
			`del ws: expecting <ws>.<field>`,
		},
		{
			step(`del ws.field`,
				[]string{"bad"},
			),
			`del ws.field: unreadable index bad`,
		},
		{
			step(`del ws.field`),
			`del ws.field: must provide an index table`,
		},
		{
			step(`del ws.field bad`),
			`del ws.field bad: unreadable index bad`,
		},

		// assert
		{
			step(`assert`),
			`assert: expecting <ws> with data table or <ws.field> with value`,
		},
		{
			step(`assert ws`),
			`assert ws: must provide a data table`,
		},
		{
			step(`assert ws.field`),
			`assert ws.field: missing value`,
		},
		{
			step(`assert ws.field bad`),
			`assert ws.field bad: unknown literal, found bad`,
		},
		{
			step(`assert too many here`),
			`assert too many here: expecting <ws> with data table or <ws.field> with value`,
		},
	}
	for _, ex := range cases {
		_, err := stepToCommand(ex.step)
		assert.EqualError(s.T(), err, ex.expectedErr, ex.step.Text)
	}
}

func (s *Zuite) TestDocToScenarios() {
	cases := []struct {
		doc      string
		expected []Scenario
	}{
		// simple
		{
			doc: `
Feature: something
Scenario: the_name_here
	Given load "the_filename.ws"
`,
			expected: []Scenario{
				{
					Name: "the_name_here",
					commands: []command{
						cLoad{"the_filename.ws"},
					},
				},
			},
		},

		// with bg
		{
			doc: `
Feature: something
Background:
	Given load "the_filename.ws"
Scenario: the_name_here
	Then create ws "some_ws_name"
`,
			expected: []Scenario{
				{
					Name: "the_name_here",
					commands: []command{
						cLoad{"the_filename.ws"},
						cCreate{"ws", "some_ws_name"},
					},
				},
			},
		},

		// bg is for all scenarios
		{
			doc: `
Feature: something
Background:
	Given load "the_filename.ws"
Scenario: the_name_here_1
	Then create ws1 "some_ws_name"
Scenario: the_name_here_2
	Then create ws2 "some_other_ws_name"
`,
			expected: []Scenario{
				{
					Name: "the_name_here_1",
					commands: []command{
						cLoad{"the_filename.ws"},
						cCreate{"ws1", "some_ws_name"},
					},
				},
				{
					Name: "the_name_here_2",
					commands: []command{
						cLoad{"the_filename.ws"},
						cCreate{"ws2", "some_other_ws_name"},
					},
				},
			},
		},
	}
	for _, ex := range cases {
		doc, err := gherkin.ParseGherkinDocument(strings.NewReader(ex.doc))
		require.NoError(s.T(), err)

		actual, err := docToScenarios(doc)
		if assert.NoError(s.T(), err, ex.doc) {
			for i := range actual {
				assert.Len(s.T(), actual[i].steps, len(actual[i].commands))
				actual[i].steps = nil
			}
			assert.Equal(s.T(), ex.expected, actual, ex.doc)
		}
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}

func step(text string, data ...[]string) *gherkin.Step {
	step := gherkin.Step{
		Text: text,
	}
	if len(data) != 0 {
		table := &gherkin.DataTable{Rows: make([]*gherkin.TableRow, 0)}
		for _, r := range data {
			row := &gherkin.TableRow{Cells: make([]*gherkin.TableCell, 0)}
			for _, c := range r {
				row.Cells = append(row.Cells, &gherkin.TableCell{Value: c})
			}
			table.Rows = append(table.Rows, row)
		}
		step.Argument = table
	}
	return &step
}
