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
	"github.com/stretchr/testify/require"
)

var enumsDefs = `
type team_member enum {
	"pratik",
	"jane",
	"alex",
	"the_devil",
}

type yes_or_no enum {
	"yes",
	"no",
}

type questionnaire worksheet {
	1:who  team_member
	2:whos []team_member
	3:is_a_hotdog_a_sandwich yes_or_no computed_by {
		return first_of(
			if(who == "pratik",    "yes" ),
			if(who == "jane",      "no"  ),
			if(who == "the_devil", "42!" ),
		)
	}
}`

func (s *Zuite) TestEnum_setAndAppend() {
	ws := s.enumsDefs.MustNewWorksheet("questionnaire")

	var err error

	// set

	err = ws.Set("who", NewText("pratik"))
	require.NoError(s.T(), err)

	err = ws.Set("who", NewText("clara"))
	require.EqualError(s.T(), err, "cannot assign clara to team_member")

	// append

	err = ws.Append("whos", NewText("pratik"))
	require.NoError(s.T(), err)

	err = ws.Append("whos", NewText("clara"))
	require.EqualError(s.T(), err, "cannot append clara to []team_member")
}

func (s *Zuite) TestEnum_hotdogConundrum() {
	ws := s.enumsDefs.MustNewWorksheet("questionnaire")

	ws.MustSet("who", NewText("pratik"))
	require.Equal(s.T(), `"yes"`, ws.MustGet("is_a_hotdog_a_sandwich").String())

	ws.MustSet("who", NewText("jane"))
	require.Equal(s.T(), `"no"`, ws.MustGet("is_a_hotdog_a_sandwich").String())

	ws.MustSet("who", NewText("alex"))
	require.Equal(s.T(), `undefined`, ws.MustGet("is_a_hotdog_a_sandwich").String())

	err := ws.Set("who", NewText("the_devil"))
	require.EqualError(s.T(), err, "cannot assign 42! to yes_or_no")
}
