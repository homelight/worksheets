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

	"github.com/stretchr/testify/require"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

func (s *Zuite) TestSliceExample() {
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

func (s *Zuite) TestSliceOps() {
	slice1 := newSliceWithIdAndLastRank(&tSliceType{&tTextType{}}, "a-cool-id", 0)

	require.Len(s.T(), slice1.elements, 0)

	slice2 := slice1.doAppend(alice)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), alice, slice2.elements[0].value)

	slice3, err := slice2.doDel(0)
	require.NoError(s.T(), err)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)

	slice4 := slice3.doAppend(carol)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)
	require.Len(s.T(), slice4.elements, 1)
	require.Equal(s.T(), carol, slice4.elements[0].value)

	slice5 := slice4.doAppend(bob)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)
	require.Len(s.T(), slice4.elements, 1)
	require.Equal(s.T(), sliceElement{2, carol}, slice4.elements[0])
	require.Len(s.T(), slice5.elements, 2)
	require.Equal(s.T(), sliceElement{2, carol}, slice5.elements[0])
	require.Equal(s.T(), sliceElement{3, bob}, slice5.elements[1])

	slice6, err := slice5.doDel(0)
	require.NoError(s.T(), err)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)
	require.Len(s.T(), slice4.elements, 1)
	require.Equal(s.T(), sliceElement{2, carol}, slice4.elements[0])
	require.Len(s.T(), slice5.elements, 2)
	require.Equal(s.T(), sliceElement{2, carol}, slice5.elements[0])
	require.Equal(s.T(), sliceElement{3, bob}, slice5.elements[1])
	require.Len(s.T(), slice6.elements, 1)
	require.Equal(s.T(), sliceElement{3, bob}, slice6.elements[0])
}

func (s *DbZuite) TestSliceSave() {
	ws := defs.MustNewWorksheet("with_slice")
	ws.MustAppend("names", alice)

	// We're reaching into the data store to get the slice id in order to write
	// assertions against it.
	slice := ws.data[42].(*slice)
	theSliceId := slice.id
	slice.lastRank = 89

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, sliceElementsRecs := s.DbState()

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
			Value:       fmt.Sprintf(`[:89:%s`, theSliceId),
		},
	}, valuesRecs)

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

func (s *DbZuite) TestSliceLoad() {
	var (
		wsId       string
		theSliceId string
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_slice")
		ws.MustAppend("names", alice)
		ws.MustAppend("names", carol)
		ws.MustAppend("names", bob)
		ws.MustAppend("names", carol)

		wsId, theSliceId = ws.Id(), (ws.data[42].(*slice)).id

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Load into a fresh worksheet, and look at the slice.
	var (
		fresh *Worksheet
		err   error
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		fresh, err = session.Load("with_slice", wsId)
		return err
	})
	require.Equal(s.T(), []Value{alice, carol, bob, carol}, fresh.MustGetSlice("names"))

	slice := fresh.data[42].(*slice)
	require.Equal(s.T(), theSliceId, slice.id)
	require.Equal(s.T(), 4, slice.lastRank)
	require.Equal(s.T(), &tSliceType{&tTextType{}}, slice.typ)
}

func (s *DbZuite) TestSliceUpdate_appendsThenDelThenAppendAgain() {
	var (
		wsId       string
		theSliceId string
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_slice")
		wsId = ws.Id()
		ws.MustAppend("names", alice)
		ws.MustAppend("names", bob)

		wsId, theSliceId = ws.Id(), (ws.data[42].(*slice)).id

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load("with_slice", wsId)
		if err != nil {
			return err
		}
		ws.MustDel("names", 0)

		return session.Update(ws)
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load("with_slice", wsId)
		if err != nil {
			return err
		}
		ws.MustAppend("names", alice)

		return session.Update(ws)
	})

	wsRecs, valuesRecs, sliceElementsRecs := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 3,
			Name:    "with_slice",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, wsId),
		},
		{
			WorksheetId: wsId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: wsId,
			Index:       IndexVersion,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `2`,
		},
		{
			WorksheetId: wsId,
			Index:       IndexVersion,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `3`,
		},
		{
			WorksheetId: wsId,
			Index:       42,
			FromVersion: 1,
			ToVersion:   1,
			Value:       fmt.Sprintf(`[:2:%s`, theSliceId),
		},
		{
			WorksheetId: wsId,
			Index:       42,
			FromVersion: 2,
			ToVersion:   2,
			Value:       fmt.Sprintf(`[:2:%s`, theSliceId),
		},
		{
			WorksheetId: wsId,
			Index:       42,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`[:3:%s`, theSliceId),
		},
	}, valuesRecs)

	require.Equal(s.T(), []rSliceElement{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   1,
			Rank:        1,
			Value:       `"Alice"`,
		},
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        2,
			Value:       `"Bob"`,
		},
		{
			SliceId:     theSliceId,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Rank:        3,
			Value:       `"Alice"`,
		},
	}, sliceElementsRecs)
}

// impl notes:
// - when loading from DB, must order by index
// - test Get on slice type fails, even if it is undefined
// - test GetSlice on non-slice field fails
// - test Append on non-slice field fails
// - test append of non-assignable type e.g. putting a bool in []text
// - test deletes on out of bound indexes
// - support for slices of slices (may want to do this later)
