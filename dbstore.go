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
	"math"
	"strconv"
	"strings"

	"gopkg.in/mgutz/dat.v2/dat"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

// Store ... TODO(pascal): write about abstraction.
type Store interface {
	// Load loads the worksheet with identifier `id` from the store.
	Load(id string) (*Worksheet, error)

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
	Id          int64          `db:"id"`
	WorksheetId string         `db:"worksheet_id"`
	Index       int            `db:"index"`
	FromVersion int            `db:"from_version"`
	ToVersion   int            `db:"to_version"`
	Value       dat.NullString `db:"value"`
}

// rSliceElement represents a record of the worksheet_slice_elements table.
type rSliceElement struct {
	Id          int64          `db:"id"`
	SliceId     string         `db:"slice_id"`
	Rank        int            `db:"rank"`
	FromVersion int            `db:"from_version"`
	ToVersion   int            `db:"to_version"`
	Value       dat.NullString `db:"value"`
}

var tableToEntities = map[string]interface{}{
	"worksheets":               &rWorksheet{},
	"worksheet_values":         &rValue{},
	"worksheet_slice_elements": &rSliceElement{},
}

func (s *Session) Load(id string) (*Worksheet, error) {
	loader := &loader{
		s:               s,
		graph:           make(map[string]*Worksheet),
		slicesToHydrate: make(map[string]*slice),
	}
	return loader.loadWorksheet(id)
}

func (s *Session) SaveOrUpdate(ws *Worksheet) error {
	p := &persister{
		s:     s,
		graph: make(map[string]bool),
	}
	return p.saveOrUpdate(ws)
}

func (s *Session) Save(ws *Worksheet) error {
	p := &persister{
		s:     s,
		graph: make(map[string]bool),
	}
	return p.save(ws)
}

func (s *Session) Update(ws *Worksheet) error {
	p := &persister{
		s:     s,
		graph: make(map[string]bool),
	}
	return p.update(ws)
}

type loader struct {
	s               *Session
	graph           map[string]*Worksheet
	slicesToHydrate map[string]*slice
}

func (l *loader) loadWorksheet(id string) (*Worksheet, error) {
	if ws, ok := l.graph[id]; ok {
		return ws, nil
	}

	var wsRecs []rWorksheet
	if err := l.s.tx.
		Select("*").
		From("worksheets").
		Where("id = $1", id).
		QueryStructs(&wsRecs); err != nil {
		return nil, fmt.Errorf("unable to load worksheets records: %s", err)
	} else if len(wsRecs) == 0 {
		return nil, fmt.Errorf("unknown worksheet with id %s", id)
	}

	wsRec := wsRecs[0]

	ws, err := l.s.defs.newUninitializedWorksheet(wsRec.Name)
	if err != nil {
		return nil, err
	}

	l.graph[id] = ws

	var valuesRecs []rValue
	if err := l.s.tx.
		Select("*").
		From("worksheet_values").
		Where("worksheet_id = $1", id).
		Where("from_version <= $1 and $1 <= to_version", wsRec.Version).
		QueryStructs(&valuesRecs); err != nil {
		return nil, err
	}
	for _, valueRec := range valuesRecs {
		index := valueRec.Index

		// field
		field, ok := ws.def.fieldsByIndex[index]
		if !ok {
			return nil, fmt.Errorf("unknown value with field index %d", index)
		}

		// load, and potentially defer hydration of value
		if valueRec.Value.Valid {
			value, err := l.readValue(field.typ, valueRec.Value)
			if err != nil {
				return nil, err
			}

			// set orig and data
			ws.orig[index] = value
			ws.data[index] = value
		}
	}

	for {
		slicesToHydrate := l.nextSlicesToHydrate()
		if len(slicesToHydrate) == 0 {
			break
		}
		slicesIds := make([]interface{}, len(slicesToHydrate))
		for _, slice := range slicesToHydrate {
			slicesIds = append(slicesIds, slice.id)
		}
		var sliceElementsRecs []rSliceElement
		err = l.s.tx.
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
			slice := slicesToHydrate[sliceElementsRec.SliceId]
			value, err := l.readValue(slice.typ.elementType, sliceElementsRec.Value)
			if err != nil {
				return nil, err
			}
			slice.elements = append(slice.elements, sliceElement{
				rank:  sliceElementsRec.Rank,
				value: value,
			})
		}
	}

	return ws, nil
}

func (l *loader) readValue(typ Type, optValue dat.NullString) (Value, error) {
	if !optValue.Valid {
		return &Undefined{}, nil
	}

	value := optValue.String
	switch t := typ.(type) {
	case *tTextType:
		return NewText(value), nil
	case *SliceType:
		if !strings.HasPrefix(value, "[:") {
			return nil, fmt.Errorf("unreadable value for slice %s", value)
		}
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("unreadable value for slice %s", value)
		}
		lastRank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("unreadable value for slice %s", value)
		}
		slice := newSliceWithIdAndLastRank(t, parts[2], lastRank)
		l.slicesToHydrate[slice.id] = slice
		return slice, nil
	case *Definition:
		if !strings.HasPrefix(value, "*:") {
			return nil, fmt.Errorf("unreadable value for ref %s", value)
		}
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("unreadable value for ref %s", value)
		}
		value, err := l.loadWorksheet(parts[1])
		if err != nil {
			return nil, fmt.Errorf("unable to load referenced worksheet %s: %s", parts[1], err)
		}
		return value, nil
	default:
		return NewValue(value)
	}
}

