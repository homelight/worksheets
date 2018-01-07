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
	"strconv"
	"strings"

	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

// Store ... TODO(pascal): write about abstraction.
type Store interface {
	// Load loads the worksheet with identifier `id` from the store.
	Load(name, id string) (*Worksheet, error)

	// Save saves a new worksheet to the store.
	Save(ws *Worksheet) error

	// Update updates an existing worksheet in the store.
	Update(ws *Worksheet) error
}

type DbStore struct {
	defs *Definitions
}

func NewStore(defs *Definitions) *DbStore {
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

// Assert Session implements Store interface.
var _ Store = &Session{}

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

// rSliceElement represents a record of the worksheet_slice_elements table.
type rSliceElement struct {
	Id          int64  `db:"id"`
	SliceId     string `db:"slice_id"`
	Rank        int    `db:"rank"`
	FromVersion int    `db:"from_version"`
	ToVersion   int    `db:"to_version"`
	Value       string `db:"value"`
}

var tableToEntities = map[string]interface{}{
	"worksheets":               &rWorksheet{},
	"worksheet_values":         &rValue{},
	"worksheet_slice_elements": &rSliceElement{},
}

func (s *Session) Load(name, id string) (*Worksheet, error) {
	graph := make(map[string]*Worksheet)
	return s.load(graph, name, id)
}
func (s *Session) load(graph map[string]*Worksheet, name, id string) (*Worksheet, error) {
	wsRef := fmt.Sprintf("%s:%s", name, id)

	if ws, ok := graph[wsRef]; ok {
		return ws, nil
	}

	ws, err := s.defs.newUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	var wsRecs []rWorksheet
	if err = s.tx.
		Select("*").
		From("worksheets").
		Where("id = $1 and name = $2", id, name).
		QueryStructs(&wsRecs); err != nil {
		return nil, fmt.Errorf("unable to load worksheets records: %s", err)
	} else if len(wsRecs) == 0 {
		return nil, fmt.Errorf("unknown worksheet %s:%s", name, id)
	}

	wsRec := wsRecs[0]
	graph[wsRef] = ws

	var (
		valuesRecs   []rValue
		slicesToLoad = make(map[string]*slice)
		slicesIds    []interface{}
	)
	err = s.tx.
		Select("*").
		From("worksheet_values").
		Where("worksheet_id = $1", id).
		Where("from_version <= $1 and $1 <= to_version", wsRec.Version).
		QueryStructs(&valuesRecs)
	if err != nil {
		return nil, err
	}
	for _, valueRec := range valuesRecs {
		index := valueRec.Index

		// field
		field, ok := ws.def.fieldsByIndex[index]
		if !ok {
			return nil, fmt.Errorf("unknown value with field index %d", index)
		}

		// type dependent treatment
		var value Value
		switch t := field.typ.(type) {
		case *tSliceType:
			if !strings.HasPrefix(valueRec.Value, "[:") {
				return nil, fmt.Errorf("unreadable value for slice %s", valueRec.Value)
			}
			parts := strings.Split(valueRec.Value, ":")
			if len(parts) != 3 {
				return nil, fmt.Errorf("unreadable value for slice %s", valueRec.Value)
			}
			lastRank, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("unreadable value for slice %s", valueRec.Value)
			}
			slice := newSliceWithIdAndLastRank(t, parts[2], lastRank)
			slicesToLoad[slice.id] = slice
			slicesIds = append(slicesIds, slice.id)
			value, err = slice, nil
		case *tWorksheetType:
			if !strings.HasPrefix(valueRec.Value, "*:") {
				return nil, fmt.Errorf("unreadable value for ref %s", valueRec.Value)
			}
			parts := strings.Split(valueRec.Value, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("unreadable value for ref %s", valueRec.Value)
			}
			value, err = s.load(graph, t.name, parts[1])
			if err != nil {
				return nil, fmt.Errorf("unable to load referenced worksheet %s: %s", parts[1], err)
			}
		default:
			value, err = NewValue(valueRec.Value)
		}
		if err != nil {
			return nil, err
		}

		// set orig and data
		ws.orig[index] = value
		ws.data[index] = value
	}

	if len(slicesToLoad) != 0 {
		var sliceElementsRecs []rSliceElement
		err = s.tx.
			Select("*").
			From("worksheet_slice_elements").
			Where(inClause("slice_id", len(slicesIds)), slicesIds...).
			Where("from_version <= $1 and $1 <= to_version", wsRec.Version).
			OrderBy("slice_id, rank").
			QueryStructs(&sliceElementsRecs)
		if err != nil {
			return nil, err
		}
		for _, sliceElementsRec := range sliceElementsRecs {
			value, err := NewValue(sliceElementsRec.Value) // wrong, this could be a slice too!
			if err != nil {
				return nil, err
			}
			slice := slicesToLoad[sliceElementsRec.SliceId]
			slice.elements = append(slice.elements, sliceElement{
				rank:  sliceElementsRec.Rank,
				value: value,
			})
		}
	}

	return ws, nil
}

func (s *Session) SaveOrUpdate(ws *Worksheet) error {
	var count int
	if err := s.tx.
		Select("count(*)").
		From("worksheets").
		Where("id = $1", ws.Id()).
		QueryScalar(&count); err != nil {
		return err
	}

	if count == 0 {
		return s.Save(ws)
	} else {
		return s.Update(ws)
	}
}

