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

	"github.com/helloeave/dat/sqlx-runner"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestRefsExample() {
	ws := s.defs.MustNewWorksheet("with_refs")

	require.False(s.T(), ws.MustIsSet("simple"))

	simple := s.defs.MustNewWorksheet("simple")
	ws.MustSet("simple", simple)
}

func (s *Zuite) TestRefsErrors_setWithWrongWorksheet() {
	ws := s.defs.MustNewWorksheet("with_refs")
	err := ws.Set("simple", ws)
	require.EqualError(s.T(), err, "cannot assign value of type with_refs to simple")
}

func (s *Zuite) TestRefsSave_noDataInRefWorksheet() {
	var (
		ws     = s.defs.MustNewWorksheet("with_refs")
		simple = s.defs.MustNewWorksheet("simple")

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
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
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

func (s *Zuite) TestRefsSave_withDataInRefWorksheet() {
	var (
		ws     = s.defs.MustNewWorksheet("with_refs")
		simple = s.defs.MustNewWorksheet("simple")

		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)
	ws.MustSet("simple", simple)
	simple.MustSet("name", alice)
	simple.MustSet("age", NewNumberFromInt(120))

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
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
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

func (s *Zuite) TestRefsSave_refWorksheetAlreadySaved() {
	var (
		ws     = s.defs.MustNewWorksheet("with_refs")
		simple = s.defs.MustNewWorksheet("simple")

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
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
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

func (s *Zuite) TestRefsSave_refWorksheetCascadesAnUpdate() {
	var (
		ws     = s.defs.MustNewWorksheet("with_refs")
		simple = s.defs.MustNewWorksheet("simple")

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
			Index:       87,
			FromVersion: 1,
			ToVersion:   1,
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
		},
		{
			WorksheetId: wsId,
			Index:       87,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s@2`, simpleId),
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

func (s *Zuite) TestRefsSave_withCycles() {
	ws := s.defs.MustNewWorksheet("with_refs_and_cycles")
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
			Value:       fmt.Sprintf(`*:%s@1`, ws.Id()),
		},
	}, snap.valuesRecs)

	// Upon Save, orig needs to be set to data.
	require.Empty(s.T(), ws.diff())
}

func (s *Zuite) TestRefsLoad_noCycles() {
	var (
		wsId     = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		simpleId = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := s.defs.MustNewWorksheet("with_refs")
		simple := s.defs.MustNewWorksheet("simple")

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

func (s *Zuite) TestRefsLoad_withCycles() {
	var wsId string

	s.MustRunTransaction(func(tx *runner.Tx) error {
		ws := s.defs.MustNewWorksheet("with_refs_and_cycles")
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

func (s *Zuite) TestRefsUpdate_updateParentNoChangeInChild() {
	var (
		ws       = s.defs.MustNewWorksheet("with_refs")
		simple   = s.defs.MustNewWorksheet("simple")
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
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
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

func (s *Zuite) TestRefsUpdate_updateParentWithChangesInChild() {
	var (
		ws       = s.defs.MustNewWorksheet("with_refs")
		simple   = s.defs.MustNewWorksheet("simple")
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
			ToVersion:   1,
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
		},
		{
			WorksheetId: wsId,
			Index:       87,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       fmt.Sprintf(`*:%s@2`, simpleId),
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

func (s *Zuite) TestRefsUpdate_updateParentWithChildRequiringToBeSaved() {
	var (
		ws       = s.defs.MustNewWorksheet("with_refs")
		simple   = s.defs.MustNewWorksheet("simple")
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
			Value:       fmt.Sprintf(`*:%s@1`, simpleId),
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

func (s *Zuite) TestRefsWithVersionNumber() {
	defs := MustNewDefinitions(strings.NewReader(`type parent worksheet {
		83:child child
		89:text text
	}

	type child worksheet {
		97:text text
	}`))

	var (
		store    = NewStore(defs)
		parentId = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		childId  = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// Throughout this test, we work with a parent worksheet pointing to a
	// child worksheet. We mutate the parent, and mutate the child, to check
	// that all cases of ref tracking are properly handled.
	//
	// We start with
	//
	//     parent("parent text (A)") -> child("child text (i)")
	//
	// where both parent and child are at version 1.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		child := defs.MustNewWorksheet("child")
		child.MustSet("text", NewText("child text (i)"))

		parent := defs.MustNewWorksheet("parent")
		parent.MustSet("child", child)
		parent.MustSet("text", NewText("parent text (A)"))

		forciblySetId(child, childId)
		forciblySetId(parent, parentId)

		session := store.Open(tx)
		_, err := session.SaveOrUpdate(parent)
		return err
	})

	// Here, we simply expect all records to be persisted normally, and all
	// references to be @1.
	require.Equal(s.T(), []rValueForTesting{
		// parent
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `*:` + childId + `@1`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `parent text (A)`,
		},

		// child
		{
			WorksheetId: childId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       childId,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `child text (i)`,
		},
	}, s.snapshotDbState().valuesRecs)

	// We proceed to modify only the child, which means the child will go
	// from version 1 to version 2. However, since the parent has not been
	// modified, the parent does not change version, and therefore, its pointer
	// to the child stays fixed at version 1. Said another way, if we load
	// the parent at version 1 (historical load), we would want the child
	// at version 1 to be loaded (hence not seeing the update below).
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)

		child, err := session.Load(childId)
		if err != nil {
			return err
		}
		child.MustSet("text", NewText("child text (ii)"))

		if _, err := session.SaveOrUpdate(child); err != nil {
			return err
		}

		return nil
	})

	// From a records standpoint, we do not expect any changes to the parent
	// worksheets' values, only to the child.
	require.Equal(s.T(), []rValueForTesting{
		// parent
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `*:` + childId + `@1`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `parent text (A)`,
		},

		// child
		{
			WorksheetId: childId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       childId,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `child text (i)`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `child text (ii)`,
		},
	}, s.snapshotDbState().valuesRecs)

	// Now, we modify the parent only. Since we load "at head", the version of
	// the child being loaded is the latest, i.e. version 2. As a result,
	// when we store the parent, the reference to the child will be updated.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)

		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustSet("text", NewText("parent text (B)"))

		if _, err := session.SaveOrUpdate(parent); err != nil {
			return err
		}

		return nil
	})

	// In addition to the text value changing in the parent, the reference to
	// the child should also change.
	require.Equal(s.T(), []rValueForTesting{
		// parent
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `*:` + childId + `@1`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `*:` + childId + `@2`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `parent text (A)`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `parent text (B)`,
		},

		// child
		{
			WorksheetId: childId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       childId,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `child text (i)`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `child text (ii)`,
		},
	}, s.snapshotDbState().valuesRecs)

	// Lastly, we modify both the parent and the child. When we store them
	// we need the parent's pointer to update to pointing to child @ version 3.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)

		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustSet("text", NewText("parent text (C)"))

		child := parent.MustGet("child").(*Worksheet)
		child.MustSet("text", NewText("child text (iii)"))

		if _, err := session.SaveOrUpdate(parent); err != nil {
			return err
		}

		return nil
	})

	// Here, in addition to the changes for storing updates to the two texts
	// (i.e. the parent text, and child text), we also need to see the
	// reference update to the child to change.
	require.Equal(s.T(), []rValueForTesting{
		// parent
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `2`,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `3`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `*:` + childId + `@1`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `*:` + childId + `@2`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `*:` + childId + `@3`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `parent text (A)`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `parent text (B)`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `parent text (C)`,
		},

		// child
		{
			WorksheetId: childId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       childId,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `2`,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `3`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `child text (i)`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 2,
			ToVersion:   2,
			Value:       `child text (ii)`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 3,
			ToVersion:   math.MaxInt32,
			Value:       `child text (iii)`,
		},
	}, s.snapshotDbState().valuesRecs)
}

// TestRefsWithoutVersionNumber tests that references which were stored without
// version pointers (`*:UUID` instead of `*:UUID@version`) can be properly
// loaded, and then properly convert to being versioned afterwards.
//
// (This is important for backwards compatibility, and databases storing
// worksheets in the prior format, before proper pointer versioning was
// introduced.)
func (s *Zuite) TestRefsWithoutVersionNumber() {
	defs := MustNewDefinitions(strings.NewReader(`type parent worksheet {
		83:child child
		89:text text
	}

	type child worksheet {
		97:text text
	}`))

	var (
		store    = NewStore(defs)
		parentId = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		childId  = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// Throughout this test, we work with a parent worksheet pointing to a
	// child worksheet. We mutate the parent, and mutate the child, to check
	// that all cases of ref tracking are properly handled.
	//
	// We start with
	//
	//     parent("parent text (A)") -> child("child text (i)")
	//
	// where both parent and child are at version 1.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		child := defs.MustNewWorksheet("child")
		child.MustSet("text", NewText("child text (i)"))

		parent := defs.MustNewWorksheet("parent")
		parent.MustSet("child", child)
		parent.MustSet("text", NewText("parent text (A)"))

		forciblySetId(child, childId)
		forciblySetId(parent, parentId)

		session := store.Open(tx)
		_, err := session.SaveOrUpdate(parent)
		return err
	})

	// We force the old format `*:UUID` instead of `*:UUID@version`.
	res, err := s.db.DB.Exec(`
		update
			worksheet_values
		set
			value = '*:e310c9b6-fc48-4b29-8a66-eeafa9a8ec16'
		where
			value = '*:e310c9b6-fc48-4b29-8a66-eeafa9a8ec16@1'
		`)
	s.Require().NoError(err)
	rowsAffected, err := res.RowsAffected()
	s.Require().NoError(err)
	s.Require().Equal(int64(1), rowsAffected)

	// We load the parent (thus checking backwards compatibility), and update
	// its text. We then check that the reference was updated to the new format.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)

		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustSet("text", NewText("parent text (B)"))

		if _, err := session.SaveOrUpdate(parent); err != nil {
			return err
		}

		return nil
	})

	require.Equal(s.T(), []rValueForTesting{
		// parent
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `*:` + childId,
		},
		{
			WorksheetId: parentId,
			Index:       83,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `*:` + childId + `@1`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `parent text (A)`,
		},
		{
			WorksheetId: parentId,
			Index:       89,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `parent text (B)`,
		},

		// child
		{
			WorksheetId: childId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       childId,
		},
		{
			WorksheetId: childId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: childId,
			Index:       97,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `child text (i)`,
		},
	}, s.snapshotDbState().valuesRecs)
}

func (s *Zuite) TestRefsInSlicesWithVersionNumber() {
	defs := MustNewDefinitions(strings.NewReader(`type parent worksheet {
		83:children []child
		89:text text
	}

	type child worksheet {
		97:text text
	}`))

	var (
		store    = NewStore(defs)
		parentId = "d55cba7e-d08f-43df-bcd7-f48c2ecf6da7"
		childId  = "e310c9b6-fc48-4b29-8a66-eeafa9a8ec16"
	)

	// Throughout this test, we work with a parent worksheet pointing to a
	// single child worksheet (though its children slice). We mutate the parent,
	// and mutate the child, to check that all cases of ref tracking are
	// properly handled.
	//
	// We start with
	//
	//     parent("parent text (A)") -> children{ child("child text (i)") }
	//
	// where both parent and child are at version 1.
	var theSliceId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		child := defs.MustNewWorksheet("child")
		child.MustSet("text", NewText("child text (i)"))

		parent := defs.MustNewWorksheet("parent")
		parent.MustAppend("children", child)
		theSliceId = parent.data[83].(*Slice).id
		parent.MustSet("text", NewText("parent text (A)"))

		forciblySetId(child, childId)
		forciblySetId(parent, parentId)

		session := store.Open(tx)
		_, err := session.SaveOrUpdate(parent)
		return err
	})

	snap := s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      parentId,
			Version: 1,
			Name:    "parent",
		},
		{
			Id:      childId,
			Version: 1,
			Name:    "child",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       `*:` + childId + `@1`,
		},
	}, snap.sliceElementsRecs)

	// We modify the child, which will make it bump from version 1 to version 2.
	// However, since the parent isn't modified, the children slice will not
	// be modified.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		child, err := session.Load(childId)
		if err != nil {
			return err
		}
		child.MustSet("text", NewText("child text (ii)"))
		_, err = session.SaveOrUpdate(child)
		return err
	})

	snap = s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      parentId,
			Version: 1,
			Name:    "parent",
		},
		{
			Id:      childId,
			Version: 2,
			Name:    "child",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       `*:` + childId + `@1`,
		},
	}, snap.sliceElementsRecs)

	// We now load the parent, and save it back. It's slice will have been
	// modified since the load saw the child at version 1, whereas the child is
	// now at version 2.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustSet("text", NewText("parent text (B)"))
		_, err = session.SaveOrUpdate(parent)
		return err
	})

	snap = s.snapshotDbState()

	require.Equal(s.T(), []rWorksheet{
		{
			Id:      parentId,
			Version: 2,
			Name:    "parent",
		},
		{
			Id:      childId,
			Version: 2,
			Name:    "child",
		},
	}, snap.wsRecs)

	require.Equal(s.T(), []rSliceElementForTesting{
		{
			SliceId:     theSliceId,
			FromVersion: 1,
			ToVersion:   1,
			Rank:        1,
			Value:       `*:` + childId + `@1`,
		},
		{
			SliceId:     theSliceId,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Rank:        1,
			Value:       `*:` + childId + `@2`,
		},
	}, snap.sliceElementsRecs)
}
