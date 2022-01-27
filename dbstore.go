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
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"

	runner "github.com/homelight/dat/sqlx-runner"
)

// Store ... TODO(pascal): write about abstraction.
type Store interface {
	// Load loads the worksheet with identifier `id` from the store.
	Load(id string) (*Worksheet, error)
	LoadContext(ctx context.Context, id string) (*Worksheet, error)

	// SaveOrUpdate saves or updates a worksheet to the store. On success,
	// returns an edit identifier.
	SaveOrUpdate(ws *Worksheet) (string, error)
	SaveOrUpdateContext(ctx context.Context, ws *Worksheet) (string, error)

	// Save saves a new worksheet to the store. On success, returns an edit
	// identifier.
	Save(ws *Worksheet) (string, error)
	SaveContext(ctx context.Context, ws *Worksheet) (string, error)

	// Update updates an existing worksheet in the store. On success, returns an
	// edit identifier.
	Update(ws *Worksheet) (string, error)
	UpdateContext(ctx context.Context, ws *Worksheet) (string, error)

	// Edit returns a specific edit, the time at which the edit occurred, and all
	// worksheets modified as a map of their ids to the resulting version.
	Edit(editId string) (time.Time, map[string]int, error)
	EditContext(ctx context.Context, editId string) (time.Time, map[string]int, error)
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
		clock:   &realClock{},
	}
}

type clock interface {
	// now returns the current time as a Unix time, the number of nanoseconds
	// elapsed since January 1, 1970 UTC.
	nowAsUnixNano() int64
}

type realClock struct{}

// Assert realClock implements the clock interface.
var _ clock = &realClock{}

func (_ *realClock) nowAsUnixNano() int64 {
	return time.Now().UnixNano()
}

// Session is the ... TODO(pascal): write
type Session struct {
	*DbStore
	tx    *runner.Tx
	clock clock
}

// Assert Session implements Store interface.
var _ Store = &Session{}

// rWorksheet represents a record of the worksheets table.
type rWorksheet struct {
	Id      string `db:"id"`
	Version int    `db:"version"`
	Name    string `db:"name"`
}

// rEdit represents a record of the worksheet_edits table.
type rEdit struct {
	EditId      string `db:"edit_id"`
	CreatedAt   int64  `db:"created_at"`
	WorksheetId string `db:"worksheet_id"`
	ToVersion   int    `db:"to_version"`
}

// rValue represents a record of the worksheet_values table.
type rValue struct {
	Id          int64   `db:"id"`
	WorksheetId string  `db:"worksheet_id"`
	Index       int     `db:"index"`
	FromVersion int     `db:"from_version"`
	ToVersion   int     `db:"to_version"`
	Value       *string `db:"value"`
}

// rParent represents a record of the worksheet_parents table.
type rParent struct {
	ChildId          string `db:"child_id"`
	ParentId         string `db:"parent_id"`
	ParentFieldIndex int    `db:"parent_field_index"`
}

// rSliceElement represents a record of the worksheet_slice_elements table.
type rSliceElement struct {
	Id          int64   `db:"id"`
	SliceId     string  `db:"slice_id"`
	Rank        int     `db:"rank"`
	FromVersion int     `db:"from_version"`
	ToVersion   int     `db:"to_version"`
	Value       *string `db:"value"`
}

var tableToEntities = map[string]interface{}{
	"worksheets":               &rWorksheet{},
	"worksheet_edits":          &rEdit{},
	"worksheet_values":         &rValue{},
	"worksheet_parents":        &rParent{},
	"worksheet_slice_elements": &rSliceElement{},
}

func (s *Session) Edit(editId string) (time.Time, map[string]int, error) {
	return s.editCommon(context.Background(), editId)
}

func (s *Session) EditContext(ctx context.Context, editId string) (time.Time, map[string]int, error) {
	return s.editCommon(ctx, editId)
}

func (s *Session) editCommon(ctx context.Context, editId string) (time.Time, map[string]int, error) {
	var editRecs []rEdit
	if err := s.tx.
		Select("*").
		From("worksheet_edits").
		Where("edit_id = $1", editId).
		QueryStructs(&editRecs); err != nil {
		return time.Time{}, nil, err
	}
	if len(editRecs) == 0 {
		return time.Time{}, nil, fmt.Errorf("unknown edit %s", editId)
	}

	// By construction, all rEdit are set to the exact same time, hence choosing
	// arbitrarily the first is safe.
	createdAt := time.Unix(0, editRecs[0].CreatedAt)

	touchedWs := make(map[string]int, len(editRecs))
	for _, editRec := range editRecs {
		touchedWs[editRec.WorksheetId] = editRec.ToVersion
	}

	return createdAt, touchedWs, nil
}

