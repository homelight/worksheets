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
	"fmt"
	"math"
	"strings"

	"github.com/stretchr/testify/require"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var (
	alice = NewText("Alice")
	bob   = NewText("Bob")
	carol = NewText("Carol")

	defsWithSlice = MustNewDefinitions(strings.NewReader(`
		worksheet with_slice {
			42:names []text
		}`))
)

func (s *Zuite) TestSliceExample() {
	ws := defsWithSlice.MustNewWorksheet("with_slice")

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

func (s *DbZuite) TestSliceSave() {
	ws := defsWithSlice.MustNewWorksheet("with_slice")
	ws.MustAppend("names", alice)

	// We're reaching into the data store to get the slice id in order to write
	// assertions against it.
	theSliceId := (ws.data[42].(*slice)).id

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, slicesRecs, sliceElementsRecs := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      ws.Id(),
			Version: 1,
			Name:    "with_slice",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: ws.Id(),
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
		},
		{
			WorksheetId: ws.Id(),
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: ws.Id(),
			Index:       42,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`[:%s`, theSliceId),
		},
	}, valuesRecs)

	require.Equal(s.T(), []rSlice{
		{
			Id:          theSliceId,
			WorksheetId: ws.Id(),
			Version:     1,
		},
	}, slicesRecs)

	require.Equal(s.T(), []rSliceElement{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       `"Alice"`,
		},
	}, sliceElementsRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

// impl notes:
// - when loading from DB, must order by index
// - test Get on slice type fails, even if it is undefined
// - test GetSlice on non-slice field fails
// - test Append on non-slice field fails
// - test append of non-assignable type e.g. putting a bool in []text
// - test deletes on out of bound indexes
// - denormalize things like max_index to speed up append and such
