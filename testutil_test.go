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
var defs = MustNewDefinitions(strings.NewReader(`
worksheet simple {
	83:name text
	91:age  number[0]
}

worksheet all_types {
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

worksheet with_slice {
	42:names []text
}

worksheet with_slice_of_refs {
	42:many_simples []simple
}

worksheet with_refs {
	46:some_flag bool
	87:simple simple
}

worksheet with_refs_and_cycles {
	404:point_to_me with_refs_and_cycles
}`))

func forciblySetId(ws *Worksheet, id string) {
	ws.data[indexId] = NewText(id)
}

type Zuite struct {
	suite.Suite
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