func (s *Session) Load(id string) (*Worksheet, error) {
	return s.loadCommon(context.Background(), id)
}

func (s *Session) LoadContext(ctx context.Context, id string) (*Worksheet, error) {
	return s.loadCommon(ctx, id)
}

func (s *Session) loadCommon(ctx context.Context, id string) (*Worksheet, error) {
	loader := &loader{
		s:               s,
		graph:           make(map[string]*Worksheet),
		slicesToHydrate: make(map[string]slicepair),
	}
	return loader.loadWorksheet(id)
}

func (s *Session) newPersister() *persister {
	return &persister{
		editId:    uuid.Must(uuid.NewV4()).String(),
		createdAt: s.clock.nowAsUnixNano(),
		s:         s,
		graph:     make(map[string]bool),
	}
}

func (s *Session) SaveOrUpdate(ws *Worksheet) (string, error) {
	return s.saveOrUpdateCommon(context.Background(), ws)
}

func (s *Session) SaveOrUpdateContext(ctx context.Context, ws *Worksheet) (string, error) {
	return s.saveOrUpdateCommon(ctx, ws)
}

func (s *Session) saveOrUpdateCommon(ctx context.Context, ws *Worksheet) (string, error) {
	p := s.newPersister()
	if err := p.saveOrUpdate(ctx, ws); err != nil {
		return "", err
	}
	return p.editId, nil
}

func (s *Session) Save(ws *Worksheet) (string, error) {
	return s.saveCommon(context.Background(), ws)
}

func (s *Session) SaveContext(ctx context.Context, ws *Worksheet) (string, error) {
	return s.saveCommon(ctx, ws)
}

func (s *Session) saveCommon(ctx context.Context, ws *Worksheet) (string, error) {
	p := s.newPersister()
	if err := p.save(ctx, ws); err != nil {
		return "", err
	}
	return p.editId, nil
}

func (s *Session) Update(ws *Worksheet) (string, error) {
	return s.updateCommon(context.Background(), ws)
}

func (s *Session) UpdateContext(ctx context.Context, ws *Worksheet) (string, error) {
	return s.updateCommon(ctx, ws)
}

func (s *Session) updateCommon(ctx context.Context, ws *Worksheet) (string, error) {
	p := s.newPersister()
	if err := p.update(ctx, ws); err != nil {
		return "", err
	}
	return p.editId, nil
}

type slicepair struct {
	orig *Slice
	data *Slice
}

type loader struct {
	s               *Session
	graph           map[string]*Worksheet
	slicesToHydrate map[string]slicepair
}

func (l *loader) loadWorksheet(id string) (*Worksheet, error) {
	// Early exit for worksheets we are already in the process of loading.
	// Important to note that the returned worksheet may be only partially
	// hydrated. Callers beware.
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

	// Before placing the worksheet in the graph, we set the id manually so
	// callers can rely on this even if the worksheet itself is not fully
	// loaded.
	ws.data[indexId] = NewText(id)
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
			continue // skip deprecated fields
		}

		// load, and potentially defer hydration of value
		if valueRec.Value != nil {
			orig, current, err := l.dbReadValue(field.typ, valueRec.Value)
			if err != nil {
				return nil, err
			}

			// set orig and data
			ws.orig[index] = orig
			ws.data[index] = current
		}
	}

	// hydrate slices
	for {
		slicesToHydrate := l.nextSlicesToHydrate()
		if len(slicesToHydrate) == 0 {
			break
		}
		slicesIds := make([]interface{}, len(slicesToHydrate))
		for sliceId := range slicesToHydrate {
			slicesIds = append(slicesIds, sliceId)
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
			slices := slicesToHydrate[sliceElementsRec.SliceId]
			orig, data, err := l.dbReadValue(slices.data.typ.elementType, sliceElementsRec.Value)
			if err != nil {
				return nil, err
			}
			slices.orig.elements = append(slices.orig.elements, sliceElement{
				rank:  sliceElementsRec.Rank,
				value: orig,
			})
			slices.data.elements = append(slices.data.elements, sliceElement{
				rank:  sliceElementsRec.Rank,
				value: data,
			})
		}
	}

	// load parents
	var parentsRecs []rParent
	if err := l.s.tx.
		Select("*").
		From("worksheet_parents").
		Where("child_id = $1", id).
		QueryStructs(&parentsRecs); err != nil {
		return nil, err
	}
	for _, parentRec := range parentsRecs {
		parentWs, err := l.loadWorksheet(parentRec.ParentId)
		if err != nil {
			return nil, err
		}
		ws.parents.addParentViaFieldIndex(parentWs, parentRec.ParentFieldIndex)
	}

	return ws, nil
}

