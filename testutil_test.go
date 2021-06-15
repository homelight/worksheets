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
	"database/sql"
	"fmt"
	"strings"
	"testing"

	runner "github.com/homelight/dat/sqlx-runner"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// some useful values
var (
	alice = NewText("Alice")
	bob   = NewText("Bob")
	carol = NewText("Carol")
)

// definitions
var defs = `
type simple worksheet {
	83:name text
	91:age  number[0]
}

type all_types worksheet {
	 1:text      text
	 2:bool      bool
	 3:num_0     number[0]
	 4:num_2     number[2]
	 5:undefined undefined
	 6:ws        all_types
	 7:slice_t   []text
	 11:slice_b  []bool
	 12:slice_bu []bool
	 9:slice_n0  []number[0]
	10:slice_n2  []number[2]
	13:slice_nu  []number[0]
	 8:slice_ws  []all_types
}

type with_slice worksheet {
	42:names []text
}

type with_slice_of_refs worksheet {
	42:many_simples []simple
}

type with_refs worksheet {
	46:some_flag bool
	87:simple simple
}

type with_refs_and_cycles worksheet {
	404:point_to_me         with_refs_and_cycles
	500:point_to_my_friends []with_refs_and_cycles
}

type with_repeat_refs worksheet {
	111:point_to_something      simple
	112:point_to_the_same_thing simple
	113:and_again               []simple
}

type Ping worksheet {
	123:point_to_pong pong
	124:slice_of_Ping []Ping
}

type pong worksheet {
	321:point_to_Ping Ping
}

type DefaultMappingsTest worksheet {
	83:Name  text
	91:Age   number[0]
	99:Child DefaultMappingsTest
}`

func forciblySetId(ws *Worksheet, id string) {
	ws.data[indexId] = NewText(id)
}

type allDefs struct {
	defs                    *Definitions
	cloneDefs               *Definitions
	defsForSelectors        *Definitions
	defsCrossWs             *Definitions
	defsCrossWsThroughSlice *Definitions
	enumsDefs               *Definitions
}

func newAllDefs() allDefs {
	var s allDefs

	// When initializing, we purposefully ignore errors to make it easier to work
	// on specific parts of the parser by running single tests:
	// - If we're running a single test which does not depend on these
	//   definitions, we shouldn't fail early, so as to provide feedback to the
	//   programmer on the test being ran (rather than whether full parsing works).
	// - And since the suite itself will fail if any of these are nil, we are not
	//   changing the test suite outcome by ignoring errors, simply shifting where
	//   and how these errors are reported.
	s.defs, _ = NewDefinitions(strings.NewReader(defs))
	s.cloneDefs, _ = NewDefinitions(strings.NewReader(cloneDefs))
	s.defsForSelectors, _ = NewDefinitions(strings.NewReader(defsForSelectors))
	s.defsCrossWs, _ = NewDefinitions(strings.NewReader(defsCrossWs))
	s.defsCrossWsThroughSlice, _ = NewDefinitions(strings.NewReader(defsCrossWsThroughSlice), defsCrossWsThroughSliceOptions)
	s.enumsDefs, _ = NewDefinitions(strings.NewReader(enumsDefs))

	return s
}

type Zuite struct {
	suite.Suite
	allDefs
	db    *runner.DB
	store *DbStore
}

func (s *Zuite) SetupSuite() {
	// init
	s.allDefs = newAllDefs()

	// db
	dbUrl := "postgres://ws_user:@localhost/ws_test?sslmode=disable"
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		panic(err)
	}
	s.db = runner.NewDB(db, "postgres")

	// store
	s.store = NewStore(s.defs)
}

func (s *Zuite) SetupTest() {
	for table := range tableToEntities {
		_, err := s.db.Exec(fmt.Sprintf("truncate %s", table))
		if err != nil {
			panic(err)
		}
	}
}

