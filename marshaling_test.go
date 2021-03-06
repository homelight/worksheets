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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestMarshaling_simple() {
	ws := s.defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustSet("text", NewText(`some text with " and stuff`))
	ws.MustSet("bool", NewBool(true))
	ws.MustSet("num_0", NewNumberFromInt(123))
	ws.MustSet("num_2", NewNumberFromFloat64(123.45))
	ws.MustSet("undefined", vUndefined)

	expected := `{"the-id":{
		"text": "some text with \" and stuff",
		"bool": true,
		"num_0": "123",
		"num_2": "123.45",
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfText() {
	ws := s.defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustAppend("slice_t", alice)
	ws.MustAppend("slice_t", bob)

	expected := `{"the-id":{
		"slice_t": ["Alice", "Bob"],
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceWithUndefined() {
	ws := s.defs.MustNewWorksheet("all_types")
	forciblySetId(ws, "the-id")
	ws.MustAppend("slice_t", vUndefined)
	ws.MustAppend("slice_t", bob)

	expected := `{"the-id":{
		"slice_t": [null, "Bob"],
		"id": "the-id",
		"version":"1"
	}}`
	actual, err := json.Marshal(ws)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_wsRef() {
	parent := s.defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	child := s.defs.MustNewWorksheet("all_types")
	forciblySetId(child, "the-child")

	parent.MustSet("ws", child)

	expected := `{
	"the-parent":{
		"ws": "the-child",
		"id": "the-parent",
		"version":"1"
	},
	"the-child":{
		"id": "the-child",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_wsRefToItself() {
	parent := s.defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent-and-child")

	parent.MustSet("ws", parent)

	expected := `{"the-parent-and-child":{
		"ws": "the-parent-and-child",
		"id": "the-parent-and-child",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfRefs() {
	parent := s.defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	child1 := s.defs.MustNewWorksheet("all_types")
	forciblySetId(child1, "the-child1")

	child2 := s.defs.MustNewWorksheet("all_types")
	forciblySetId(child2, "the-child2")

	parent.MustAppend("slice_ws", child1)
	parent.MustAppend("slice_ws", child2)

	expected := `{
	"the-parent":{
		"slice_ws": ["the-child1", "the-child2"],
		"id": "the-parent",
		"version":"1"
	},
	"the-child1":{
		"id": "the-child1",
		"version":"1"
	},
	"the-child2":{
		"id": "the-child2",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) TestMarshaling_sliceOfRefsToItself() {
	parent := s.defs.MustNewWorksheet("all_types")
	forciblySetId(parent, "the-parent")

	parent.MustAppend("slice_ws", parent)
	parent.MustAppend("slice_ws", parent)

	expected := `{"the-parent":{
		"slice_ws": ["the-parent", "the-parent"],
		"id": "the-parent",
		"version":"1"
	}}`
	actual, err := json.Marshal(parent)
	require.NoError(s.T(), err)
	s.requireSameJson(expected, actual)
}

func (s *Zuite) requireSameJson(expected string, actual []byte) {
	var e, a interface{}

	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		require.Fail(s.T(), "bad expected JSON", expected)
	}
	if err := json.Unmarshal(actual, &a); err != nil {
		require.Fail(s.T(), "bad actual JSON", actual)
	}
	require.Equal(s.T(), e, a)
}

func (s *Zuite) TestStructScan_onlyStarStruct() {
	ws := s.defs.MustNewWorksheet("all_types")
	err := ws.StructScan("")
	require.EqualError(s.T(), err, "dest must be a *struct")
}

func (s *Zuite) TestStructScan_emptyTagName() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Text string `ws:""`
	}
	err := ws.StructScan(&data)
	require.EqualError(s.T(), err, "struct field Text: cannot have empty tag name")
}

func (s *Zuite) TestStructScan_badTagName() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Text string `ws:"abc"`
	}
	err := ws.StructScan(&data)
	require.EqualError(s.T(), err, "struct field Text: unknown ws field abc")
}

func (s *Zuite) TestStructScan_defaultMappings() {
	ws := s.defs.MustNewWorksheet("DefaultMappingsTest")
	ws.MustSet("Name", NewText("pedro"))
	ws.MustSet("Age", NewNumberFromInt(28))
	childWs := s.defs.MustNewWorksheet("DefaultMappingsTest")
	childWs.MustSet("Name", NewText("alicia"))
	childWs.MustSet("Age", NewNumberFromInt(1))
	ws.MustSet("Child", childWs)

	type person struct {
		Name  string
		Age   int
		Child *person
	}

	var data person
	err := ws.StructScan(&data)
	s.Require().NoError(err)
	s.Equal("pedro", data.Name)
	s.Equal(28, data.Age)
	s.Require().NotNil(data.Child)
	s.Equal("alicia", data.Child.Name)
	s.Equal(1, data.Child.Age)
}

func (s *Zuite) TestStructScan_defaultMappingsIgnore() {
	ws := s.defs.MustNewWorksheet("DefaultMappingsTest")
	ws.MustSet("Name", NewText("julia"))
	ws.MustSet("Age", NewNumberFromInt(31))

	var data struct {
		Name  string
		Age   int    `ws:"-"`
		Name2 string `ws:"Name"`
	}

	err := ws.StructScan(&data)
	s.Require().NoError(err)
	s.Equal("julia", data.Name)
	s.Equal(0, data.Age) // zero-value, ws field value was ignored
	s.Equal("julia", data.Name2)
}

func (s *Zuite) TestStructScan_notOptionalWithValue() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("hello, world!"))

	var data struct {
		Text string `ws:"text"`
	}
	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "hello, world!", data.Text)
}

func (s *Zuite) TestStructScan_notOptionalYetUndefined() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Text string `ws:"text"`
	}
	err := ws.StructScan(&data)
	require.EqualError(s.T(), err, "field text to struct field Text: cannot convert undefined to string, dest must be a *string")
}

func (s *Zuite) TestStructScan_notOptionalYetUndefinedYetRelaxed() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Text string `ws:"text"`
	}
	ss := NewStructScanner()
	ss.AllowUndefinedToNonPointer = true
	err := ss.StructScan(ws, &data)
	s.Require().NoError(err)
	s.Require().Zero(data.Text)
}

func (s *Zuite) TestStructScan_optionalWithUndefined() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Text *string `ws:"text"`
	}
	previous := "must overwrite me"
	data.Text = &previous

	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.Nil(s.T(), data.Text)
}

func (s *Zuite) TestStructScan_optionalWithValue() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("hello, world!"))

	var data struct {
		Text *string `ws:"text"`
	}
	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), data.Text)
	require.Equal(s.T(), "hello, world!", *data.Text)
}

func (s *Zuite) TestStructScan_skipUndefinedFieldsWithNoTag() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("hello, world!"))

	var data struct {
		Text   string `ws:"text"`
		Ignore string
	}
	data.Ignore = "ignore me"
	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "hello, world!", data.Text)
	require.Equal(s.T(), "ignore me", data.Ignore)
}

func (s *Zuite) TestStructScan_slices() {
	type allSortsOfSlices struct {
		Texts  []string    `ws:"slice_t"`
		Bools  []*bool     `ws:"slice_b"`
		Ints   *[]int      `ws:"slice_n0"`
		Floats *[]*float64 `ws:"slice_n2"`
	}
	var (
		t  = true
		f  = false
		n1 = float64(1)
		n2 = 2.12
	)
	testCases := []struct {
		field          string
		elems          []Value
		expectedStruct allSortsOfSlices
	}{
		{
			field:          "slice_t",
			elems:          []Value{NewText("a"), NewText("b"), NewText("c"), NewText("")},
			expectedStruct: allSortsOfSlices{Texts: []string{"a", "b", "c", ""}},
		},
		{
			field:          "slice_b",
			elems:          []Value{NewBool(true), vUndefined, NewBool(false)},
			expectedStruct: allSortsOfSlices{Bools: []*bool{&t, nil, &f}},
		},
		{
			field:          "slice_n0",
			elems:          []Value{NewNumberFromInt(1), NewNumberFromInt(2), NewNumberFromInt(3)},
			expectedStruct: allSortsOfSlices{Ints: &[]int{1, 2, 3}},
		},
		{
			field:          "slice_n2",
			elems:          []Value{NewNumberFromInt(1), NewNumberFromFloat64(2.12), vUndefined},
			expectedStruct: allSortsOfSlices{Floats: &[]*float64{&n1, &n2, nil}},
		},
	}

	for _, tc := range testCases {
		ws := s.defs.MustNewWorksheet("all_types")
		for _, e := range tc.elems {
			ws.MustAppend(tc.field, e)
		}
		var data allSortsOfSlices
		err := ws.StructScan(&data)
		s.NoError(err)
		s.Equal(tc.expectedStruct, data)
	}
}

func (s *Zuite) TestStructScan_slicesEmpty() {
	var data struct {
		Texts  []string    `ws:"slice_t"`
		Bools  []*bool     `ws:"slice_b"`
		Ints   *[]int      `ws:"slice_n0"`
		Floats *[]*float64 `ws:"slice_n2"`
	}

	ws := s.defs.MustNewWorksheet("all_types")

	err := ws.StructScan(&data)
	s.Require().NoError(err)
	s.Nil(data.Texts)
	s.Nil(data.Bools)
	s.Nil(data.Ints)
	s.Nil(data.Floats)
}

func (s *Zuite) TestStructScan_refsPtr() {
	type allTypesStruct struct {
		Ws   *allTypesStruct `ws:"ws"`
		Num0 int             `ws:"num_0"`
	}

	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("num_0", NewNumberFromInt(123))

	wsChild := s.defs.MustNewWorksheet("all_types")
	wsChild.MustSet("num_0", NewNumberFromInt(456))

	ws.MustSet("ws", wsChild)

	var parent allTypesStruct
	err := ws.StructScan(&parent)
	s.Require().NoError(err)

	s.Equal(123, parent.Num0)

	child := parent.Ws
	s.Equal(456, child.Num0)
	s.Nil(child.Ws)
}

func (s *Zuite) TestStructScan_refsNoPtr() {
	type allTypesOtherStruct struct {
		Num0 int `ws:"num_0"`
	}
	type allTypesStruct struct {
		Ws   allTypesOtherStruct `ws:"ws"`
		Num0 int                 `ws:"num_0"`
	}

	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("num_0", NewNumberFromInt(123))

	wsChild := s.defs.MustNewWorksheet("all_types")
	wsChild.MustSet("num_0", NewNumberFromInt(456))

	ws.MustSet("ws", wsChild)

	var parent allTypesStruct
	err := ws.StructScan(&parent)
	s.Require().NoError(err)

	s.Equal(123, parent.Num0)

	child := parent.Ws
	s.Equal(456, child.Num0)
}

type ping struct {
	Pong *pong `ws:"point_to_pong"`
}

type pong struct {
	Ping *ping `ws:"point_to_Ping"`
}

func (s *Zuite) TestStructScan_refsCircularIndirect() {
	pingWs := s.defs.MustNewWorksheet("Ping")
	pongWs := s.defs.MustNewWorksheet("pong")

	pongWs.MustSet("point_to_Ping", pingWs)
	pingWs.MustSet("point_to_pong", pongWs)

	var ping ping
	err := pingWs.StructScan(&ping)
	s.Require().NoError(err)

	s.Equal(ping, *ping.Pong.Ping)
	s.Equal(ping.Pong, ping.Pong.Ping.Pong)
}

func (s *Zuite) TestStructScan_refsCircularDirect() {
	type meAndMyFriends struct {
		Me      *meAndMyFriends  `ws:"point_to_me"`
		MyPeeps []meAndMyFriends `ws:"point_to_my_friends"`
	}

	joey := s.defs.MustNewWorksheet("with_refs_and_cycles")
	phoebe := s.defs.MustNewWorksheet("with_refs_and_cycles")
	russ := s.defs.MustNewWorksheet("with_refs_and_cycles")

	// know thyself
	joey.MustSet("point_to_me", joey)
	phoebe.MustSet("point_to_me", phoebe)
	russ.MustSet("point_to_me", russ)

	// everyone is friends (except russ is too cool to reciprocate)
	phoebe.MustAppend("point_to_my_friends", joey)
	phoebe.MustAppend("point_to_my_friends", russ)
	joey.MustAppend("point_to_my_friends", phoebe)
	joey.MustAppend("point_to_my_friends", russ)

	// normally: f1 = joey, f2 = phoebe, and f3 = russ
	var f1 meAndMyFriends
	err := joey.StructScan(&f1)
	s.Require().NoError(err)

	s.Require().Equal(2, len(f1.MyPeeps))
	f2 := f1.MyPeeps[0]
	f3 := f1.MyPeeps[1]
	s.Equal(f1, *f1.Me)
	s.Equal(f1.MyPeeps[0], f2)
	s.Equal(f1.MyPeeps[1], f3)
	s.Require().Equal(2, len(f2.MyPeeps))
	s.Equal(f2.MyPeeps[0], f1)
	s.Equal(f2.MyPeeps[1], f3)
	s.Zero(len(f3.MyPeeps))
}

func (s *Zuite) TestStructScan_refsRepeat() {
	type thing struct {
		Name string `ws:"name"`
	}

	type pedantic struct {
		ThatThing      thing   `ws:"point_to_something"`
		JustMakingSure *thing  `ws:"point_to_the_same_thing"`
		KeepItSafe     []thing `ws:"and_again"`
	}

	t := s.defs.MustNewWorksheet("simple")
	t.MustSet("name", NewText("hi look at me"))

	ws := s.defs.MustNewWorksheet("with_repeat_refs")
	ws.MustSet("point_to_something", t)
	ws.MustSet("point_to_the_same_thing", t)
	ws.MustAppend("and_again", t)

	var p pedantic
	err := ws.StructScan(&p)
	s.Require().NoError(err)

	s.Require().Equal(1, len(p.KeepItSafe))
	s.Equal(p.ThatThing, *p.JustMakingSure)
	s.Equal(p.ThatThing, p.KeepItSafe[0])
}

func (s *Zuite) TestStructScan_refsNestedMoreThanOnce() {
	type level3 struct {
		S  string `ws:"text"`
		IP *int   `ws:"num_0"`
	}

	type level2 struct {
		S     string `ws:"text"`
		L3    level3 `ws:"ws"`
		L3Ptr level3 `ws:"ws"` // relies on multiple struct fields mapping to a single ws field
	}

	type level1 struct {
		L2    level2  `ws:"ws"`
		L2Ptr *level2 `ws:"ws"` // relies on multiple struct fields mapping to a single ws field
	}

	l3 := s.defs.MustNewWorksheet("all_types")
	l3.MustSet("text", NewText("we all fall down"))
	l3.MustSet("num_0", NewNumberFromInt(123))

	l2 := s.defs.MustNewWorksheet("all_types")
	l2.MustSet("text", NewText("ring around the rosie"))
	l2.MustSet("ws", l3)

	l1 := s.defs.MustNewWorksheet("all_types")
	l1.MustSet("ws", l2)

	var t level1
	err := l1.StructScan(&t)
	s.Require().NoError(err)

	// test non-pointer paths for values
	s.Equal("ring around the rosie", t.L2.S)
	s.Equal("we all fall down", t.L2.L3.S)
	s.Require().NotNil(t.L2.L3.IP)
	s.Equal(123, *(t.L2.L3.IP))

	// test pointer paths for values
	s.Equal("ring around the rosie", t.L2Ptr.S)
	s.Equal("we all fall down", t.L2Ptr.L3Ptr.S)
	s.Require().NotNil(t.L2Ptr.L3Ptr.IP)
	s.Equal(123, *(t.L2Ptr.L3Ptr.IP))
}

type special struct {
	HelloText string `ws:"xyz"` // this mapping should be totally ignored because converters kick in
}

var _ WorksheetConverter = &special{}

func (sp *special) WorksheetConvert(value Value) error {
	if text, ok := value.(*Text); ok {
		sp.HelloText = "hello, " + text.value
		return nil
	} else if ws, ok := value.(*Worksheet); ok {
		ht := ws.MustGet("text").(*Text)
		sp.HelloText = "hello! " + ht.value
		return nil
	}

	return fmt.Errorf("can only convert text and worksheet, was %s", value)
}

type specialSlice []special

var _ WorksheetConverter = &specialSlice{}

func (sps *specialSlice) WorksheetConvert(value Value) error {
	if wsSlice, ok := value.(*Slice); ok {
		for _, e := range wsSlice.Elements() {
			newSp := &special{}
			newSp.WorksheetConvert(e)
			*sps = append(*sps, *newSp)
		}
		return nil
	}

	return fmt.Errorf("can only convert slice, was %s", value)
}

func (s *Zuite) TestStructScan_worksheetConverter() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("world!"))
	ws.MustAppend("slice_t", NewText("friend"))
	ws.MustAppend("slice_t", NewText("other friend"))
	childWs := s.defs.MustNewWorksheet("all_types")
	childWs.MustSet("text", NewText("parent"))
	ws.MustSet("ws", childWs)

	var data struct {
		Special         special       `ws:"text"`
		SpecialPtr      *special      `ws:"text"`
		SpecialSlice    specialSlice  `ws:"slice_t"`
		SpecialSlicePtr *specialSlice `ws:"slice_t"`
		SpecialWs       special       `ws:"ws"`
		SpecialWsPtr    *special      `ws:"ws"`
	}
	err := ws.StructScan(&data)
	s.Require().NoError(err)

	s.Equal("hello, world!", data.Special.HelloText)

	s.Require().NotNil(data.SpecialPtr)
	s.Equal("hello, world!", data.SpecialPtr.HelloText)

	s.Require().Len(data.SpecialSlice, 2)
	s.Equal("hello, friend", data.SpecialSlice[0].HelloText)
	s.Equal("hello, other friend", data.SpecialSlice[1].HelloText)

	s.Require().NotNil(data.SpecialSlicePtr)
	spSlice := *data.SpecialSlicePtr
	s.Require().Len(spSlice, 2)
	s.Equal("hello, friend", spSlice[0].HelloText)
	s.Equal("hello, other friend", spSlice[1].HelloText)

	s.Equal("hello! parent", data.SpecialWs.HelloText)

	s.Require().NotNil(data.SpecialWsPtr)
	s.Equal("hello! parent", data.SpecialWsPtr.HelloText)
}

func (s *Zuite) TestStructScan_worksheetConverterWithUndefined() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		SpecialPtr *special `ws:"text"`
	}
	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.Nil(s.T(), data.SpecialPtr)
}

func (s *Zuite) TestStructScanner_overrideTypes() {
	type simple struct {
		Name              string    `ws:"name"`
		AgeButReallyATime time.Time `ws:"age"`
	}

	ws := s.defs.MustNewWorksheet("simple")
	ws.MustSet("name", NewText("json"))
	u := 1526671232
	ws.MustSet("age", NewNumberFromInt(u))

	var (
		timeConverter = func(v Value) (interface{}, error) {
			return time.Unix(v.(*Number).value, 0), nil
		}
		stringConverter = func(v Value) (interface{}, error) {
			return "yaml", nil
		}
	)

	testCases := []struct {
		converters     map[reflect.Type]func(Value) (interface{}, error)
		expectedError  string
		expectedStruct simple
	}{
		{
			converters:     nil,
			expectedError:  "field age to struct field AgeButReallyATime: cannot convert number[0] to time.Time",
			expectedStruct: simple{},
		},
		{
			converters: map[reflect.Type]func(Value) (interface{}, error){
				reflect.TypeOf(time.Time{}): timeConverter,
			},
			expectedError:  "",
			expectedStruct: simple{Name: "json", AgeButReallyATime: time.Unix(int64(u), 0)},
		},
		{
			converters: map[reflect.Type]func(Value) (interface{}, error){
				reflect.TypeOf(time.Time{}): timeConverter,
				reflect.TypeOf(""):          stringConverter,
			},
			expectedError:  "",
			expectedStruct: simple{Name: "yaml", AgeButReallyATime: time.Unix(int64(u), 0)},
		},
	}

	for _, tc := range testCases {
		var data simple
		ss := NewStructScanner()
		for t, f := range tc.converters {
			ss.RegisterConverter(t, f)
		}
		err := ss.StructScan(ws, &data)
		if tc.expectedError == "" {
			s.NoError(err)
			s.Equal(tc.expectedStruct, data)
			// while we are here, test that a scanner can be used multiple times on the same ws without issues
			err = ss.StructScan(ws, &data)
			s.NoError(err)
			s.Equal(tc.expectedStruct, data)
		} else {
			s.EqualError(err, tc.expectedError)
		}
	}
}

func (s *Zuite) TestStructScanner_precedenceOverWorksheetConverter() {
	ws := s.defs.MustNewWorksheet("simple")
	ws.MustSet("name", NewText("wework"))

	specialConverter := func(v Value) (interface{}, error) {
		return special{HelloText: "hello, " + strings.ToUpper(v.(*Text).value)}, nil
	}

	var data struct {
		Special special `ws:"name"`
	}
	ss := NewStructScanner()
	ss.RegisterConverter(reflect.TypeOf(data.Special), specialConverter)
	err := ss.StructScan(ws, &data)
	s.Require().NoError(err)

	s.Equal("hello, WEWORK", data.Special.HelloText)
}

type myString string
type myInt64 int64
type myBool bool

var (
	stringTyp   = reflect.TypeOf(string(""))
	intTyp      = reflect.TypeOf(int(0))
	int8Typ     = reflect.TypeOf(int8(0))
	int16Typ    = reflect.TypeOf(int16(0))
	int32Typ    = reflect.TypeOf(int32(0))
	int64Typ    = reflect.TypeOf(int64(0))
	uintTyp     = reflect.TypeOf(uint(0))
	uint8Typ    = reflect.TypeOf(uint8(0))
	uint16Typ   = reflect.TypeOf(uint16(0))
	uint32Typ   = reflect.TypeOf(uint32(0))
	uint64Typ   = reflect.TypeOf(uint64(0))
	float32Typ  = reflect.TypeOf(float32(0))
	float64Typ  = reflect.TypeOf(float64(0))
	boolTyp     = reflect.TypeOf(bool(true))
	myStringTyp = reflect.TypeOf(myString(""))
	myInt64Typ  = reflect.TypeOf(myInt64(0))
	myBoolTyp   = reflect.TypeOf(myBool(true))
)

func (s *Zuite) TestStructScan_convert() {
	cases := []struct {
		source   Value
		dest     reflect.Type
		expected interface{}
	}{
		{NewText("hello"), stringTyp, "hello"},
		{NewBool(true), stringTyp, "true"},
		{NewNumberFromFloat64(123.45), stringTyp, "123.45"},

		{NewBool(true), boolTyp, true},

		{NewNumberFromInt(123), intTyp, int(123)},
		{NewNumberFromInt64(123), int64Typ, int64(123)},

		{NewNumberFromFloat32(123.45), float32Typ, float32(123.45)},
		{NewNumberFromFloat64(123.45), float64Typ, float64(123.45)},

		{NewNumberFromInt8(127), int8Typ, int8(127)},
		{NewNumberFromInt8(-128), int8Typ, int8(-128)},
		{NewNumberFromInt16(32767), int16Typ, int16(32767)},
		{NewNumberFromInt16(-32768), int16Typ, int16(-32768)},
		{NewNumberFromInt32(2147483647), int32Typ, int32(2147483647)},
		{NewNumberFromInt32(-2147483648), int32Typ, int32(-2147483648)},
		{NewNumberFromInt64(9223372036854775807), int64Typ, int64(9223372036854775807)},
		{NewNumberFromInt64(-9223372036854775808), int64Typ, int64(-9223372036854775808)},

		{NewNumberFromUint(255), uintTyp, uint(255)},
		{NewNumberFromUint8(255), uint8Typ, uint8(255)},
		{NewNumberFromUint16(65535), uint16Typ, uint16(65535)},
		{NewNumberFromUint32(4294967295), uint32Typ, uint32(4294967295)},
		// TODO: See issue #29: support for arbitrary precision numbers
		// {NewNumberFromUint64(18446744073709551615), uint64Typ, uint64(18446744073709551615)},

		{NewText("hello"), myStringTyp, "hello"},
		{NewBool(true), myBoolTyp, true},
		{NewNumberFromInt64(33), myInt64Typ, int64(33)},
	}
	for _, ex := range cases {
		ctx := &structScanCtx{}
		fieldCtx := structScanFieldCtx{
			sourceFieldName: "source",
			sourceType:      ex.source.Type(),
			destFieldName:   "Dest",
			destType:        ex.dest,
		}
		actual, err := ctx.convert(fieldCtx, ex.source)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), ex.expected, actual.Interface())
	}
}

func (s *Zuite) TestStructScan_convertErrors() {
	cases := []struct {
		source   Value
		dest     reflect.Type
		expected string
	}{
		{NewText("hello"), intTyp, "text to int"},
		{NewBool(true), intTyp, "bool to int"},

		{NewText("hello"), boolTyp, "text to bool"},
		{NewNumberFromFloat64(123.45), boolTyp, "number[2] to bool"},

		{NewText("hello"), intTyp, "text to int"},
		{NewBool(true), intTyp, "bool to int"},
		{NewNumberFromFloat64(123.45), intTyp, "number[2] to int"},

		{NewText("hello"), int64Typ, "text to int64"},
		{NewBool(true), int64Typ, "bool to int64"},
		{NewNumberFromFloat64(123.45), int64Typ, "number[2] to int64"},

		{NewText("hello"), float32Typ, "text to float32"},
		{NewBool(true), float32Typ, "bool to float32"},

		{NewText("hello"), float64Typ, "text to float64"},
		{NewBool(true), float64Typ, "bool to float64"},

		{NewNumberFromInt(128), int8Typ, "number[0] to int8, value out of range"},
		{NewNumberFromInt(-129), int8Typ, "number[0] to int8, value out of range"},
		{NewNumberFromInt(32768), int16Typ, "number[0] to int16, value out of range"},
		{NewNumberFromInt(-32769), int16Typ, "number[0] to int16, value out of range"},
		{NewNumberFromInt(2147483648), int32Typ, "number[0] to int32, value out of range"},
		{NewNumberFromInt(-2147483649), int32Typ, "number[0] to int32, value out of range"},
		// TODO: See issue #29: support for arbitrary precision numbers
		// {MustParseLiteral("9_223_372_036_854_775_808"), int64Typ, "number[0] to int64, value out of range"},
		// {MustParseLiteral("-9_223_372_036_854_775_809"), int64Typ, "number[0] to int64, value out of range"},

		{NewNumberFromInt(256), uint8Typ, "number[0] to uint8, value out of range"},
		{NewNumberFromInt(65536), uint16Typ, "number[0] to uint16, value out of range"},
		{NewNumberFromInt(4294967296), uint32Typ, "number[0] to uint32, value out of range"},
		// TODO: See issue #29: support for arbitrary precision numbers
		// {MustParseLiteral("18_446_744_073_709_551_616"), uint32Typ, "number[0] to int64, value out of range"},
	}
	for _, ex := range cases {
		ctx := &structScanCtx{}
		fieldCtx := structScanFieldCtx{
			sourceFieldName: "source",
			sourceType:      ex.source.Type(),
			destFieldName:   "Dest",
			destType:        ex.dest,
		}
		_, err := ctx.convert(fieldCtx, ex.source)
		assert.EqualErrorf(s.T(), err, "field source to struct field Dest: cannot convert "+ex.expected,
			"converting %s to %s", ex.source, ex.dest)
	}
}

func (s *Zuite) TestStructScan_convertibleTypes() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("abc"))
	ws.MustSet("bool", NewBool(true))
	ws.MustSet("num_0", NewNumberFromInt64(8765))

	type altTypes struct {
		Text    myString  `ws:"text"`
		Bool    myBool    `ws:"bool"`
		Num     myInt64   `ws:"num_0"`
		TextPtr *myString `ws:"text"`
		BoolPtr *myBool   `ws:"bool"`
		NumPtr  *myInt64  `ws:"num_0"`
	}
	var data altTypes
	err := ws.StructScan(&data)
	s.Require().NoError(err)
	var (
		textResult = myString("abc")
		boolResult = myBool(true)
		numResult  = myInt64(8765)
	)
	s.Equal(altTypes{
		Text:    "abc",
		Bool:    true,
		Num:     8765,
		TextPtr: &textResult,
		BoolPtr: &boolResult,
		NumPtr:  &numResult,
	}, data)
}
