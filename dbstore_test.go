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
	"time"

	"github.com/helloeave/dat/sqlx-runner"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestDbExample() {
	ws := s.store.defs.MustNewWorksheet("simple")
	ws.MustSet("name", NewText("Alice"))

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	var wsFromStore *Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		var err error
		wsFromStore, err = session.Load(ws.Id())
		return err
	})

	require.Len(s.T(), wsFromStore.MustGet("id").String(), 38)
	require.Equal(s.T(), `1`, wsFromStore.MustGet("version").String())
	require.Equal(s.T(), `"Alice"`, wsFromStore.MustGet("name").String())
}

func (s *Zuite) TestSave() {
	ws, err := s.store.defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	var editId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		session.clock = &fakeClock{1234}

		var err error
		editId, err = session.Save(ws)
		return err
	})

	snap := s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      ws.Id(),
			Version: 1,
			Name:    "simple",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rEdit{
		{
			EditId:      editId,
			CreatedAt:   1234,
			WorksheetId: ws.Id(),
			ToVersion:   1,
		},
	}, snap.editRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: ws.Id(),
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: ws.Id(),
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: ws.Id(),
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `Alice`,
		},
	}, snap.valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())

	// Edit.
	var (
		editCreatedAt time.Time
		editTouchedWs map[string]int
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)

		var err error
		editCreatedAt, editTouchedWs, err = session.Edit(editId)
		return err
	})
	require.Equal(s.T(), int64(1234), editCreatedAt.UnixNano())
	require.Equal(s.T(), map[string]int{
		ws.Id(): 1,
	}, editTouchedWs)
}

func (s *Zuite) TestUpdate() {
	ws, err := s.store.defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	var saveId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		session.clock = &fakeClock{1000}

		var err error
		saveId, err = session.Save(ws)
		return err
	})

	err = ws.Set("name", NewText("Bob"))
	require.NoError(s.T(), err)

	var updateId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		session.clock = &fakeClock{2000}

		var err error
		updateId, err = session.Update(ws)
		return err
	})

	snap := s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      ws.Id(),
			Version: 2,
			Name:    "simple",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rEdit{
		{
			EditId:      saveId,
			CreatedAt:   1000,
			WorksheetId: ws.Id(),
			ToVersion:   1,
		},
		{
			EditId:      updateId,
			CreatedAt:   2000,
			WorksheetId: ws.Id(),
			ToVersion:   2,
		},
	}, snap.editRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: ws.Id(),
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: ws.Id(),
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: ws.Id(),
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: ws.Id(),
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `Alice`,
		},
		{
			WorksheetId: ws.Id(),
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `Bob`,
		},
	}, snap.valuesRecs)

	// Upon update, version must increase
	require.Equal(s.T(), 2, ws.Version())

	// Upon Update, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())

	// Edit.
	var (
		updateCreatedAt time.Time
		updateTouchedWs map[string]int
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)

		var err error
		updateCreatedAt, updateTouchedWs, err = session.Edit(updateId)
		return err
	})
	require.Equal(s.T(), int64(2000), updateCreatedAt.UnixNano())
	require.Equal(s.T(), map[string]int{
		ws.Id(): 2,
	}, updateTouchedWs)
}

func (s *Zuite) TestUpdateUndefinedField() {
	ws, err := s.store.defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	err = ws.Set("name", NewText("Alice"))
	require.NoError(s.T(), err)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	err = ws.Set("age", MustNewValue("73"))
	require.NoError(s.T(), err)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Update(ws)
		return err
	})
}

func (s *Zuite) TestProperlyLoadUndefinedField() {
	var wsId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := s.defs.MustNewWorksheet("simple")
		wsId = ws.Id()
		ws.MustSet("age", MustNewValue("123456"))

		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)

		ws, err := session.Load(wsId)
		if err != nil {
			return err
		}
		ws.MustUnset("age")

		_, err = session.Update(ws)
		return err
	})

	// Fresh load should show age as being unset.
	var fresh *Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		var err error
		fresh, err = session.Load(wsId)
		return err
	})

	require.False(s.T(), fresh.MustIsSet("age"))

	// Lastly, check db state.
	snap := s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 2,
			Name:    "simple",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       wsId,
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: wsId,
			Index:       91,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `123456`,
		},
		{
			WorksheetId: wsId,
			Index:       91,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			IsUndefined: true,
		},
	}, snap.valuesRecs)
}

func (s *Zuite) TestUpdateOnUpdateDoesNothing() {
	ws := s.store.defs.MustNewWorksheet("simple")
	ws.MustSet("name", alice)

	require.Equal(s.T(), 1, ws.Version())

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	require.Equal(s.T(), 1, ws.Version())

	ws.MustSet("name", bob)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Update(ws)
		return err
	})

	require.Equal(s.T(), 2, ws.Version())

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Update(ws)
		return err
	})

	require.Equal(s.T(), 2, ws.Version())
}

