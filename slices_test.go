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

	"github.com/stretchr/testify/require"
)

var (
	alice = NewText("Alice")
	bob   = NewText("Bob")
	carol = NewText("Carol")
)

func (s *Zuite) TestSliceExample() {
	defs, err := NewDefinitions(strings.NewReader(`
		worksheet with_slice {
			1:names []text
		}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("with_slice")

	require.Len(s.T(), ws.MustGetSlice("names"), 0)

	ws.MustAppend("names", alice)
	require.Equal(s.T(), []Value{alice}, ws.MustGetSlice("names"))

	ws.MustAppend("names", bob)
	require.Equal(s.T(), []Value{alice, bob}, ws.MustGetSlice("names"))

	ws.MustAppend("names", carol)
	require.Equal(s.T(), []Value{alice, bob, carol}, ws.MustGetSlice("names"))

	ws.MustDel("names", 1)
	require.Equal(s.T(), []Value{alice, carol}, ws.MustGetSlice("names"))

	ws.MustDel("names", 1)
	require.Equal(s.T(), []Value{alice}, ws.MustGetSlice("names"))

	ws.MustDel("names", 0)
	require.Len(s.T(), ws.MustGetSlice("names"), 0)
}

// impl notes:
// - when loading from DB, must order by index
// - test Get on slice type fails, even if it is undefined
// - test GetSlice on non-slice field fails
// - test Append on non-slice field fails
// - test append of non-assignable type e.g. putting a bool in []text
// - test deletes on out of bound indexes
