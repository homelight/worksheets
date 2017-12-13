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

type parser struct {
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

type tWorksheet struct {
	name          string
	fields        []*tField
	fieldsByName  map[string]*tField
	fieldsByIndex map[int]*tField

	// derived values handling
	externals  map[int]ComputedBy
	dependants map[int][]int
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
	index      int
	name       string
	typ        Type
	computedBy expression
	// also need constrainedBy *tExpression
}

type tUndefinedType struct{}

type tTextType struct{}

type tBoolType struct{}

type tNumberType struct {
	scale int
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

var (
	// tokens
	pLacco      = newTokenPattern("{", "\\{")
	pRacco      = newTokenPattern("}", "\\}")
	pLparen     = newTokenPattern("(", "\\(")
	pRparen     = newTokenPattern(")", "\\)")
	pLbracket   = newTokenPattern("[", "\\[")
	pRbracket   = newTokenPattern("]", "\\]")
	pColon      = newTokenPattern(":", ":")
	pWorksheet  = newTokenPattern("worksheet", "worksheet")
	pComputedBy = newTokenPattern("computed_by", "computed_by")
	pExternal   = newTokenPattern("external", "external")

	// token patterns
	pName   = newTokenPattern("name", "[a-z]+([a-z_]*[a-z])?")
	pIndex  = newTokenPattern("index", "[0-9]+")
	pNumber = newTokenPattern("number", "[0-9]+(\\.[0-9]+)?")
	pString = newTokenPattern("string", "\".*\"")
)

func (p *parser) parseWorksheets() (map[string]*tWorksheet, error) {
	wsDefs := make(map[string]*tWorksheet)

	for pWorksheet.re.MatchString(p.peek()) {
		def, err := p.parseWorksheet()
		if err != nil {
			return nil, err
		}
		if _, exists := wsDefs[def.name]; exists {
			return nil, fmt.Errorf("multiple worksheets with name %s", def.name)
		}
		wsDefs[def.name] = def
	}
	if len(wsDefs) == 0 {
		return nil, fmt.Errorf("no worksheets defined")
	}

	return wsDefs, nil
}

func (p *parser) parseWorksheet() (*tWorksheet, error) {
	// initialize tWorksheet
	ws := tWorksheet{
		fieldsByName:  make(map[string]*tField),
		fieldsByIndex: make(map[int]*tField),
	}
	if err := ws.addField(&tField{
		index: IndexId,
		name:  "id",
		typ:   &tTextType{},
	}); err != nil {
		panic("unexpected")
	}
	if err := ws.addField(&tField{
		index: IndexVersion,
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

	for !p.peek(pRacco) {
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

	var computedBy expression
	if p.peek(pComputedBy) {
		_, err = p.nextAndCheck(pComputedBy)
		if err != nil {
			return nil, err
		}

		_, err = p.nextAndCheck(pLacco)
		if err != nil {
			return nil, err
		}

		computedBy, err = p.parseExpression()
		if err != nil {
			return nil, err
		}

		_, err = p.nextAndCheck(pRacco)
		if err != nil {
			return nil, err
		}
	}

	f := &tField{
		index:      index,
		name:       name,
		typ:        typ,
		computedBy: computedBy,
	}

	return f, nil
}

type tExternal struct{}

func (p *parser) parseExpression() (expression, error) {
	_, err := p.nextAndCheck(pExternal)
	if err != nil {
		return nil, err
	}

	return &tExternal{}, nil
}

func (p *parser) parseType() (Type, error) {
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
		_, err := p.nextAndCheck(pLbracket)
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
		_, err = p.nextAndCheck(pRbracket)
		if err != nil {
			return nil, err
		}
		return &tNumberType{scale}, nil
	}

	return nil, fmt.Errorf("unknown type %s", name)
}

func (p *parser) parseLiteral() (Value, error) {
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

func (p *parser) peek(maybes ...*tokenPattern) bool {
	token := p.next()
	p.toks = append(p.toks, token)

	for _, maybe := range maybes {
		if maybe.re.MatchString(token) {
			return true
		}
	}

	return false
}
