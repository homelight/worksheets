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
	"io"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
)

// Definitions encapsulate one or many worksheet definitions, and is the
// overall entry point into the worksheet framework.
type Definitions struct {
	// wss holds all worksheet definitions
	wss map[string]*tWorksheet
}

// Worksheet is an instance of a worksheet, which can be manipulated, as well
// as saved, and restored from a permanent storage.
type Worksheet struct {
	// dfn holds the definition of this worksheet
	tws *tWorksheet

	// data holds all the worksheet data
	data map[int]rValue
}

// NewDefinitions parses a worksheet definition, and creates a worksheet
// model from it.
func NewDefinitions(src io.Reader) (*Definitions, error) {
	// TODO(pascal): support reading multiple worksheet definitions in one file
	p := newParser(src)
	tws, err := p.parseWorksheet()
	if err != nil {
		return nil, err
	}
	return &Definitions{
		wss: map[string]*tWorksheet{
			tws.name: tws,
		},
	}, nil
}

func (d *Definitions) NewWorksheet(name string) (*Worksheet, error) {
	tws, ok := d.wss[name]
	if !ok {
		return nil, fmt.Errorf("unknown worksheet %s", name)
	}
	return &Worksheet{
		tws:  tws,
		data: make(map[int]rValue),
	}, nil
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
	field, ok := ws.tws.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// type check
	fmt.Printf("%v\n", lit)
	litType := lit.value.Type()
	if ok := litType.AssignableTo(field.typ); !ok {
		return fmt.Errorf("cannot assign %s to %s", lit.value, field.typ)
	}

	// store
	ws.data[index] = lit.value

	return nil
}

func (ws *Worksheet) Unset(name string) error {
	// lookup field by name
	field, ok := ws.tws.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// store
	delete(ws.data, index)

	return nil
}

func (ws *Worksheet) IsSet(name string) (bool, error) {
	// lookup field by name
	field, ok := ws.tws.fieldsByName[name]
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
	field, ok := ws.tws.fieldsByName[name]
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

type tField struct {
	index int
	name  string
	typ   rType
	// also need constrainedBy *tExpression
	// also need computedBy    *tExpression
}

// TODO(pascal): do we need a *tLiteral, or can we just use an rValue in the tree?
type tLiteral struct {
	value rValue
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
	// TODO(pascal): print with proper scale
	return fmt.Sprintf("%d", value.value)
}

func (value *tText) Type() rType {
	return &tTextType{}
}

func (value *tText) String() string {
	return value.value
}

func (value *tBool) Type() rType {
	return &tBoolType{}
}

func (value *tBool) String() string {
	return fmt.Sprintf("%b", value.value)
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
	pName   = newTokenPattern("name", "[a-z]")
	pIndex  = newTokenPattern("index", "[0-9]+")
	pNumber = newTokenPattern("number", "[0-9]+(\\.[0-9]+)?")
	pString = newTokenPattern("string", "\".*\"")
)

func (p *parser) parseWorksheet() (*tWorksheet, error) {
	ws := tWorksheet{
		fieldsByName:  make(map[string]*tField),
		fieldsByIndex: make(map[int]*tField),
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
		ws.fields = append(ws.fields, field)

		if _, ok := ws.fieldsByName[field.name]; ok {
			return nil, fmt.Errorf("multiple fields with name %s", field.name)
		}
		ws.fieldsByName[field.name] = field

		if _, ok := ws.fieldsByIndex[field.index]; ok {
			return nil, fmt.Errorf("multiple fields with index %d", field.index)
		}
		ws.fieldsByIndex[field.index] = field
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

func parseLiteralFromString(input string) (*tLiteral, error) {
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

func (p *parser) parseLiteral() (*tLiteral, error) {
	var err error
	var negNumber bool
	token := p.next()
	switch token {
	case "undefined":
		return &tLiteral{&tUndefined{}}, nil
	case "true":
		return &tLiteral{&tBool{true}}, nil
	case "false":
		return &tLiteral{&tBool{false}}, nil
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
		return &tLiteral{&tNumber{value, &tNumberType{scale}}}, nil
	}
	if pString.re.MatchString(token) {
		value, err := strconv.Unquote(token)
		if err != nil {
			return nil, err
		}
		return &tLiteral{&tText{value}}, nil
	}
	return nil, fmt.Errorf("unknown literal")
}

type tokenPattern struct {
	name string
	re   *regexp.Regexp
}

func newTokenPattern(name, regex string) *tokenPattern {
	return &tokenPattern{
		name: name,
		re:   regexp.MustCompile(regex),
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