func (l *loader) dbReadValue(typ Type, optValue *string) (Value, Value, error) {
	if optValue == nil {
		return vUndefined, vUndefined, nil
	}
	return typ.dbReadValue(l, *optValue)
}

func (typ *UndefinedType) dbReadValue(l *loader, value string) (Value, Value, error) {
	panic("should never be called")
}

func (typ *TextType) dbReadValue(l *loader, value string) (Value, Value, error) {
	text := NewText(value)
	return text, text, nil
}

func (typ *BoolType) dbReadValue(l *loader, value string) (Value, Value, error) {
	switch value {
	case "true":
		return vTrue, vTrue, nil
	case "false":
		return vFalse, vFalse, nil
	default:
		return nil, nil, fmt.Errorf("unreadable value for bool %s", value)
	}
}

func (typ *NumberType) dbReadValue(l *loader, value string) (Value, Value, error) {
	num, err := NewNumberFromString(value)
	if err != nil {
		return nil, nil, err
	}
	return num, num, nil
}

// Slices ref syntax
//
//     [:<rank>:<slice_uuid>
var sliceRefRegex = regexp.MustCompile(`\[\:([0-9]+)\:(.*)`)

func (typ *SliceType) dbReadValue(l *loader, value string) (Value, Value, error) {
	match := sliceRefRegex.FindStringSubmatch(value)
	if len(match) != 3 {
		return nil, nil, fmt.Errorf("unreadable value for slice %s", value)
	}

	lastRank, err := strconv.Atoi(match[1])
	if err != nil {
		panic(fmt.Sprintf("unexpected: %s", match[2]))
	}
	sliceId := match[2]

	orig := newSliceWithIdAndLastRank(typ, sliceId, lastRank)
	data := newSliceWithIdAndLastRank(typ, sliceId, lastRank)
	l.slicesToHydrate[sliceId] = slicepair{
		orig: orig,
		data: data,
	}

	return orig, data, nil
}

// Worksheet ref syntax
//
//     *:<ws_uuid>
//     *:<ws_uuid>@<version>
var wsRefRegex = regexp.MustCompile(`\*\:([^@]*)(@([0-9]+))?`)

func (typ *Definition) dbReadValue(l *loader, value string) (Value, Value, error) {
	match := wsRefRegex.FindStringSubmatch(value)
	if len(match) != 4 {
		return nil, nil, fmt.Errorf("unreadable value for ref %s", value)
	}

	wsId := match[1]

	ws, err := l.loadWorksheet(wsId)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load referenced worksheet %s: %s", match[0], err)
	}

	var wsVersion int
	if match[3] != "" {
		var err error
		wsVersion, err = strconv.Atoi(match[3])
		if err != nil {
			panic("unexpected")
		}
	} else {
		wsVersion = -1 // i.e. unknown, will force the ref to be updated
	}

	return &wsRefAtVersion{ws, wsVersion}, ws, nil
}

func (typ *EnumType) dbReadValue(l *loader, value string) (Value, Value, error) {
	return (&TextType{}).dbReadValue(l, value)
}

func (l *loader) nextSlicesToHydrate() map[string]slicepair {
	slicesToHydrate := l.slicesToHydrate
	l.slicesToHydrate = make(map[string]slicepair)
	return slicesToHydrate
}

type persister struct {
	editId    string
	createdAt int64
	s         *Session
	graph     map[string]bool
}

func (p *persister) saveOrUpdate(ctx context.Context, ws *Worksheet) error {
	var count int
	if err := p.s.tx.
		Select("count(*)").
		From("worksheets").
		Where("id = $1", ws.Id()).
		QueryScalar(&count); err != nil {
		return err
	}

	if count == 0 {
		return p.save(ctx, ws)
	} else {
		return p.update(ctx, ws)
	}
}