func (l *loader) nextSlicesToHydrate() map[string]*slice {
	slicesToHydrate := l.slicesToHydrate
	l.slicesToHydrate = make(map[string]*slice)
	return slicesToHydrate
}

type persister struct {
	s     *Session
	graph map[string]bool
}

func (p *persister) saveOrUpdate(ws *Worksheet) error {
	var count int
	if err := p.s.tx.
		Select("count(*)").
		From("worksheets").
		Where("id = $1", ws.Id()).
		QueryScalar(&count); err != nil {
		return err
	}

	if count == 0 {
		return p.save(ws)
	} else {
		return p.update(ws)
	}
}

func (p *persister) save(ws *Worksheet) error {
	// already done?
	if _, ok := p.graph[ws.Id()]; ok {
		return nil
	}
	p.graph[ws.Id()] = true

	// cascade worksheets
	for _, value := range ws.data {
		for _, wsToCascade := range worksheetsToCascade(value) {
			if err := p.saveOrUpdate(wsToCascade); err != nil {
				return err
			}
		}
	}

	// insert rWorksheet
	_, err := p.s.tx.
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
	var slicesToInsert []*slice
	insertValues := p.s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	for index, value := range ws.data {
		insertValues.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: ws.Version(),
			ToVersion:   math.MaxInt32,
			Value:       p.writeValue(value),
		})

		if slice, ok := value.(*slice); ok {
			slicesToInsert = append(slicesToInsert, slice)
		}
	}
	if _, err := insertValues.Exec(); err != nil {
		return err
	}

	if len(slicesToInsert) != 0 {
		insertSliceElements := p.s.tx.InsertInto("worksheet_slice_elements").Columns("*").Blacklist("id")
		for _, slice := range slicesToInsert {
			for _, element := range slice.elements {
				insertSliceElements.Record(rSliceElement{
					SliceId:     slice.id,
					Rank:        element.rank,
					FromVersion: ws.Version(),
					ToVersion:   math.MaxInt32,
					Value:       p.writeValue(element.value),
				})
			}
		}
		if _, err := insertSliceElements.Exec(); err != nil {
			return err
		}
	}

	// now we can update ws itself to reflect the save
	for index, value := range ws.data {
		ws.orig[index] = value
	}

	return nil
}

func (p *persister) update(ws *Worksheet) error {
	// already done?
	if _, ok := p.graph[ws.Id()]; ok {
		return nil
	}
	p.graph[ws.Id()] = true

	// cascade worksheets
	for _, value := range ws.data {
		for _, wsToCascade := range worksheetsToCascade(value) {
			if err := p.saveOrUpdate(wsToCascade); err != nil {
				return err
			}
		}
	}

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
		if sliceAfter, ok := change.after.(*slice); ok {
			var sliceBefore *slice
			if actualSliceBefore, ok := change.before.(*slice); ok {
				sliceBefore = actualSliceBefore
			} else if _, ok := change.before.(*Undefined); ok {
				sliceBefore = &slice{id: sliceAfter.id}
			} else {
				continue
			}
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

	// update old rValues
	if _, err := p.s.tx.
		Update("worksheet_values").
		Set("to_version", oldVersion).
		Where("worksheet_id = $1", ws.Id()).
		Where("from_version <= $1 and $1 <= to_version", oldVersion).
		Where(inClause("index", len(valuesToUpdate)), ughconvert(valuesToUpdate)...).
		Exec(); err != nil {
		return err
	}

	// insert new rValues
	insert := p.s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	for _, index := range valuesToUpdate {
		change := diff[index]
		insert.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: newVersion,
			ToVersion:   math.MaxInt32,
			Value:       p.writeValue(change.after),
		})
	}
	if _, err := insert.Exec(); err != nil {
		return err
	}

	// slices: deleted elements
	for sliceId, ranks := range slicesRanksOfDels {
		if _, err := p.s.tx.
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
		insert := p.s.tx.InsertInto("worksheet_slice_elements").Columns("*").Blacklist("id")
		for _, add := range adds {
			insert.Record(rSliceElement{
				SliceId:     sliceId,
				FromVersion: newVersion,
				ToVersion:   math.MaxInt32,
				Rank:        add.rank,
				Value:       p.writeValue(add.value),
			})
		}
		if _, err := insert.Exec(); err != nil {
			return err
		}
	}

	// update rWorksheet
	if result, err := p.s.tx.
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

func (p *persister) writeValue(value Value) dat.NullString {
	if _, ok := value.(*Undefined); ok {
		return dat.NullString{sql.NullString{"", false}}
	}

	var result string
	switch v := value.(type) {
	case *Text:
		result = v.value
	case *slice:
		result = fmt.Sprintf("[:%d:%s", v.lastRank, v.id)
	case *Worksheet:
		result = fmt.Sprintf("*:%s", v.Id())
	default:
		result = value.String()
	}
	return dat.NullStringFrom(result)
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

func worksheetsToCascade(value Value) []*Worksheet {
	switch v := value.(type) {
	case *Worksheet:
		return []*Worksheet{v}
	case *slice:
		var result []*Worksheet
		for _, element := range v.elements {
			result = append(result, worksheetsToCascade(element.value)...)
		}
		return result
	default:
		return nil
	}
}
