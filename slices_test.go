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

	require.False(s.T(), ws.MustIsSet("names"))
	require.Len(s.T(), ws.MustGetSlice("names"), 0)

	ws.MustAppend("names", alice)
	require.True(s.T(), ws.MustIsSet("names"))
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

func (s *Zuite) TestSliceErrors_getOnSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("with_slice")
	_, err := ws.Get("names")
	require.EqualError(s.T(), err, "Get on slice field names, use GetSlice")
}

func (s *Zuite) TestSliceErrors_setOnSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("with_slice")
	err := ws.Set("names", alice)
	require.EqualError(s.T(), err, "Set on slice field names, use Append, or Del")
}

func (s *Zuite) TestSliceErrors_unsetOnSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("with_slice")
	err := ws.Unset("names")
	require.EqualError(s.T(), err, "Unset on slice field names, must use Del")
}

func (s *Zuite) TestSliceErrors_appendOfNonAssignableValue() {
	ws := defs.MustNewWorksheet("with_slice")
	err := ws.Append("names", NewBool(true))
	require.EqualError(s.T(), err, "cannot append bool to []text")
}

func (s *Zuite) TestSliceErrors_delOutOfBound() {
	var err error
	ws := defs.MustNewWorksheet("with_slice")

	// no slice
	err = ws.Del("names", 0)
	require.EqualError(s.T(), err, "index out of range")

	// slice with one element
	ws.MustAppend("names", alice)

	err = ws.Del("names", -1)
	require.EqualError(s.T(), err, "index out of range")

	err = ws.Del("names", 1)
	require.EqualError(s.T(), err, "index out of range")
}

func (s *Zuite) TestSliceErrors_getSliceOnNonSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("simple")
	_, err := ws.GetSlice("name")
	require.EqualError(s.T(), err, "GetSlice on non-slice field name, use Get")
}

func (s *Zuite) TestSliceErrors_appendOnNonSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("simple")
	err := ws.Append("name", alice)
	require.EqualError(s.T(), err, "Append on non-slice field name")
}

func (s *Zuite) TestSliceErrors_delOnNonSliceFailsEvenIfUndefined() {
	ws := defs.MustNewWorksheet("simple")
	err := ws.Del("name", 0)
	require.EqualError(s.T(), err, "Del on non-slice field name")
}

func (s *Zuite) TestSliceOps() {
	slice1 := newSliceWithIdAndLastRank(&SliceType{&tTextType{}}, "a-cool-id", 0)

	require.Len(s.T(), slice1.elements, 0)

	slice2, err := slice1.doAppend(alice)
	require.NoError(s.T(), err)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), alice, slice2.elements[0].value)

	slice3, err := slice2.doDel(0)
	require.NoError(s.T(), err)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)

	slice4, err := slice3.doAppend(carol)
	require.NoError(s.T(), err)

	require.Len(s.T(), slice1.elements, 0)
	require.Len(s.T(), slice2.elements, 1)
	require.Equal(s.T(), sliceElement{1, alice}, slice2.elements[0])
	require.Len(s.T(), slice3.elements, 0)
	require.Len(s.T(), slice4.elements, 1)
	require.Equal(s.T(), carol, slice4.elements[0].value)

	slice5, err := slice4.doAppend(bob)
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

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: ws.Id(),
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
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

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       `Alice`,
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
		fresh, err = session.Load(wsId)
		return err
	})
	require.Equal(s.T(), []Value{alice, carol, bob, carol}, fresh.MustGetSlice("names"))

	slice := fresh.data[42].(*slice)
	require.Equal(s.T(), theSliceId, slice.id)
	require.Equal(s.T(), 4, slice.lastRank)
	require.Equal(s.T(), &SliceType{&tTextType{}}, slice.typ)
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
		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}
		ws.MustDel("names", 0)

		return session.Update(ws)
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load(wsId)
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

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       wsId,
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

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   1,
			Rank:        1,
			Value:       `Alice`,
		},
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        2,
			Value:       `Bob`,
		},
		{
			SliceId:     theSliceId,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Rank:        3,
			Value:       `Alice`,
		},
	}, sliceElementsRecs)
}

func (s *DbZuite) TestSliceOfRefs_saveLoad() {
	var (
		wsId      = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		wsSliceId string
		simple1Id = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
		simple2Id = "ff982d9b-5859-4c88-9fbf-14af5ee08ce2"
	)

	// Initial state.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_slice_of_refs")
		simple1 := defs.MustNewWorksheet("simple")
		simple2 := defs.MustNewWorksheet("simple")
		ws.MustAppend("many_simples", simple1)
		ws.MustAppend("many_simples", simple2)
		simple1.MustSet("name", alice)
		simple2.MustSet("name", bob)

		// We forcibly set worksheets' identifiers to have a known ordering when
		// comparing the db state.
		ws.data[IndexId] = NewText(wsId)
		simple1.data[IndexId] = NewText(simple1Id)
		simple2.data[IndexId] = NewText(simple2Id)

		// We keep the slice' identifier handy for assertions.
		wsSliceId = (ws.data[42].(*slice)).id

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, sliceElementsRecs := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 1,
			Name:    "with_slice_of_refs",
		},
		{
			Id:      simple1Id,
			Version: 1,
			Name:    "simple",
		},
		{
			Id:      simple2Id,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       wsId,
		},
		{
			WorksheetId: wsId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: wsId,
			Index:       42,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`[:2:%s`, wsSliceId),
		},
		{
			WorksheetId: simple1Id,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simple1Id,
		},
		{
			WorksheetId: simple1Id,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simple1Id,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `Alice`,
		},
		{
			WorksheetId: simple2Id,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simple2Id,
		},
		{
			WorksheetId: simple2Id,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simple2Id,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `Bob`,
		},
	}, valuesRecs)

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     wsSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       fmt.Sprintf("*:%s", simple1Id),
		},
		{
			SliceId:     wsSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        2,
			Value:       fmt.Sprintf("*:%s", simple2Id),
		},
	}, sliceElementsRecs)

	// Load into a fresh worksheet, and look at the slice.
	var fresh *Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		var err error
		fresh, err = session.Load(wsId)
		return err
	})

	// Ensure everything was properly loaded.
	slice := fresh.MustGetSlice("many_simples")
	require.Len(s.T(), slice, 2)

	simple1 := slice[0].(*Worksheet)
	require.Equal(s.T(), `"Alice"`, simple1.MustGet("name").String())

	simple2 := slice[1].(*Worksheet)
	require.Equal(s.T(), `"Bob"`, simple2.MustGet("name").String())
}

func (s *DbZuite) TestSliceUpdate_appendUndefinedAndEnsureItLoadsCorrectly() {
	var wsId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_slice")
		wsId = ws.Id()
		ws.MustAppend("names", NewUndefined())

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}
		ws.MustAppend("names", alice)
		ws.MustAppend("names", NewUndefined())

		return session.Update(ws)
	})

	var fresh *Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}

		fresh = ws
		return nil
	})

	require.Equal(s.T(), []Value{
		&Undefined{},
		alice,
		&Undefined{},
	}, fresh.MustGetSlice("names"))
}

func (s *DbZuite) TestSliceUpdate_appendOntoUndefinedSlice() {
	var wsId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_slice")
		wsId = ws.Id()

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}
		ws.MustAppend("names", NewUndefined())

		return session.Update(ws)
	})

	var fresh *Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}

		fresh = ws
		return nil
	})

	require.Equal(s.T(), []Value{
		&Undefined{},
	}, fresh.MustGetSlice("names"))
}
