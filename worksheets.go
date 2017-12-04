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
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/satori/go.uuid"
)

// Store ... TODO(pascal): write about abstraction.
type Store interface {
	// Load loads the worksheet with identifier `id` from the store.
	Load(name, id string) (*Worksheet, error)

	// Save saves the worksheet to the store.
	Save(*Worksheet) error
}

// Definitions encapsulate one or many worksheet definitions, and is the
// overall entry point into the worksheet framework.
type Definitions struct {
	// defs holds all worksheet definitions
	defs map[string]*tWorksheet
}

// Worksheet is an instance of a worksheet, which can be manipulated, as well
// as saved, and restored from a permanent storage.
type Worksheet struct {
	// dfn holds the definition of this worksheet
	def *tWorksheet

	// data holds all the worksheet data
	data map[int]rValue
}

const (
	// indexId is the reserved index to store a worksheet's identifier.
	indexId = -1

	// indexVersion is the reserved index to store a worksheet's version.
	indexVersion = -2
)

// NewDefinitions parses a worksheet definition, and creates a worksheet
// model from it.
func NewDefinitions(src io.Reader) (*Definitions, error) {
	// TODO(pascal): support reading multiple worksheet definitions in one file
	p := newParser(src)
	def, err := p.parseWorksheet()
	if err != nil {
		return nil, err
	}
	return &Definitions{
		defs: map[string]*tWorksheet{
			def.name: def,
		},
	}, nil
}

func (defs *Definitions) NewWorksheet(name string) (*Worksheet, error) {
	ws, err := defs.newUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	// uuid
	id := uuid.NewV4()
	if err := ws.Set("id", fmt.Sprintf(`"%s"`, id)); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	// version
	if err := ws.Set("version", strconv.Itoa(1)); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	// validate
	if err := ws.validate(); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}

	return ws, nil
}

func (defs *Definitions) newUninitializedWorksheet(name string) (*Worksheet, error) {
	def, ok := defs.defs[name]
	if !ok {
		return nil, fmt.Errorf("unknown worksheet %s", name)
	}

	ws := &Worksheet{
		def:  def,
		data: make(map[int]rValue),
	}

	return ws, nil
}

func (ws *Worksheet) validate() error {
	// ensure we have an id and a version
	if _, ok := ws.data[indexId]; !ok {
		return fmt.Errorf("missing id")
	}
	if _, ok := ws.data[indexVersion]; !ok {
		return fmt.Errorf("missing version")
	}

	// ensure all values are of the proper type
	for index, value := range ws.data {
		field, ok := ws.def.fieldsByIndex[index]
		if !ok {
			return fmt.Errorf("value present for unknown field index %d", index)
		}
		if ok := value.Type().AssignableTo(field.typ); !ok {
			return fmt.Errorf("value present with unassignable type for field index %d", index)
		}
	}

	return nil
}

func (ws *Worksheet) Set(name string, value string) error {
	// TODO(pascal): create a 'change', and then commit that change, garantee
	// that commits are atomic, and either win or lose the race by using
	// optimistic concurrency. Change must be a a Definition level, since it
	// could span multiple worksheets at once.

	// parse literal
	lit, err := parseLiteralFromString(value)
	if err != nil {
		return err
	}

	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// type check
	litType := lit.Type()
	if ok := litType.AssignableTo(field.typ); !ok {
		return fmt.Errorf("cannot assign %s to %s", lit, field.typ)
	}

	// store
	if lit.Type().AssignableTo(&tUndefinedType{}) {
		delete(ws.data, index)
	} else {
		ws.data[index] = lit
	}

	return nil
}

func (ws *Worksheet) Unset(name string) error {
	return ws.Set(name, "undefined")
}

func (ws *Worksheet) IsSet(name string) (bool, error) {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return false, fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// check presence of value
	_, isSet := ws.data[index]

	return isSet, nil
}

