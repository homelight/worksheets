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

	runner "github.com/helloeave/dat/sqlx-runner"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestRefsExample() {
	ws := defs.MustNewWorksheet("with_refs")

	require.False(s.T(), ws.MustIsSet("simple"))

	simple := defs.MustNewWorksheet("simple")
	ws.MustSet("simple", simple)
}

func (s *Zuite) TestRefsErrors_setWithWrongWorksheet() {
	ws := defs.MustNewWorksheet("with_refs")
	err := ws.Set("simple", ws)
	require.EqualError(s.T(), err, "cannot assign value of type with_refs to field of type simple")
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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, snap.valuesRecs)

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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `Alice`,
		},
		{
			WorksheetId: simpleId,
			Index:       91,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `120`,
		},
	}, snap.valuesRecs)

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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	// We save simple. Because ws is a parent to simple, we will also
	// saveOrUpdate ws.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(simple)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, snap.valuesRecs)

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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	// We first save simple, this also saves ws since it is a parent.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(simple)
		return err
	})

	// We update simple.
	simple.MustSet("name", carol)

	// Then we proceed to save ws.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.SaveOrUpdate(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
		},
		{
			WorksheetId: wsId,
			Index:       indexVersion,
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `Bob`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `Carol`,
		},
	}, snap.valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *DbZuite) TestRefsSave_withCycles() {
	ws := defs.MustNewWorksheet("with_refs_and_cycles")
	ws.MustSet("point_to_me", ws)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	snap := s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      ws.Id(),
			Version: 1,
			Name:    "with_refs_and_cycles",
		},
	}, snap.wsRecs)

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
			Index:       404,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, ws.Id()),
		},
	}, snap.valuesRecs)

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

		// We forcibly set both worksheets' identifiers to have a known ordering
		// when comparing the db state.
		forciblySetId(ws, wsId)
		forciblySetId(simple, simpleId)

		// ws.simple = simple
		// simple.name = bob
		ws.MustSet("simple", simple)
		simple.MustSet("name", bob)

		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// Load into a fresh worksheet, and inspect.
	var (
		fresh *Worksheet
		err   error
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		fresh, err = session.Load(wsId)
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
		_, err := session.Save(ws)
		return err
	})

	// Load into a fresh worksheet, and inspect.
	var (
		fresh *Worksheet
		err   error
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		fresh, err = session.Load(wsId)
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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	// Initial state.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws.MustSet("simple", simple)
		ws.MustSet("some_flag", NewBool(false))

		simple.MustSet("name", carol)

		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// Update.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("some_flag", NewBool(true))
		_, err := session.Update(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `Carol`,
		},
	}, snap.valuesRecs)

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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	// Initial state.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws.MustSet("simple", simple)
		ws.MustSet("some_flag", NewBool(false))

		simple.MustSet("name", carol)

		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// Update.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("some_flag", NewBool(true))
		simple.MustSet("name", bob)
		_, err := session.Update(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
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
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `Carol`,
		},
		{
			WorksheetId: simpleId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `Bob`,
		},
	}, snap.valuesRecs)

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
	forciblySetId(ws, wsId)
	forciblySetId(simple, simpleId)

	// Initial state: simple is not attached to ws, and will therefore not be
	// persisted.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		_, err := session.Save(ws)
		return err
	})

	// Update: we attach simple, which should now be persisted.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		ws.MustSet("simple", simple)
		_, err := session.Update(ws)
		return err
	})

	snap := s.snapshotDbState()

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
	}, snap.wsRecs)

	require.Equal(s.T(), []rValueForTesting{
		{
			WorksheetId: wsId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       ws.Id(),
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
			Index:       87,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s`, simpleId),
		},
		{
			WorksheetId: simpleId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       simpleId,
		},
		{
			WorksheetId: simpleId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
	}, snap.valuesRecs)

	// Upon Update, there should be no more changes to persist.
	require.Empty(s.T(), ws.diff())
}
