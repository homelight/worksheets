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

worksheet with_slice {
	42:names []text
}

worksheet with_refs {
	87:simple simple
}

worksheet with_refs_and_cycles {
	87:point_to_me with_refs_and_cycles
}`))

type Zuite struct {
	suite.Suite
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