func (s *Session) Save(ws *Worksheet) error {
	// insert rWorksheet
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

	// insert rValues
	var (
		slicesToInsert      []*slice
		worksheetsToCascade []*Worksheet
	)
	insertValues := s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	for index, value := range ws.data {
		insertValues.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: ws.Version(),
			ToVersion:   math.MaxInt32,
			Value:       value.String(),
		})

		switch v := value.(type) {
		case *slice:
			slicesToInsert = append(slicesToInsert, v)
		case *Worksheet:
			worksheetsToCascade = append(worksheetsToCascade, v)
		}
	}
	if _, err := insertValues.Exec(); err != nil {
		return err
	}

	if len(slicesToInsert) != 0 {
		insertSliceElements := s.tx.InsertInto("worksheet_slice_elements").Columns("*").Blacklist("id")
		for _, slice := range slicesToInsert {
			for _, element := range slice.elements {
				insertSliceElements.Record(rSliceElement{
					SliceId:     slice.id,
					Rank:        element.rank,
					FromVersion: ws.Version(),
					ToVersion:   math.MaxInt32,
					Value:       element.value.String(),
				})
			}
		}
		if _, err := insertSliceElements.Exec(); err != nil {
			return err
		}
	}

	for _, wsToCascade := range worksheetsToCascade {
		if err := s.SaveOrUpdate(wsToCascade); err != nil {
			return err
		}
	}

	// now we can update ws itself to reflect the save
	for index, value := range ws.data {
		ws.orig[index] = value
	}

	return nil
}

func (s *Session) Update(ws *Worksheet) error {
	oldVersion := ws.Version()
	newVersion := oldVersion + 1
	newVersionValue := MustNewValue(strconv.Itoa(newVersion))

	// diff
	diff := func() map[int]change {
		oldVersionValue := ws.data[IndexVersion]
		ws.data[IndexVersion] = MustNewValue(strconv.Itoa(newVersion))
		d := ws.diff()
		ws.data[IndexVersion] = oldVersionValue
		return d
	}()

	// no change, i.e. only the version would change
	if len(diff) == 1 {
		return nil
	}

	// split the diff into the various changes we need to do
	var (
		valuesToUpdate      = make([]int, 0, len(diff))
		slicesRanksOfDels   = make(map[string][]int)
		slicesElementsAdded = make(map[string][]sliceElement)
	)
	for index, change := range diff {
		valuesToUpdate = append(valuesToUpdate, index)
		if sliceBefore, ok := change.before.(*slice); ok {
			if sliceAfter, ok := change.after.(*slice); ok {
				if sliceBefore.id == sliceAfter.id {
					ranksOfDels, elementsAdded := diffSlices(sliceBefore, sliceAfter)

					sliceId := sliceBefore.id
					if len(ranksOfDels) != 0 {
						slicesRanksOfDels[sliceId] = ranksOfDels
					}
					if len(elementsAdded) != 0 {
						slicesElementsAdded[sliceId] = elementsAdded
					}
				}
			}
		}
	}

	// update old rValues
	if _, err := s.tx.
		Update("worksheet_values").
		Set("to_version", oldVersion).
		Where("worksheet_id = $1", ws.Id()).
		Where("from_version <= $1 and $1 <= to_version", oldVersion).
		Where(inClause("index", len(valuesToUpdate)), ughconvert(valuesToUpdate)...).
		Exec(); err != nil {
		return err
	}

	// insert new rValues
	insert := s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	for _, index := range valuesToUpdate {
		change := diff[index]
		insert.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: newVersion,
			ToVersion:   math.MaxInt32,
			Value:       change.after.String(),
		})
	}
	if _, err := insert.Exec(); err != nil {
		return err
	}

	// slices: deleted elements
	for sliceId, ranks := range slicesRanksOfDels {
		if _, err := s.tx.
			Update("worksheet_slice_elements").
			Set("to_version", oldVersion).
			Where("slice_id = $1", sliceId).
			Where("from_version <= $1 and $1 <= to_version", oldVersion).
			Where(inClause("rank", len(ranks)), ughconvert(ranks)...).
			Exec(); err != nil {
			return err
		}
	}

	// slices: added elements
	for sliceId, adds := range slicesElementsAdded {
		insert := s.tx.InsertInto("worksheet_slice_elements").Columns("*").Blacklist("id")
		for _, add := range adds {
			insert.Record(rSliceElement{
				SliceId:     sliceId,
				FromVersion: newVersion,
				ToVersion:   math.MaxInt32,
				Rank:        add.rank,
				Value:       add.value.String(),
			})
		}
		if _, err := insert.Exec(); err != nil {
			return err
		}
	}

	// update rWorksheet
	if result, err := s.tx.
		Update("worksheets").
		Set("version", newVersion).
		Where("id = $1 and version = $2", ws.Id(), oldVersion).
		Exec(); err != nil {
		return err
	} else if result.RowsAffected != 1 {
		return fmt.Errorf("concurrent update detected")
	}

	// now we can update ws itself to reflect the store
	ws.data[IndexVersion] = newVersionValue
	for index, value := range ws.data {
		ws.orig[index] = value
	}

	return nil
}

func inClause(column string, num int) string {
	vars := make([]string, num)
	for i := 0; i < num; i++ {
		vars[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("%s in (%s)", column, strings.Join(vars, ", "))
}

func ughconvert(ids []int) []interface{} {
	convert := make([]interface{}, len(ids))
	for i := range ids {
		convert[i] = ids[i]
	}
	return convert
}
