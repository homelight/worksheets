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
	dependents map[int][]int
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

var (
	// tokens
	pLacco      = newTokenPattern("{", "\\{")
	pRacco      = newTokenPattern("}", "\\}")
	pLparen     = newTokenPattern("(", "\\(")
	pRparen     = newTokenPattern(")", "\\)")
	pLbracket   = newTokenPattern("[", "\\[")
	pRbracket   = newTokenPattern("]", "\\]")
	pColon      = newTokenPattern(":", "\\:")
	pPlus       = newTokenPattern("+", "\\+")
	pMinus      = newTokenPattern("-", "\\-")
	pMult       = newTokenPattern("*", "\\*")
	pDiv        = newTokenPattern("/", "\\/")
	pNot        = newTokenPattern("!", "\\!")
	pEqual      = newTokenPattern("==", "\\=\\=")
	pNotEqual   = newTokenPattern("!=", "\\!\\=")
	pAnd        = newTokenPattern("&&", "\\&\\&")
	pOr         = newTokenPattern("||", "\\|\\|")
	pWorksheet  = newTokenPattern("worksheet", "worksheet")
	pComputedBy = newTokenPattern("computed_by", "computed_by")
	pExternal   = newTokenPattern("external", "external")
	pUndefined  = newTokenPattern("undefined", "undefined")
	pTrue       = newTokenPattern("true", "true")
	pFalse      = newTokenPattern("false", "false")
	pRound      = newTokenPattern("round", "round")
	pUp         = newTokenPattern(string(ModeUp), string(ModeUp))
	pDown       = newTokenPattern(string(ModeDown), string(ModeDown))
	pHalf       = newTokenPattern(string(ModeHalf), string(ModeHalf))

	// token patterns
	pName  = newTokenPattern("name", "[a-z]+([a-z_]*[a-z])?")
	pIndex = newTokenPattern("index", "[0-9]+")
	pText  = newTokenPattern("text", "\".*\"")

	pNumber               = newTokenPattern("number", "[0-9]+(\\.[0-9]+)?")
	pNumberWithUnderscore = newTokenPattern("number", "[_0-9]+")
	pNumberWithDot        = newTokenPattern("number", "\\.[0-9]*")
)

