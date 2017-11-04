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

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Zuite struct {
	suite.Suite
}

func (s *Zuite) TestParser_Worksheet1() {
	input := `worksheet simple {42:full_name text}`
	p := newParser(strings.NewReader(input))

	ws, err := p.parseWorksheet()
	require.NoError(s.T(), err)
	require.Equal(s.T(), "simple", ws.name)
	require.Equal(s.T(), 1, len(ws.fields))
	field := ws.fields[0]
	require.Equal(s.T(), 42, field.index)
	require.Equal(s.T(), "full_name", field.name)
	require.Equal(s.T(), "text", field.typ)
}

func (s *Zuite) TestTokenizer_Simple() {
	input := `worksheet simple {1:full_name text}`
	p := newParser(strings.NewReader(input))

	toks := []string{
		"worksheet",
		"simple",
		"{",
		"1",
		":",
		"full_name",
		"text",
		"}",
	}
	for _, tok := range toks {
		require.Equal(s.T(), tok, p.next())
	}
	require.Equal(s.T(), "", p.next())
	require.Equal(s.T(), "", p.next())
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