func (p *persister) save(ctx context.Context, ws *Worksheet) error {
	// already done?
	if _, ok := p.graph[ws.Id()]; ok {
		return nil
	}
	p.graph[ws.Id()] = true

	// cascade updates to children and parents
	for _, value := range ws.data {
		for _, childWs := range extractChildWs(value) {
			if err := p.saveOrUpdate(ctx, childWs); err != nil {
				return err
			}
		}
	}
	for _, byParentFieldIndex := range ws.parents {
		for _, byParentId := range byParentFieldIndex {
			for _, parentWs := range byParentId {
				if err := p.saveOrUpdate(ctx, parentWs); err != nil {
					return err
				}
			}
		}
	}

	// insert rWorksheet
	if _, err := p.s.tx.
		InsertInto("worksheets").
		Columns("*").
		Record(&rWorksheet{
			Id:      ws.Id(),
			Version: ws.Version(),
			Name:    ws.Name(),
		}).
		ExecContext(ctx); err != nil {
		return err
	}

	// insert rEdit
	if _, err := p.s.tx.
		InsertInto("worksheet_edits").
		Columns("*").
		Record(&rEdit{
			EditId:      p.editId,
			CreatedAt:   p.createdAt,
			WorksheetId: ws.Id(),
			ToVersion:   ws.Version(),
		}).
		ExecContext(ctx); err != nil {
		return err
	}

	// adopted children
	adoptedChildren := make(map[int][]string)

	// insert rValues
	var slicesToInsert []*Slice
	insertValues := p.s.tx.InsertInto("worksheet_values").Columns("*").Blacklist("id")
	for index, value := range ws.data {
		insertValues.Record(rValue{
			WorksheetId: ws.Id(),
			Index:       index,
			FromVersion: ws.Version(),
			ToVersion:   math.MaxInt32,
			Value:       dbWriteValue(value),
		})

		if slice, ok := value.(*Slice); ok {
			slicesToInsert = append(slicesToInsert, slice)
			for _, elem := range slice.elements {
				for _, childWs := range extractChildWs(elem.value) {
					adoptedChildren[index] = append(adoptedChildren[index], childWs.Id())
				}
			}
		} else {
			for _, childWs := range extractChildWs(value) {
				adoptedChildren[index] = append(adoptedChildren[index], childWs.Id())
			}
		}
	}
	if _, err := insertValues.ExecContext(ctx); err != nil {
		return err
	}

	// insert rSliceElement
	if len(slicesToInsert) != 0 {
		atLeastOneSliceElementExists := false
		insertSliceElements := p.s.tx.InsertInto("worksheet_slice_elements").Columns("*").Blacklist("id")
		for _, slice := range slicesToInsert {
			for _, element := range slice.elements {
				atLeastOneSliceElementExists = true
				insertSliceElements.Record(rSliceElement{
					SliceId:     slice.id,
					Rank:        element.rank,
					FromVersion: ws.Version(),
					ToVersion:   math.MaxInt32,
					Value:       dbWriteValue(element.value),
				})
			}
		}
		if atLeastOneSliceElementExists {
			if _, err := insertSliceElements.ExecContext(ctx); err != nil {
				return err
			}
		}
	}

	// insert rParent
	if len(adoptedChildren) != 0 {
		insertParentElements := p.s.tx.InsertInto("worksheet_parents").Columns("*")
		for index, childrenWsId := range adoptedChildren {
			for _, childId := range childrenWsId {
				insertParentElements.Record(rParent{
					ChildId:          childId,
					ParentId:         ws.Id(),
					ParentFieldIndex: index,
				})
			}
		}
		if _, err := insertParentElements.ExecContext(ctx); err != nil {
			return err
		}
	}

	// now we can update ws itself to reflect the save
	for index, value := range ws.data {
		ws.orig[index] = toOrig(value)
	}

	return nil
}