func (p *parser) parseWorksheets() (map[string]*tWorksheet, error) {
	wsDefs := make(map[string]*tWorksheet)

	for p.peek(pWorksheet) {
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

		computedBy, err = p.parseExpressionOrExternal()
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

type tOp string

const (
	opPlus     tOp = "plus"
	opMinus        = "minus"
	opMult         = "mult"
	opDiv          = "div"
	opNot          = "not"
	opEqual        = "equal"
	opNotEqual     = "not-equal"
	opOr           = "or"
	opAnd          = "and"
)

type tRound struct {
	mode  RoundingMode
	scale int
}

func (t *tRound) String() string {
	return fmt.Sprintf("%s %d", t.mode, t.scale)
}

type tExternal struct{}

type tUnop struct {
	op   tOp
	expr expression
}

type tBinop struct {
	op          tOp
	left, right expression
	round       *tRound
}

func (t *tBinop) String() string {
	return fmt.Sprintf("binop(%s, %s, %s, %s)", t.op, t.left, t.right, t.round)
}

type tVar struct {
	name string
}

// parseExpressionOrExternal
//
//  := 'external'
//   | parseExpression
func (p *parser) parseExpressionOrExternal() (expression, error) {
	choice, ok := p.peekWithChoice([]*tokenPattern{
		pExternal,
		pUndefined,
		pTrue,
		pFalse,
		pNumber,
		pMinus,
		pText,
		pName,
		pLparen,
		pNot,
	}, []string{
		"external",
		"expr",
		"expr",
		"expr",
		"expr",
		"expr",
		"expr",
		"expr",
		"expr",
		"expr",
	})
	if !ok {
		return nil, fmt.Errorf("expecting expression or external")
	}
	switch choice {
	case "external":
		p.next()
		return &tExternal{}, nil

	case "expr":
		return p.parseExpression(true)

	default:
		panic(fmt.Sprintf("nextAndChoice returned '%s'", choice))
	}
}

// parseExpression
//
//  := parseLiteral
//   | var
//   | exp (+ -  * /) exp
func (p *parser) parseExpression(withOp bool) (expression, error) {
	choice, ok := p.peekWithChoice([]*tokenPattern{
		pUndefined,
		pTrue,
		pFalse,
		pNumber,
		pNumberWithDot,
		pNumberWithUnderscore,
		pMinus,
		pText,
		pName,
		pLparen,
		pNot,
	}, []string{
		"literal",
		"literal",
		"literal",
		"literal",
		"literal",
		"literal",
		"literal",
		"literal",
		"var",
		"paren",
		"unop",
	})
	if !ok {
		return nil, fmt.Errorf("expecting expression")
	}

	// first
	var first expression
	switch choice {
	case "literal":
		val, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		first = val.(expression)

	case "var":
		token := p.next()
		first = &tVar{token}

	case "paren":
		p.next()

		expr, err := p.parseExpression(true)
		if err != nil {
			return nil, err
		}
		first = expr

		if _, err := p.nextAndCheck(pRparen); err != nil {
			return nil, err
		}

	case "unop":
		op, ok := p.peekWithChoice([]*tokenPattern{
			pNot,
		}, []string{
			string(opNot),
		})
		if !ok {
			panic("should not be in unop")
		}
		p.next()

		expr, err := p.parseExpression(true)
		if err != nil {
			return nil, err
		}
		first = &tUnop{tOp(op), expr}

	default:
		panic(fmt.Sprintf("nextAndChoice returned '%s'", choice))
	}

	if !withOp {
		return first, nil
	}

	if p.peek(pRound) {
		round, err := p.parseRound()
		if err != nil {
			return nil, err
		}
		first = &tBinop{opPlus, first, vZero, round}
	}

	// more?
	var (
		exprs  []expression
		ops    []tOp
		rounds [][]*tRound
	)
	for {
		op, ok := p.peekWithChoice([]*tokenPattern{
			pPlus,
			pMinus,
			pMult,
			pDiv,
			pEqual,
			pNotEqual,
			pAnd,
			pOr,
		}, []string{
			string(opPlus),
			string(opMinus),
			string(opMult),
			string(opDiv),
			string(opEqual),
			string(opNotEqual),
			string(opAnd),
			string(opOr),
		})
		if !ok {
			if exprs == nil {
				return first, nil
			} else {
				return foldExprs(exprs, ops, rounds), nil
			}
		}

		if exprs == nil {
			exprs = []expression{first}
		}

		p.next()
		ops = append(ops, tOp(op))

		next, err := p.parseExpression(false)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, next)

		// roundings?
		var roundsForOp []*tRound
		for p.peek(pRound) {
			round, err := p.parseRound()
			if err != nil {
				return nil, err
			}
			roundsForOp = append(roundsForOp, round)
		}
		rounds = append(rounds, roundsForOp)
	}
}

var opPrecedence = map[tOp]int{
	opAnd:      1,
	opOr:       1,
	opEqual:    2,
	opNotEqual: 2,
	opPlus:     3,
	opMinus:    3,
	opMult:     4,
	opDiv:      5,
}

// foldExprs folds expressions separated by operators by respecting the
// operator precedence rules.
//
// Implementation note: The `exprs` array has one more element than the
// `ops` array at all times. The operator at index `i` joins the expressions
// at index `i` and `i+1`. The algorithm folds left by iteratively finding
// a local maxima for operator precedence -- i.e. a place in the `ops` array
// where the left and right are lower than the operator to fold.
func foldExprs(exprs []expression, ops []tOp, rounds [][]*tRound) expression {
folding:
	for {
		for i, end := 0, len(ops)-1; i <= end; i++ {
			left := (i == 0)
			if !left {
				left = opPrecedence[ops[i-1]] <= opPrecedence[ops[i]]
			}

			right := (i == end)
			if !right {
				right = opPrecedence[ops[i]] >= opPrecedence[ops[i+1]]
			}

			if left && right {
				var round *tRound

				// TODO(pascal): properly folding roundings requires an
				// explanation. It is not trivial.
				for j := i; j < len(rounds); j++ {
					if len(rounds[j]) != 0 {
						round = rounds[j][0]
						rounds[j] = rounds[j][1:]
						j = j - 1
						for 0 <= j && len(rounds[j]) != 0 {
							for _, remainderRound := range rounds[j] {
								exprs[j+1] = &tBinop{opPlus, exprs[j+1], vZero, remainderRound}
							}
							rounds[j] = nil
							j--
						}
						break
					}
				}

				folded := &tBinop{ops[i], exprs[i], exprs[i+1], round}
				if end == 0 {
					return folded
				}

				ops = append(ops[:i], ops[i+1:]...)
				exprs = append(exprs[:i], exprs[i+1:]...)
				exprs[i] = folded

				continue folding
			}
		}
	}
}

func (p *parser) parseRound() (*tRound, error) {
	if _, err := p.nextAndCheck(pRound); err != nil {
		return nil, err
	}

	mode, ok := p.peekWithChoice([]*tokenPattern{
		pUp,
		pDown,
		pHalf,
	}, []string{
		string(ModeUp),
		string(ModeDown),
		string(ModeHalf),
	})
	if !ok {
		return nil, fmt.Errorf("expecting rounding mode (up, down, or half)")
	}
	p.next()

	sIndex, err := p.nextAndCheck(pIndex)
	if err != nil {
		return nil, err
	}
	index, err := strconv.Atoi(sIndex)
	if err != nil {
		// unexpected since sIndex should conform to pIndex
		panic(err)
	}

	return &tRound{RoundingMode(mode), index}, nil
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
		return &Undefined{}, nil
	case "true":
		return &Bool{true}, nil
	case "false":
		return &Bool{false}, nil
	case "-":
		negNumber = true
		token, err = p.nextAndCheck(pNumber)
		if err != nil {
			return nil, err
		}
	}
	if pNumber.re.MatchString(token) {
		for p.peek(pNumberWithUnderscore) || p.peek(pNumberWithDot) {
			addToken := p.next()
			if strings.HasSuffix(addToken, "_") {
				return nil, fmt.Errorf("number cannot terminate with underscore")
			}
			if strings.HasSuffix(addToken, ".") {
				if p.peek(pNumberWithUnderscore) {
					return nil, fmt.Errorf("number fraction cannot start with underscore")
				}
				return nil, fmt.Errorf("number cannot terminate with dot")
			}
			token = token + strings.Replace(addToken, "_", "", -1)
		}
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
		return &Number{value, &tNumberType{scale}}, nil
	}
	if pText.re.MatchString(token) {
		value, err := strconv.Unquote(token)
		if err != nil {
			return nil, err
		}
		return &Text{value}, nil
	}
	if pNumberWithUnderscore.re.MatchString(token) {
		return nil, fmt.Errorf("number cannot start with underscore")
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

var tokensToCombine = map[string]string{
	"=": "=",
	"!": "=",
	"&": "&",
	"|": "|",
}

func (p *parser) next() string {
	if len(p.toks) == 0 {
		p.s.Scan()
		token := p.s.TokenText()

		second, ok := tokensToCombine[token]
		if !ok {
			return token
		}

		first := token
		firstPos := p.s.Position
		p.s.Scan()
		seconPos := p.s.Position
		token = p.s.TokenText()
		if token == second && firstPos.Line == seconPos.Line && firstPos.Column == seconPos.Column-1 {
			return first + second
		}
		p.toks = append(p.toks, token)
		return first
	} else {
		token := p.toks[len(p.toks)-1]
		p.toks = p.toks[:len(p.toks)-1]
		return token
	}
}

func (p *parser) peek(maybe *tokenPattern) bool {
	token := p.next()
	p.toks = append(p.toks, token)

	return maybe.re.MatchString(token)
}

// peekWithChoice peeks, and matches against a set of possible tokens. When a
// match is found, it returns the choice in the choice array corresponding to
// the index of the token in the maybes array.
//
// We use two arrays here, rather than a map, to guarantee a prioritized
// selection of the choices.
func (p *parser) peekWithChoice(maybes []*tokenPattern, choices []string) (string, bool) {
	if len(maybes) != len(choices) {
		panic("peekWithChoice invoked with maybes not equal to choices")
	}

	token := p.next()
	p.toks = append(p.toks, token)

	for index, maybe := range maybes {
		if maybe.re.MatchString(token) {
			return choices[index], true
		}
	}
	return "", false
}
