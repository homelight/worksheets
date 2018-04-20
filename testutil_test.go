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
	"testing"

	"github.com/stretchr/testify/suite"
)

// some useful values
var (
	alice = NewText("Alice")
	bob   = NewText("Bob")
	carol = NewText("Carol")
)

// definitions
var defs = `
type simple worksheet {
	83:name text
	91:age  number[0]
}

type all_types worksheet {
	 1:text      text
	 2:bool      bool
	 3:num_0     number[0]
	 4:num_2     number[2]
	 5:undefined undefined
	 6:ws        all_types
	 7:slice_t   []text
	 11:slice_b  []bool
	 12:slice_bu []bool
	 9:slice_n0  []number[0]
	10:slice_n2  []number[2]
	13:slice_nu  []number[0]
	 8:slice_ws  []all_types
}

type with_slice worksheet {
	42:names []text
}

type with_slice_of_refs worksheet {
	42:many_simples []simple
}

type with_refs worksheet {
	46:some_flag bool
	87:simple simple
}

type with_refs_and_cycles worksheet {
	404:point_to_me with_refs_and_cycles
}`

func forciblySetId(ws *Worksheet, id string) {
	ws.data[indexId] = NewText(id)
}

type allDefs struct {
	defs                    *Definitions
	cloneDefs               *Definitions
	defsForSelectors        *Definitions
	defsCrossWs             *Definitions
	defsCrossWsThroughSlice *Definitions
}

func newAllDefs() allDefs {
	var s allDefs

	// When initializing, we purposefully ignore errors to make it easier to work
	// on specific parts of the parser by running single tests:
	// - If we're running a single test which does not depend on these
	//   definitions, we shouldn't fail early, so as to provide feedback to the
	//   programmer on the test being ran (rather than whether full parsing works).
	// - And since the suite itself will fail if any of these are nil, we are not
	//   changing the test suite outcome by ignoring errors, simply shifting where
	//   and how these errors are reported.
	s.defs, _ = NewDefinitions(strings.NewReader(defs))
	s.cloneDefs, _ = NewDefinitions(strings.NewReader(cloneDefs))
	s.defsForSelectors, _ = NewDefinitions(strings.NewReader(defsForSelectors))
	s.defsCrossWs, _ = NewDefinitions(strings.NewReader(defsCrossWs))
	s.defsCrossWsThroughSlice, _ = NewDefinitions(strings.NewReader(defsCrossWsThroughSlice), defsCrossWsThroughSliceOptions)

	return s
}

type Zuite struct {
	suite.Suite
	allDefs
}

func (s *Zuite) SetupSuite() {
	s.allDefs = newAllDefs()
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