func (p *persister) update(ctx context.Context, ws *Worksheet) error {
	// already done?
	if _, ok := p.graph[ws.Id()]; ok {
		return nil
	}
	p.graph[ws.Id()] = true

	// cascade updates to children and parents
	for _, value := range ws.data {
		for _, childWs := range extractChildWs(value) {
			if err := p.saveOrUpdate(ctx, childWs); err != nil {
				return err
			}
		}
	}
	for _, byParentFieldIndex := range ws.parents {
		for _, byParentId := range byParentFieldIndex {
			for _, parentWs := range byParentId {
				if err := p.saveOrUpdate(ctx, parentWs); err != nil {
					return err
				}
			}
		}
	}

	oldVersion := ws.Version()
	newVersion := oldVersion + 1

	// diff
	ws.set(ws.def.fieldsByIndex[indexVersion], &Number{int64(newVersion), &NumberType{0}})
	diff := ws.diff()

	// plan rollback
	hasFailed := true
	defer func() {
		if hasFailed {
			ws.set(ws.def.fieldsByIndex[indexVersion], &Number{int64(oldVersion), &NumberType{0}})
		}
	}()

	// no change, i.e. only the version would change
	if len(diff) == 1 {
		return nil
	}

	// split the diff into the various changes we need to do
	var (
		valuesToUpdate        = make([]int, 0, len(diff))
		slicesElementsDeleted = make(map[string][]sliceElement)
		slicesElementsAdded   = make(map[string][]sliceElement)
		orphanedChildren      = make(map[int][]interface{})
		adoptedChildren       = make(map[int][]string)
	)
	for index, change := range diff {
		shouldUpdateValueRecord := true

		// non-slice values
		if _, ok := change.before.(*Slice); !ok {
			for _, childWs := range extractChildWs(change.before) {
				orphanedChildren[index] = append(orphanedChildren[index], childWs.Id())
			}
		}
		if _, ok := change.after.(*Slice); !ok {
			for _, childWs := range extractChildWs(change.after) {
				adoptedChildren[index] = append(adoptedChildren[index], childWs.Id())
			}
		}

		// slice values
		if sliceAfter, ok := change.after.(*Slice); ok {
			var sliceBefore *Slice
			if actualSliceBefore, ok := change.before.(*Slice); ok {
				sliceBefore = actualSliceBefore
			} else if _, ok := change.before.(*Undefined); ok {
				sliceBefore = &Slice{id: sliceAfter.id}
			} else {
				panic(fmt.Sprintf("unexpected: before=%s, after%s", change.before, change.after))
			}
			if sliceBefore.id == sliceAfter.id {
				shouldUpdateValueRecord = sliceBefore.lastRank != sliceAfter.lastRank

				sliceChange := diffSlices(sliceBefore, sliceAfter)

				sliceId := sliceBefore.id
				if len(sliceChange.deleted) != 0 {
					slicesElementsDeleted[sliceId] = sliceChange.deleted
					for _, del := range sliceChange.deleted {
						for _, childWs := range extractChildWs(del.value) {
							orphanedChildren[index] = append(orphanedChildren[index], childWs.Id())
						}
					}
				}
				if len(sliceChange.added) != 0 {
					slicesElementsAdded[sliceId] = sliceChange.added
					for _, add := range sliceChange.added {
						for _, childWs := range extractChildWs(add.value) {
							adoptedChildren[index] = append(adoptedChildren[index], childWs.Id())
						}
					}
				}
			} else {
				panic("unexpected: changing slice id not supported yet")
			}
		}
		if shouldUpdateValueRecord {
			valuesToUpdate = append(valuesToUpdate, index)
		}
	}

	// no change, i.e. only the version would change
	//
	// Right now, we have to redo this check after going through full diffing
	// for the case where the only diff was a slice change, optimized away
	// because the rank and id didn't change. Doing this late check, and in a
	// relatively special purpose way, will make adding additional features
	// (e.g.bag, maps) harder. Instead, we should push more logic to the diffing
	// algorithm, so that operations at the store level are a more straighforward
	// application of recording those diffs (vs computing them). Something to
	// refactor soon!
	if len(valuesToUpdate) == 1 &&
		len(slicesElementsAdded) == 0 &&
		len(slicesElementsDeleted) == 0 {
		return nil
	}

	// insert rEdit
	_, err := p.s.tx.
		InsertInto("worksheet_edits").
		Columns("*").
		Record(&rEdit{
			EditId:      p.editId,
			CreatedAt:   p.createdAt,
			WorksheetId: ws.Id(),
			ToVersion:   newVersion,
		}).
		ExecContext(ctx)
	if isSpecificUniqueConstraintErr(err, "worksheet_edits_worksheet_id_to_version_key") {
		return fmt.Errorf("concurrent update detected (%s)", err)
	} else if err != nil {
		return err
	}

	// update old rValues
	if _, err := p.s.tx.
		Update("worksheet_values").
		Set("to_version", oldVersion).
		Where("worksheet_id = $1", ws.Id()).
		Where("from_version <= $1 and $1 <= to_version", oldVersion).
		Where(inClause("index", len(valuesToUpdate)), ughconvert(valuesToUpdate)...).
		ExecContext(ctx); err != nil {
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
			Value:       dbWriteValue(change.after),
		})
	}
	if _, err := insert.ExecContext(ctx); err != nil {
		return err
	}

	// slices: deleted elements
	for sliceId, dels := range slicesElementsDeleted {
		var ranks []interface{}
		for _, del := range dels {
			ranks = append(ranks, del.rank)
		}
		if _, err := p.s.tx.
			Update("worksheet_slice_elements").
			Set("to_version", oldVersion).
			Where("slice_id = $1", sliceId).
			Where("from_version <= $1 and $1 <= to_version", oldVersion).
			Where(inClause("rank", len(ranks)), ranks...).
			ExecContext(ctx); err != nil {
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
				Value:       dbWriteValue(add.value),
			})
		}
		if _, err := insert.ExecContext(ctx); err != nil {
			return err
		}
	}

	// update rParent
	for index, childrenWsId := range orphanedChildren {
		if _, err := p.s.tx.DeleteFrom("worksheet_parents").
			Where("parent_id = $1", ws.Id()).
			Where("parent_field_index = $1", index).
			Where(inClause("child_id", len(childrenWsId)), childrenWsId...).
			ExecContext(ctx); err != nil {
			return err
		}
	}
	if len(adoptedChildren) != 0 {
		insertParentElements := p.s.tx.InsertInto("worksheet_parents").Columns("*")
		for index, childrenWsId := range adoptedChildren {
			for _, childId := range childrenWsId {
				insertParentElements.Record(rParent{
					ChildId:          childId,
					ParentId:         ws.Id(),
					ParentFieldIndex: index,
				})
			}
		}
		if _, err := insertParentElements.ExecContext(ctx); err != nil {
			return err
		}
	}

	// update rWorksheet
	if result, err := p.s.tx.
		Update("worksheets").
		Set("version", newVersion).
		Where("id = $1 and version = $2", ws.Id(), oldVersion).
		ExecContext(ctx); err != nil {
		return err
	} else if result.RowsAffected != 1 {
		return fmt.Errorf("concurrent update detected")
	}

	// now we can update ws itself to reflect the store
	for index, value := range ws.data {
		ws.orig[index] = toOrig(value)
	}

	hasFailed = false
	return nil
}

