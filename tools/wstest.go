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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cucumber/gherkin-go"

	"github.com/helloeave/worksheets"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wstest filename...")
		os.Exit(1)
	}

	var encounteredFailure bool
	for i, filename := range os.Args[1:] {
		if 0 < i {
			fmt.Println()
		}

		if ok := runFeature(filename); !ok {
			encounteredFailure = true
		}
	}

	if encounteredFailure {
		os.Exit(1)
	}
	os.Exit(0)
}

func runFeature(filename string) bool {
	file, err := os.Open(filepath.Join(filename))
	if err != nil {
		fmt.Printf("%s\n", filename)
		fmt.Printf("FAIL\t%s\n", err)
		return false
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	doc, err := gherkin.ParseGherkinDocument(reader)
	if err != nil {
		fmt.Printf("%s\n", filename)
		fmt.Printf("FAIL\t%s\n", err)
		return false
	}

	var (
		currentDir = filepath.Dir(filename)
		ok         = true
	)

	fmt.Printf("%s: %s\n", filename, doc.Feature.Name)
	for _, untypedChild := range doc.Feature.Children {
		switch child := untypedChild.(type) {
		case *gherkin.Scenario:
			runner := &runner{
				currentDir: currentDir,
				scenario:   child,
				sheets:     make(map[string]*worksheets.Worksheet),
			}
			err := runner.run()
			if err != nil {
				fmt.Printf("FAIL\t%s: %s\n", child.Name, err)
				ok = false
			} else {
				fmt.Printf("ok\t%s\n", child.Name)
			}
		default:
			panic(fmt.Sprintf("unknwon child type %T\n", child))
		}
	}

	return ok
}

type runner struct {
	currentDir string
	scenario   *gherkin.Scenario
	defs       *worksheets.Definitions
	sheets     map[string]*worksheets.Worksheet
}

func (r *runner) run() error {
onto_next_step:
	for _, step := range r.scenario.Steps {
		for re, fn := range stepFuncs {
			args := re.FindStringSubmatch(step.Text)
			if args == nil {
				continue
			}
			if err := fn(r, args[1:], step.Argument); err != nil {
				return err
			}
			continue onto_next_step
		}
		return fmt.Errorf("unknown step '%s'", step.Text)
	}
	return nil
}

var stepFuncs = map[*regexp.Regexp]func(*runner, []string, interface{}) error{
	re(`definitions (.*)`): func(r *runner, args []string, _ interface{}) error {
		if r.defs != nil {
			return fmt.Errorf("cannot provide multiple definitions files")
		}

		filename := args[0]

		defsFile, err := os.Open(filepath.Join(r.currentDir, filename))
		if err != nil {
			return err
		}
		defer defsFile.Close()

		reader := bufio.NewReader(defsFile)
		defs, err := worksheets.NewDefinitions(reader)
		if err != nil {
			return err
		}

		r.defs = defs

		return nil
	},

	re(`{name} = worksheet\({name}\)`): func(r *runner, args []string, extra interface{}) error {
		varname, name := args[0], args[1]

		var contents map[string]worksheets.Value
		if extra != nil {
			var err error
			contents, err = tableToContents(extra)
			if err != nil {
				return err
			}
		}

		if r.defs == nil {
			return fmt.Errorf("must first provide a definitions file: definitions {filename}")
		}

		ws, err := r.defs.NewWorksheet(name)
		if err != nil {
			return err
		}
		for key, value := range contents {
			if err := ws.Set(key, value); err != nil {
				return err
			}
		}

		r.sheets[varname] = ws

		return nil
	},

	re(`{name}\.{name} = {value}`): func(r *runner, args []string, _ interface{}) error {
		varname, key, v := args[0], args[1], args[2]

		ws, ok := r.sheets[varname]
		if !ok {
			return fmt.Errorf("unknown worksheet %s, did you instantiate it?", varname)
		}

		value, err := worksheets.NewValue(v)
		if err != nil {
			return err
		}

		if err := ws.Set(key, value); err != nil {
			return err
		}

		return nil
	},

	re(`{name}`): func(r *runner, args []string, extra interface{}) error {
		varname := args[0]

		ws, ok := r.sheets[varname]
		if !ok {
			return fmt.Errorf("unknown worksheet %s, did you instantiate it?", varname)
		}

		expected, err := tableToContents(extra)
		if err != nil {
			return err
		}

		for key, value := range expected {
			actual, err := ws.Get(key)
			if err != nil {
				return err
			} else if !value.Equals(actual) {
				return fmt.Errorf("%s: %s != %s", key, value, actual)
			}
		}

		return nil
	},
}

func re(s string) *regexp.Regexp {
	replacements := map[string]string{
		`{name}`:  `([a-z](?:[a-z_]*[a-z])?)`,
		`{value}`: `(.*)`,
		` `:       `\s+`,
	}
	for from, to := range replacements {
		s = strings.Replace(s, from, to, -1)
	}
	return regexp.MustCompile("^" + s + "$")
}

func tableToContents(extra interface{}) (map[string]worksheets.Value, error) {
	table, ok := extra.(*gherkin.DataTable)
	if !ok {
		return nil, fmt.Errorf("must provide a table")
	}

	contents := make(map[string]worksheets.Value)
	for _, row := range table.Rows {
		if len(row.Cells) != 2 {
			return nil, fmt.Errorf("must provide a table with two columns on every row")
		}
		key := row.Cells[0].Value
		value, err := worksheets.NewValue(row.Cells[1].Value)
		if err != nil {
			return nil, err
		}
		contents[key] = value
	}

	return contents, nil
}
