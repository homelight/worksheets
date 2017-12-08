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

package main

import (
	"testing"

	"github.com/cucumber/gherkin-go"
	"github.com/stretchr/testify/require"

	"github.com/helloeave/worksheets"
)

func TestStep_definitions(t *testing.T) {
	runner := newRunner([]*gherkin.Step{
		{Text: "definitions example.ws"},
	})
	require.NoError(t, runner.run())

	require.NotNil(t, runner.defs)
}

func TestStep_instantiate(t *testing.T) {
	runner := newRunner([]*gherkin.Step{
		{Text: "definitions example.ws"},
		{Text: "foo = worksheet(simple)"},
	})
	require.NoError(t, runner.run())

	require.NotNil(t, runner.sheets["foo"])
}

func TestStep_instantiateWithTable(t *testing.T) {
	runner := newRunner([]*gherkin.Step{
		{Text: "definitions example.ws"},
		{
			Text: "foo = worksheet(simple)",
			Argument: newDataTable([][]string{
				{"age", "78"},
			}),
		},
	})
	require.NoError(t, runner.run())

	require.NotNil(t, runner.sheets["foo"])
	require.Equal(t, "78", runner.sheets["foo"].MustGet("age").String())
}

func newRunner(steps []*gherkin.Step) *runner {
	return &runner{
		currentDir: "../features",
		sheets:     make(map[string]*worksheets.Worksheet),
		steps:      steps,
	}
}

func newDataTable(data [][]string) *gherkin.DataTable {
	table := &gherkin.DataTable{Rows: make([]*gherkin.TableRow, 0)}
	for _, r := range data {
		row := &gherkin.TableRow{Cells: make([]*gherkin.TableCell, 0)}
		for _, c := range r {
			row.Cells = append(row.Cells, &gherkin.TableCell{Value: c})
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}
