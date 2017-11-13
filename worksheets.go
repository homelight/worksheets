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

func (ws *Worksheet) SetText(name string, value string) error {
	field, ok := ws.tws.fieldsByName[name]
	if !ok {
		return fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	// TODO(pascal): check using field.typ
	ws.data[index] = value

	return nil
}

func (ws *Worksheet) GetText(name string) (string, error) {
	field, ok := ws.tws.fieldsByName[name]
	if !ok {
		return "", fmt.Errorf("unknown field %s", name)
	}
	index := field.index

	value, ok := ws.data[index]
	if !ok {
		return "", fmt.Errorf("no value for field %s", name)
	}

	// TODO(pascal): check using field.typ
	return value.(string), nil
}

// ------ parsing ------

type tWorksheet struct {
	name          string
	fields        []*tField
	fieldsByName  map[string]*tField
	fieldsByIndex map[int]*tField
}

type tField struct {
	index int
	name  string
	typ   string // should be tType
	// also need constrainedBy *tExpression
	// also need computedBy    *tExpression
}

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

	f := &tField{
		index: index,
		name:  name,
		typ:   typ,
	}

	return f, nil
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
