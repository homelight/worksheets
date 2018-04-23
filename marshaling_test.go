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
	require.EqualError(s.T(), err, "field text to struct field Text: undefined into not nullable")
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

func (s *Zuite) TestStructScan_skipFieldsWithNoTag() {
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

func (s *Zuite) TestStructScan_slicesNotSupported() {
	ws := s.defs.MustNewWorksheet("all_types")

	var data struct {
		Texts []string `ws:"slice_t"`
	}
	err := ws.StructScan(&data)
	require.EqualError(s.T(), err, "struct field Texts: cannot StructScan slices (yet)")
}

type allTypesStruct struct {
	Ws   *allTypesStruct `ws:"ws"`
	Num0 int             `ws:"num_0"`
}

func (s *Zuite) TestStructScan_refsNotSupported() {
	ws := s.defs.MustNewWorksheet("all_types")

	var parent allTypesStruct
	err := ws.StructScan(&parent)
	require.EqualError(s.T(), err, "struct field Ws: cannot StructScan worksheets (yet)")
}

type special struct {
	helloText string
}

var _ WorksheetConverter = &special{}

func (sp *special) WorksheetConvert(value Value) error {
	if text, ok := value.(*Text); ok {
		sp.helloText = "hello, " + text.value
		return nil
	}

	return fmt.Errorf("can only convert text, was %s", value)
}

func (s *Zuite) TestStructScan_worksheetConverter() {
	ws := s.defs.MustNewWorksheet("all_types")
	ws.MustSet("text", NewText("world!"))

	var data struct {
		Special    special  `ws:"text"`
		SpecialPtr *special `ws:"text"`
	}
	err := ws.StructScan(&data)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "hello, world!", data.Special.helloText)
	require.NotNil(s.T(), data.SpecialPtr)
	require.Equal(s.T(), "hello, world!", data.SpecialPtr.helloText)
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

var (
	stringTyp  = reflect.TypeOf(string(""))
	intTyp     = reflect.TypeOf(int(0))
	int8Typ    = reflect.TypeOf(int8(0))
	int16Typ   = reflect.TypeOf(int16(0))
	int32Typ   = reflect.TypeOf(int32(0))
	int64Typ   = reflect.TypeOf(int64(0))
	uintTyp    = reflect.TypeOf(uint(0))
	uint8Typ   = reflect.TypeOf(uint8(0))
	uint16Typ  = reflect.TypeOf(uint16(0))
	uint32Typ  = reflect.TypeOf(uint32(0))
	uint64Typ  = reflect.TypeOf(uint64(0))
	float32Typ = reflect.TypeOf(float32(0))
	float64Typ = reflect.TypeOf(float64(0))
	boolTyp    = reflect.TypeOf(bool(true))
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
	}
	for _, ex := range cases {
		ctx := convertCtx{
			sourceFieldName: "source",
			sourceType:      ex.source.Type(),
			destFieldName:   "Dest",
			destType:        ex.dest,
		}
		actual, err := convert(ctx, ex.source)
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
		ctx := convertCtx{
			sourceFieldName: "source",
			sourceType:      ex.source.Type(),
			destFieldName:   "Dest",
			destType:        ex.dest,
		}
		_, err := convert(ctx, ex.source)
		assert.EqualErrorf(s.T(), err, "field source to struct field Dest: cannot convert "+ex.expected,
			"converting %s to %s", ex.source, ex.dest)
	}
}
