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

func (s *Zuite) TestRefsExample() {
	ws := defs.MustNewWorksheet("with_refs")

	require.False(s.T(), ws.MustIsSet("simple"))

	simple := defs.MustNewWorksheet("simple")
	ws.MustSet("simple", simple)
}

func (s *DbZuite) TestRefsSave_noDataInRefWorksheet() {
	var (
		ws     = defs.MustNewWorksheet("with_refs")
		simple = defs.MustNewWorksheet("simple")

		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)
	ws.MustSet("simple", simple)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 1,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsSave_withDataInRefWorksheet() {
	var (
		ws     = defs.MustNewWorksheet("with_refs")
		simple = defs.MustNewWorksheet("simple")

		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)
	ws.MustSet("simple", simple)
	simple.MustSet("name", alice)
	simple.MustSet("age", MustNewValue("120"))

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 1,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `"Alice"`,
		},
		{
			WorksheetId: simpleId,
			Index:       91,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `120`,
		},
	}, valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsSave_refWorksheetAlreadySaved() {
	var (
		ws     = defs.MustNewWorksheet("with_refs")
		simple = defs.MustNewWorksheet("simple")

		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)
	ws.MustSet("simple", simple)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	// We first save simple.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(simple)
	})

	// Then we proceed to save ws.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 1,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsSave_refWorksheetCascadesAnUpdate() {
	var (
		ws     = defs.MustNewWorksheet("with_refs")
		simple = defs.MustNewWorksheet("simple")

		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)
	ws.MustSet("simple", simple)
	simple.MustSet("name", bob)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	// We first save simple.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(simple)
	})

	// We update simple.
	simple.MustSet("name", carol)

	// Then we proceed to save ws.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 1,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 2,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `"Bob"`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `"Carol"`,
		},
	}, valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsSave_withCycles() {
	ws := defs.MustNewWorksheet("with_refs_and_cycles")
	ws.MustSet("point_to_me", ws)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      ws.Id(),
			Version: 1,
			Name:    "with_refs_and_cycles",
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
			Index:       404,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, ws.Id()),
		},
	}, valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsLoad_noCycles() {
	var (
		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_refs")
		simple := defs.MustNewWorksheet("simple")
		ws.MustSet("simple", simple)
		simple.MustSet("name", bob)

		// We forcibly set both worksheets' identifiers to have a known ordering
		// when comparing the db state.
		ws.data[IndexId] = NewText(wsId)
		simple.data[IndexId] = NewText(simpleId)

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Load into a fresh worksheet, and inspect.
	var (
		fresh *Worksheet
		err   error
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		fresh, err = session.Load("with_refs", wsId)
		return err
	})

	value := fresh.MustGet("simple")
	simple, ok := value.(*Worksheet)
	require.True(s.T(), ok)
	require.Equal(s.T(), `"Bob"`, simple.MustGet("name").String())
}

func (s *DbZuite) TestRefsLoad_withCycles() {
	var wsId string

	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := defs.MustNewWorksheet("with_refs_and_cycles")
		wsId = ws.Id()
		ws.MustSet("point_to_me", ws)

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Load into a fresh worksheet, and inspect.
	var (
		fresh *Worksheet
		err   error
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		fresh, err = session.Load("with_refs_and_cycles", wsId)
		return err
	})

	value := fresh.MustGet("point_to_me")
	require.True(s.T(), fresh == value)
}

func (s *DbZuite) TestRefsUpdate_updateParentNoChangeInChild() {
	var (
		ws       = defs.MustNewWorksheet("with_refs")
		simple   = defs.MustNewWorksheet("simple")
		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	// Initial state.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws.MustSet("simple", simple)
		ws.MustSet("some_flag", NewBool(false))

		simple.MustSet("name", carol)

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Update.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("some_flag", NewBool(true))
		return session.Update(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 2,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: wsId,
			Index:       46,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `false`,
		},
		{
			WorksheetId: wsId,
			Index:       46,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `true`,
		},
		{
			WorksheetId: wsId,
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `"Carol"`,
		},
	}, valuesRecs)

	// Upon Update, there should be no more changes to persist.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsUpdate_updateParentWithChangesInChild() {
	var (
		ws       = defs.MustNewWorksheet("with_refs")
		simple   = defs.MustNewWorksheet("simple")
		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	// Initial state.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws.MustSet("simple", simple)
		ws.MustSet("some_flag", NewBool(false))

		simple.MustSet("name", carol)

		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Update.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("some_flag", NewBool(true))
		simple.MustSet("name", bob)
		return session.Update(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 2,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 2,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: wsId,
			Index:       46,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `false`,
		},
		{
			WorksheetId: wsId,
			Index:       46,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `true`,
		},
		{
			WorksheetId: wsId,
			Index:       87,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `"Carol"`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `"Bob"`,
		},
	}, valuesRecs)

	// Upon Update, there should be no more changes to persist.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsUpdate_updateParentWithChildRequiringToBeSaved() {
	var (
		ws       = defs.MustNewWorksheet("with_refs")
		simple   = defs.MustNewWorksheet("simple")
		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// We forcibly set both worksheets' identifiers to have a known ordering
	// when comparing the db state.
	ws.data[IndexId] = NewText(wsId)
	simple.data[IndexId] = NewText(simpleId)

	// Initial state: simple is not attached to ws, and will therefore not be
	// persisted.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	// Update: we attach simple, which should now be persisted.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("simple", simple)
		return session.Update(ws)
	})

	wsRecs, valuesRecs, _ := s.DbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      wsId,
			Version: 2,
			Name:    "with_refs",
		},
		{
			Id:      simpleId,
			Version: 1,
			Name:    "simple",
		},
	}, wsRecs)

	require.Equal(s.T(), []rValue{
		{
			WorksheetId: wsId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, ws.Id()),
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
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: wsId,
			Index:       87,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`"%s"`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       IndexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, valuesRecs)

	// Upon Update, there should be no more changes to persist.
	require.Empty(s.T(), ws.diff())
}