// TODO(pascal): need to think about proper return type here, should be consistent with Set
func (ws *Worksheet) Get(name string) (rValue, error) {
	// lookup field by name
	field, ok := ws.def.fieldsByName[name]
	if !ok {
		return nil, fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// is a value set for this field?
	value, ok := ws.data[index]
	if !ok {
		return nil, fmt.Errorf("no value for field %s", name)
	}

	// type check
	if ok := value.Type().AssignableTo(field.typ); !ok {
		return nil, fmt.Errorf("cannot assign %s to %s", value, field.typ)
	}

	return value, nil
}

// ------ definitions ------

// Naming conventsions:
// - t prefix (e.g. tWorksheet) is for AST related converns
// - r prefix (e.g. rString) is for runtime concerns

type tWorksheet struct {
	name          string
	fields        []*tField
	fieldsByName  map[string]*tField
	fieldsByIndex map[int]*tField
}

func (ws *tWorksheet) addField(field *tField) error {
	if _, ok := ws.fieldsByName[field.name]; ok {
		return fmt.Errorf("multiple fields with name %s", field.name)
	}

	if _, ok := ws.fieldsByIndex[field.index]; ok {
		return fmt.Errorf("multiple fields with index %d", field.index)
	}

	ws.fields = append(ws.fields, field)
	ws.fieldsByName[field.name] = field
	ws.fieldsByIndex[field.index] = field

	return nil
}

type tField struct {
	index int
	name  string
	typ   rType
	// also need constrainedBy *tExpression
	// also need computedBy    *tExpression
}

type tUndefinedType struct{}

type tTextType struct{}

type tBoolType struct{}

type tNumberType struct {
	scale int
}

// Assert that all type literals are rType.
var _ []rType = []rType{
	&tUndefinedType{},
	&tTextType{},
	&tBoolType{},
	&tNumberType{},
}

func (typ *tUndefinedType) AssignableTo(_ rType) bool {
	return true
}

func (typ *tUndefinedType) String() string {
	return "undefined"
}

func (typ *tTextType) AssignableTo(u rType) bool {
	_, ok := u.(*tTextType)
	return ok
}

func (typ *tTextType) String() string {
	return "text"
}

func (typ *tBoolType) AssignableTo(u rType) bool {
	_, ok := u.(*tBoolType)
	return ok
}

func (typ *tBoolType) String() string {
	return "bool"
}

func (typ *tNumberType) AssignableTo(u rType) bool {
	uNum, ok := u.(*tNumberType)
	return ok && typ.scale <= uNum.scale
}

func (typ *tNumberType) String() string {
	return fmt.Sprintf("number(%d)", typ.scale)
}

type tUndefined struct{}

type tNumber struct {
	value int64
	typ   *tNumberType
}

type tText struct {
	value string
}

type tBool struct {
	value bool
}

// Assert that all value literals are rValue.
var _ []rValue = []rValue{
	&tUndefined{},
	&tNumber{},
	&tText{},
	&tBool{},
}

func (value *tUndefined) Type() rType {
	return &tUndefinedType{}
}

func (value *tUndefined) String() string {
	return "undefined"
}

func (value *tNumber) Type() rType {
	return value.typ
}

func (value *tNumber) String() string {
	s := strconv.FormatInt(value.value, 10)
	scale := value.typ.scale
	if scale == 0 {
		return s
	}

	// We count down from most significant digit in the number we are generating.
	// For instance 123 with scale 3 means 0.123 so the most significant digit
	// is 0 (at index 4), then 1 (at index 3), and so on. While counting down,
	// we generate the correct representation, by using the digits of the value
	// or introducing 0s as necessery. We also add the period at the appropriate
	// place while iterating.
	var (
		i      = scale + 1
		l      = len(s)
		buffer bytes.Buffer
	)
	if l > i {
		i = l
	}
	for i > 0 {
		if i == scale {
			buffer.WriteRune('.')
		}
		if i > l {
			buffer.WriteRune('0')
		} else {
			buffer.WriteByte(s[l-i])
		}
		i--
	}
	return buffer.String()
}

func (value *tText) Type() rType {
	return &tTextType{}
}

func (value *tText) String() string {
	return strconv.Quote(value.value)
}

func (value *tBool) Type() rType {
	return &tBoolType{}
}

func (value *tBool) String() string {
	return strconv.FormatBool(value.value)
}

// ------ parsing ------

type parser struct {
	// parser
	// ....

	// tokenizer
	s    *scanner.Scanner
	toks []string
}

func newParser(src io.Reader) *parser {
	s := &scanner.Scanner{
		Mode: scanner.GoTokens,
	}
	s.Init(src)
	return &parser{
		s: s,
	}
}

var (
	// tokens
	pWorksheet = newTokenPattern("worksheet", "worksheet")
	pLacco     = newTokenPattern("{", "\\{")
	pRacco     = newTokenPattern("}", "\\}")
	pLparen    = newTokenPattern("(", "\\(")
	pRparen    = newTokenPattern(")", "\\)")
	pColon     = newTokenPattern(":", ":")

	// token patterns
	pName   = newTokenPattern("name", "[a-z]+([a-z_]*[a-z])?")
	pIndex  = newTokenPattern("index", "[0-9]+")
	pNumber = newTokenPattern("number", "[0-9]+(\\.[0-9]+)?")
	pString = newTokenPattern("string", "\".*\"")
)

func (p *parser) parseWorksheet() (*tWorksheet, error) {
	// initialize tWorksheet
	ws := tWorksheet{
		fieldsByName:  make(map[string]*tField),
		fieldsByIndex: make(map[int]*tField),
	}
	if err := ws.addField(&tField{
		index: indexId,
		name:  "id",
		typ:   &tTextType{},
	}); err != nil {
		panic("unexpected")
	}
	if err := ws.addField(&tField{
		index: indexVersion,
		name:  "version",
		typ:   &tNumberType{},
	}); err != nil {
		panic("unexpected")
	}

	_, err := p.nextAndCheck(pWorksheet)
	if err != nil {
		return nil, err
	}

	name, err := p.nextAndCheck(pName)
	if err != nil {
		return nil, err
	}
	ws.name = name

	_, err = p.nextAndCheck(pLacco)
	if err != nil {
		return nil, err
	}

	for token := p.peek(); token != "}"; token = p.peek() {
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		if err := ws.addField(field); err != nil {
			return nil, err
		}
	}

	_, err = p.nextAndCheck(pRacco)
	if err != nil {
		return nil, err
	}

	return &ws, nil
}

func (p *parser) parseField() (*tField, error) {
	sIndex, err := p.nextAndCheck(pIndex)
	if err != nil {
		return nil, err
	}
	index, err := strconv.Atoi(sIndex)
	if err != nil {
		// unexpected since sIndex should conform to pIndex
		panic(err)
	}

	_, err = p.nextAndCheck(pColon)
	if err != nil {
		return nil, err
	}

	name, err := p.nextAndCheck(pName)
	if err != nil {
		return nil, err
	}

	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}

	f := &tField{
		index: index,
		name:  name,
		typ:   typ,
	}

	return f, nil
}