func toOrig(value Value) Value {
	// TODO(pascal): We need to recursively convert, e.g. handle slices. Not
	// doing this today simplifies the persistence code, at the cost of missing
	// some ref updates.
	//
	// See also slices' diffCompare.
	switch v := value.(type) {
	case *Worksheet:
		return &wsRefAtVersion{v, v.Version()}
	default:
		return value
	}
}

func dbWriteValue(value Value) *string {
	if _, ok := value.(*Undefined); ok {
		return nil
	}

	result := value.dbWriteValue()
	return &result
}

func (value *Undefined) dbWriteValue() string {
	panic("should never be called")
}

func (value *Text) dbWriteValue() string {
	return value.value
}

func (value *Number) dbWriteValue() string {
	return value.String()
}

func (value *Bool) dbWriteValue() string {
	return value.String()
}

func (value *Slice) dbWriteValue() string {
	return fmt.Sprintf("[:%d:%s", value.lastRank, value.id)
}

func (value *Worksheet) dbWriteValue() string {
	return fmt.Sprintf("*:%s@%d", value.Id(), value.Version())
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

func isSpecificUniqueConstraintErr(err error, uniqueConstraintName string) bool {
	// Did we violate the unique constraint?
	// See https://www.postgresql.org/docs/9.4/static/errcodes-appendix.html
	switch err := err.(type) {
	case *pq.Error:
		return err.Code == pq.ErrorCode("23505") &&
			strings.Contains(err.Error(), uniqueConstraintName)
	default:
		return false
	}
}
