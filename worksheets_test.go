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

func (s *Zuite) TestParser_parseWorksheet() {
	cases := map[string]func(*tWorksheet){
		`worksheet simple {}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 0, len(ws.fields))
		},
		`worksheet simple {42:full_name text}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 1, len(ws.fields))

			field := ws.fields[0]
			require.Equal(s.T(), 42, field.index)
			require.Equal(s.T(), "full_name", field.name)
			require.Equal(s.T(), "text", field.typ)
		},
		`  worksheet simple {42:full_name text 45:happy bool}`: func(ws *tWorksheet) {
			require.Equal(s.T(), "simple", ws.name)
			require.Equal(s.T(), 2, len(ws.fields))

			field1 := ws.fields[0]
			require.Equal(s.T(), 42, field1.index)
			require.Equal(s.T(), "full_name", field1.name)
			require.Equal(s.T(), "text", field1.typ)

			field2 := ws.fields[1]
			require.Equal(s.T(), 45, field2.index)
			require.Equal(s.T(), "happy", field2.name)
			require.Equal(s.T(), "bool", field2.typ)
		},
	}
	for input, checks := range cases {
		p := newParser(strings.NewReader(input))
		ws, err := p.parseWorksheet()
		require.NoError(s.T(), err)
		checks(ws)
	}
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
