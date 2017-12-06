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

package db

import (
	"fmt"
	"math"

	"github.com/helloeave/worksheets"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

type DbStore struct {
	defs *worksheets.Definitions
}

func NewStore(defs *worksheets.Definitions) *DbStore {
	return &DbStore{
		defs: defs,
	}
}

func (s *DbStore) Open(tx *runner.Tx) *Session {
	return &Session{
		DbStore: s,
		tx:      tx,
	}
}

// Session is the ... TODO(pascal): write
type Session struct {
	*DbStore
	tx *runner.Tx
}

// Assert Session implements worksheets.Store interface.
var _ worksheets.Store = &Session{}

// rWorksheet represents a record of the worksheets table.
type rWorksheet struct {
	Id      string `db:"id"`
	Version int    `db:"version"`
	Name    string `db:"name"`
}

// rValue represents a record of the worksheet_values table.
type rValue struct {
	Id          int64  `db:"id"`
	WorksheetId string `db:"worksheet_id"`
	Index       int    `db:"index"`
	FromVersion int    `db:"from_version"`
	ToVersion   int    `db:"to_version"`
	Value       string `db:"value"`
}

var tableToEntities = map[string]interface{}{
	"worksheets":       &rWorksheet{},
	"worksheet_values": &rWorksheet{},
}

func (s *Session) Load(name, id string) (*worksheets.Worksheet, error) {
	ws, err := s.defs.UnsafeNewUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	var wsRec rWorksheet
	err = s.tx.
		Select("*").
		From("worksheets").
		Where("id = $1 and name = $2", id, name).
		QueryStruct(&wsRec)
	if err != nil {
		return nil, err
	} else if len(wsRec.Name) == 0 {
		return nil, fmt.Errorf("unknown worksheet %s:%s", name, id)
	}

	var valuesRecs []rValue
	err = s.tx.
		Select("*").
		From("worksheet_values").
		Where("worksheet_id = $1 and from_version <= $2 and $2 <= to_version", id, wsRec.Version).
		QueryStructs(&valuesRecs)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%v\n", valuesRecs)
	for _, valueRec := range valuesRecs {
		value, err := worksheets.NewValue(valueRec.Value)
		if err != nil {
			return nil, err
		}
		ws.UnsafeSet(valueRec.Index, value)
	}

	return ws, nil
}

func (s *Session) Save(ws *worksheets.Worksheet) error {
	// TODO(pascal): handle save of worksheets we already have in the db!

	_, err := s.tx.
		InsertInto("worksheets").
		Columns("*").
		Record(&rWorksheet{
			Id:      ws.Id(),
			Version: ws.Version(),
			Name:    ws.Name(),
		}).
		Exec()
	if err != nil {
		return err
	}

	insert := s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	ws.UnsafeRangeOverData(func(index int, value worksheets.Value) {
		insert.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: ws.Version(),
			ToVersion:   math.MaxInt32,
			Value:       value.String(),
		})
	})
	if _, err := insert.Exec(); err != nil {
		return err
	}

	return nil
}