func (s *Zuite) TearDownSuite() {
	err := s.db.DB.Close()
	if err != nil {
		panic(err)
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}

func RunTransaction(db *runner.DB, fn func(tx *runner.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.AutoRollback()

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Zuite) RunTransaction(fn func(tx *runner.Tx) error) error {
	return RunTransaction(s.db, fn)
}

func (s *Zuite) MustRunTransaction(fn func(tx *runner.Tx) error) {
	err := s.RunTransaction(fn)
	require.NoError(s.T(), err)
}

type fakeClock struct {
	now int64
}

// Assert that fakeClock implements the clock interface.
var _ clock = &fakeClock{}

func (fc *fakeClock) nowAsUnixNano() int64 {
	return fc.now
}

type rValueForTesting struct {
	WorksheetId string
	Index       int
	FromVersion int
	ToVersion   int
	Value       string
	IsUndefined bool
}

type rSliceElementForTesting struct {
	SliceId     string
	Rank        int
	FromVersion int
	ToVersion   int
	Value       string
	IsUndefined bool
}

type dbState struct {
	wsRecs            []rWorksheet
	editRecs          []rEdit
	valuesRecs        []rValueForTesting
	parentsRecs       []rParent
	sliceElementsRecs []rSliceElementForTesting
}

func (s *Zuite) snapshotDbState() *dbState {
	var (
		err                 error
		wsRecs              []rWorksheet
		editRecs            []rEdit
		dbValuesRecs        []rValue
		parentsRecs         []rParent
		dbSliceElementsRecs []rSliceElement
	)

	err = s.db.
		Select("*").
		From("worksheets").
		OrderBy("id").
		QueryStructs(&wsRecs)
	require.NoError(s.T(), err)

	err = s.db.
		Select("*").
		From("worksheet_edits").
		OrderBy("worksheet_id, to_version").
		QueryStructs(&editRecs)
	require.NoError(s.T(), err)

	err = s.db.
		Select("*").
		From("worksheet_values").
		OrderBy("worksheet_id, index, from_version").
		QueryStructs(&dbValuesRecs)
	require.NoError(s.T(), err)

	err = s.db.
		Select("*").
		From("worksheet_parents").
		OrderBy("child_id, parent_id, parent_field_index").
		QueryStructs(&parentsRecs)
	require.NoError(s.T(), err)

	err = s.db.
		Select("*").
		From("worksheet_slice_elements").
		OrderBy("slice_id, rank, from_version").
		QueryStructs(&dbSliceElementsRecs)
	require.NoError(s.T(), err)

	// rValue to rValueForTesting
	valuesRecs := make([]rValueForTesting, len(dbValuesRecs))
	for i, dbValueRec := range dbValuesRecs {
		valuesRecs[i] = rValueForTesting{
			WorksheetId: dbValueRec.WorksheetId,
			Index:       dbValueRec.Index,
			FromVersion: dbValueRec.FromVersion,
			ToVersion:   dbValueRec.ToVersion,
		}
		if dbValueRec.Value != nil {
			valuesRecs[i].Value = *dbValueRec.Value
		} else {
			valuesRecs[i].IsUndefined = true
		}
	}

	// rSliceElement to rSliceElementForTesting
	sliceElementsRecs := make([]rSliceElementForTesting, len(dbSliceElementsRecs))
	for i, dbSliceElementRec := range dbSliceElementsRecs {
		sliceElementsRecs[i] = rSliceElementForTesting{
			SliceId:     dbSliceElementRec.SliceId,
			Rank:        dbSliceElementRec.Rank,
			FromVersion: dbSliceElementRec.FromVersion,
			ToVersion:   dbSliceElementRec.ToVersion,
		}
		if dbSliceElementRec.Value != nil {
			sliceElementsRecs[i].Value = *dbSliceElementRec.Value
		} else {
			sliceElementsRecs[i].IsUndefined = true
		}
	}

	return &dbState{
		wsRecs:            wsRecs,
		editRecs:          editRecs,
		valuesRecs:        valuesRecs,
		parentsRecs:       parentsRecs,
		sliceElementsRecs: sliceElementsRecs,
	}
}
