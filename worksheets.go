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
	data map[int]interface{}
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
		data: make(map[int]interface{}),
	}, nil
}

func (ws *Worksheet) Set(name string, value interface{}) error {
	// TODO(pascal): create a 'change', and then commit that change, garantee
	// that commits are atomic, and either win or lose the race by using
	// optimistic concurrency. Change must be a a Definition level, since it
	// could span multiple worksheets at once.

	// lookup field by name
	field, ok := ws.tws.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// type check
	if err := field.typ.check(value); err != nil {
		return err
	}

	// store
	ws.data[index] = value

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

func (ws *Worksheet) Get(name string) (interface{}, error) {
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
	if err := field.typ.check(value); err != nil {
		return nil, err
	}

	return value, nil
}

func (ws *Worksheet) GetText(name string) (string, error) {
	value, err := ws.Get(name)
	if err != nil {
		return "", err
	}

	sValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("field %s cannot be converted to text (type is %T)", name, value)
	}
	return sValue, nil
}

// ------ definitions ------

type tWorksheet struct {
	name          string
	fields        []*tField
	fieldsByName  map[string]*tField
	fieldsByIndex map[int]*tField
}

type tField struct {
	index int
	name  string
	typ   *tType
	// also need constrainedBy *tExpression
	// also need computedBy    *tExpression
}

type tLiteral struct {
	value vValue
}

type tType struct {
	name  string
	check func(interface{}) error
}

type vValue interface {
}

type vUndefined struct{}

type vNumber struct {
	value int64
	scale int
}

type vString struct {
	value string
}

type vBool struct {
	value bool
}

// Assert that all values are vValue.
var _ []vValue = []vValue{
	&vUndefined{},
	&vNumber{},
	&vString{},
	&vBool{},
}

// tTypes holds system defined types
var tTypes = []*tType{
	&tType{
		name: "text",
		check: func(value interface{}) error {
			_, ok := value.(string)
			if !ok {
				return fmt.Errorf("unable to cast %T to string", value)
			}
			return nil
		},
	},
	&tType{
		name: "bool",
		check: func(value interface{}) error {
			_, ok := value.(bool)
			if !ok {
				return fmt.Errorf("unable to cast %T to bool", value)
			}
			return nil
		},
	},
}

// tTypesByName holds system defined types by name
var tTypesByName = tTypesByNameInit()

func tTypesByNameInit() map[string]*tType {
	m := make(map[string]*tType, len(tTypes))
	for _, tType := range tTypes {
		m[tType.name] = tType
	}
	return m
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

	typ, err := p.nextAndCheck(pName)
	if err != nil {
		return nil, err
	}

	// TODO(pascal): simplistic, we'd need to do type resolution on a second
	// pass, when we have proper scoping.
	tType, ok := tTypesByName[typ]
	if !ok {
		return nil, fmt.Errorf("unknown type %s", typ)
	}

	f := &tField{
		index: index,
		name:  name,
		typ:   tType,
	}

	return f, nil
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
		return &tLiteral{&vUndefined{}}, nil
	case "true":
		return &tLiteral{&vBool{true}}, nil
	case "false":
		return &tLiteral{&vBool{false}}, nil
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
		return &tLiteral{&vNumber{value, scale}}, nil
	}
	if pString.re.MatchString(token) {
		value, err := strconv.Unquote(token)
		if err != nil {
			return nil, err
		}
		return &tLiteral{&vString{value}}, nil
	}
	return nil, nil
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