func (p *parser) parseType() (rType, error) {
	name, err := p.nextAndCheck(pName)
	if err != nil {
		return nil, err
	}

	switch name {
	case "text":
		return &tTextType{}, nil
	case "bool":
		return &tBoolType{}, nil
	case "undefined":
		return &tUndefinedType{}, nil
	case "number":
		_, err := p.nextAndCheck(pLparen)
		if err != nil {
			return nil, err
		}
		sScale, err := p.nextAndCheck(pIndex)
		if err != nil {
			return nil, err
		}
		scale, err := strconv.Atoi(sScale)
		if err != nil {
			// unexpected since sIndex should conform to pIndex
			panic(err)
		}
		_, err = p.nextAndCheck(pRparen)
		if err != nil {
			return nil, err
		}
		return &tNumberType{scale}, nil
	}

	return nil, fmt.Errorf("unknown type %s", name)
}

func parseLiteralFromString(input string) (rValue, error) {
	reader := strings.NewReader(input)
	p := newParser(reader)
	lit, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	if reader.Len() != 0 {
		return nil, fmt.Errorf("expecting eof")
	}
	return lit, nil
}

func (p *parser) parseLiteral() (rValue, error) {
	var err error
	var negNumber bool
	token := p.next()
	switch token {
	case "undefined":
		return &tUndefined{}, nil
	case "true":
		return &tBool{true}, nil
	case "false":
		return &tBool{false}, nil
	case "-":
		negNumber = true
		token, err = p.nextAndCheck(pNumber)
		if err != nil {
			return nil, err
		}
	}
	if pNumber.re.MatchString(token) {
		dot := strings.Index(token, ".")
		value, err := strconv.ParseInt(strings.Replace(token, ".", "", 1), 10, 64)
		if err != nil {
			return nil, err
		}
		var scale int
		if dot < 0 {
			scale = 0
		} else {
			scale = len(token) - dot - 1
		}
		if negNumber {
			value = -value
		}
		return &tNumber{value, &tNumberType{scale}}, nil
	}
	if pString.re.MatchString(token) {
		value, err := strconv.Unquote(token)
		if err != nil {
			return nil, err
		}
		return &tText{value}, nil
	}
	return nil, fmt.Errorf("unknown literal, found %s", token)
}

type tokenPattern struct {
	name string
	re   *regexp.Regexp
}

func newTokenPattern(name, regex string) *tokenPattern {
	return &tokenPattern{
		name: name,
		re:   regexp.MustCompile("^" + regex + "$"),
	}
}

func (p *parser) nextAndCheck(expected *tokenPattern) (string, error) {
	token := p.next()

	var err error
	if !expected.re.MatchString(token) {
		err = fmt.Errorf("expected %s, found %s", expected.name, token)
	}

	return token, err
}

func (p *parser) next() string {
	if len(p.toks) == 0 {
		p.s.Scan()
		token := p.s.TokenText()
		return token
	} else {
		token := p.toks[0]
		p.toks = p.toks[1:]
		return token
	}
}

func (p *parser) peek() string {
	token := p.next()
	p.toks = append(p.toks, token)
	return token
}