func (s *Zuite) TestUpdateDetectsConcurrentModifications_onWorksheetVersion() {
	ws := s.store.defs.MustNewWorksheet("simple")
	ws.MustSet("name", alice)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// simulate modification performed by other
	_, err := s.db.Exec("update worksheets set version = version + 1 where id = $1", ws.Id())
	require.NoError(s.T(), err)

	// update should fail
	ws.MustSet("name", bob)
	var errFromUpdate error
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, errFromUpdate = session.Update(ws)
		return nil
	})

	require.EqualError(s.T(), errFromUpdate, "concurrent update detected")
}

func (s *Zuite) TestUpdateDetectsConcurrentModifications_onEditRecordAlreadyPresent() {
	ws := s.store.defs.MustNewWorksheet("simple")
	ws.MustSet("name", alice)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// simulate other update racing to add the rEdit record
	_, err := s.db.
		InsertInto("worksheet_edits").
		Columns("*").
		Record(rEdit{
			EditId:      uuid.Must(uuid.NewV4()).String(),
			WorksheetId: ws.Id(),
			ToVersion:   ws.Version() + 1,
		}).
		Exec()
	require.NoError(s.T(), err)

	// update should fail
	ws.MustSet("name", bob)
	errFromUpdate := s.RunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Update(ws)
		return err
	})

	require.Regexp(s.T(), `^concurrent update detected \(.*\)$`, errFromUpdate)
}

func (s *Zuite) TestSignoffPattern() {
	defs := MustNewDefinitions(strings.NewReader(`type needs_sign_off worksheet {
		1:signoff_at number[0]
		2:is_signedoff bool computed_by {
			return signoff_at + 1 == version
		}
		3:data text
	}`))

	ws := defs.MustNewWorksheet("needs_sign_off")

	// even with a fresh worksheet, is_signedoff should be set
	require.Equal(s.T(), "1", ws.MustGet("version").String())
	require.Equal(s.T(), "false", ws.MustGet("is_signedoff").String())

	// data is inputed into the worksheet
	ws.MustSet("data", NewText("important data 1"))
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.SaveOrUpdate(ws)
		return err
	})
	require.Equal(s.T(), "1", ws.MustGet("version").String())
	require.Equal(s.T(), "false", ws.MustGet("is_signedoff").String())

	// worksheet is signed off
	ws.MustSet("signoff_at", MustNewValue(fmt.Sprintf("%d", ws.Version())))
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.SaveOrUpdate(ws)
		return err
	})
	require.Equal(s.T(), "2", ws.MustGet("version").String())
	require.Equal(s.T(), "true", ws.MustGet("is_signedoff").String())

	// data is modified
	ws.MustSet("data", NewText("important data 2"))
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.SaveOrUpdate(ws)
		return err
	})
	require.Equal(s.T(), "3", ws.MustGet("version").String())
	require.Equal(s.T(), "false", ws.MustGet("is_signedoff").String())

	// worksheet is signed off again, except this time, the update will fail
	// (due to a concurrent modification)
	ws.MustSet("signoff_at", MustNewValue(fmt.Sprintf("%d", ws.Version())))
	_, err := s.db.Exec("update worksheets set version = version + 1 where id = $1", ws.Id())
	require.NoError(s.T(), err)

	var errFromUpdate error
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, errFromUpdate = session.SaveOrUpdate(ws)
		return nil
	})
	require.EqualError(s.T(), errFromUpdate, "concurrent update detected")

	// which means that is_signedoff should not have been modified
	require.Equal(s.T(), "3", ws.MustGet("version").String())
	require.Equal(s.T(), "false", ws.MustGet("is_signedoff").String())
}

func (s *DbZuite) TestDeprecatedField() {
	defs := MustNewDefinitions(strings.NewReader(`worksheet some_worksheet {
		1:field_one text
		2:field_two text
		3:field_three text
	}`))

	store := NewStore(defs)
	var id string

	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("some_worksheet")
		ws.MustSet("field_one", NewText("one"))
		ws.MustSet("field_two", NewText("two"))
		ws.MustSet("field_three", NewText("three"))

		id = ws.Id()
		session := store.Open(tx)
		_, err := session.SaveOrUpdate(ws)
		return err
	})

	// update definition
	defs = MustNewDefinitions(strings.NewReader(`worksheet some_worksheet {
		1:field_one text
		3:field_three text
	}`))
	store = NewStore(defs)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		_, err := session.Load(id)
		return err
	})
}
