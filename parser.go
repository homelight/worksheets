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

var (
	// tokens
	pLacco              = newTokenPattern("{", "\\{")
	pRacco              = newTokenPattern("}", "\\}")
	pLparen             = newTokenPattern("(", "\\(")
	pRparen             = newTokenPattern(")", "\\)")
	pLbracket           = newTokenPattern("[", "\\[")
	pRbracket           = newTokenPattern("]", "\\]")
	pColon              = newTokenPattern(":", "\\:")
	pPlus               = newTokenPattern("+", "\\+")
	pMinus              = newTokenPattern("-", "\\-")
	pMult               = newTokenPattern("*", "\\*")
	pDiv                = newTokenPattern("/", "\\/")
	pNot                = newTokenPattern("!", "\\!")
	pDot                = newTokenPattern(".", "\\.")
	pEqual              = newTokenPattern("==", "\\=\\=")
	pNotEqual           = newTokenPattern("!=", "\\!\\=")
	pGreaterThan        = newTokenPattern(">", "\\>")
	pGreaterThanOrEqual = newTokenPattern(">=", "\\>\\=")
	pLessThan           = newTokenPattern("<", "\\<")
	pLessThanOrEqual    = newTokenPattern("<=", "\\<\\=")
	pAnd                = newTokenPattern("&&", "\\&\\&")
	pOr                 = newTokenPattern("||", "\\|\\|")
	pWorksheet          = newTokenPattern("worksheet", "worksheet")
	pConstrainedBy      = newTokenPattern("constrained_by", "constrained_by")
	pComputedBy         = newTokenPattern("computed_by", "computed_by")
	pExternal           = newTokenPattern("external", "external")
	pUndefined          = newTokenPattern("undefined", "undefined")
	pTrue               = newTokenPattern("true", "true")
	pFalse              = newTokenPattern("false", "false")
	pRound              = newTokenPattern("round", "round")
	pReturn             = newTokenPattern("return", "return")
	pUp                 = newTokenPattern(string(ModeUp), string(ModeUp))
	pDown               = newTokenPattern(string(ModeDown), string(ModeDown))
	pHalf               = newTokenPattern(string(ModeHalf), string(ModeHalf))

	// token patterns
	pName  = newTokenPattern("name", "[a-z]+([a-z_0-9]*[a-z0-9])?")
	pIndex = newTokenPattern("index", "[0-9]+")
	pText  = newTokenPattern("text", "\".*\"")

	pNumber               = newTokenPattern("number", "[0-9]+(\\.[0-9]+)?(\\%)?")
	pNumberWithUnderscore = newTokenPattern("number", "[_0-9]+(\\%)?")
	pNumberWithDot        = newTokenPattern("number", "\\.[0-9]*(\\%)?")
)

func (p *parser) parseWorksheets() (map[string]*Definition, error) {
	wsDefs := make(map[string]*Definition)

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

	return wsDefs, nil
}

func (p *parser) parseWorksheet() (*Definition, error) {
	ws := Definition{
		fieldsByName:  make(map[string]*Field),
		fieldsByIndex: make(map[int]*Field),
	}
	if err := ws.addField(&Field{
		index: IndexId,
		name:  "id",
		typ:   &TextType{},
	}); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
	}
	if err := ws.addField(&Field{
		index: IndexVersion,
		name:  "version",
		typ:   &NumberType{},
	}); err != nil {
		panic(fmt.Sprintf("unexpected %s", err))
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

func (p *parser) parseField() (*Field, error) {
	sIndex, err := p.nextAndCheck(pIndex)
	if err != nil {
		return nil, err
	}
	index := 65536 + 1
	if len(sIndex) <= len("65536") {
		index, err = strconv.Atoi(sIndex)
		if err != nil {
			// unexpected since sIndex should conform to pIndex
			panic(err)
		}
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
	f := &Field{
		index: index,
		name:  name,
		typ:   typ,
	}

	choice, ok := p.peekWithChoice([]*tokenPattern{
		pComputedBy,
		pConstrainedBy,
	}, []string{
		"computed",
		"constrained",
	})

	if ok {
		p.next()

		_, err = p.nextAndCheck(pLacco)
		if err != nil {
			return nil, err
		}

		var expr expression
		expr, err = p.parseStatement()
		if err != nil {
			return nil, err
		}

		_, err = p.nextAndCheck(pRacco)
		if err != nil {
			return nil, err
		}

		switch choice {
		case "computed":
			f.computedBy = expr
		case "constrained":
			f.constrainedBy = expr
		}
	}

	return f, nil

}

// parseStatement
//
//  := 'external'
//   | return parseExpression
func (p *parser) parseStatement() (expression, error) {
	choice, ok := p.peekWithChoice([]*tokenPattern{
		pExternal,
		pReturn,
	}, []string{
		"external",
		"return",
	})
	if !ok {
		return nil, fmt.Errorf("expecting statement")
	}
	switch choice {
	case "external":
		p.next()
		return &tExternal{}, nil

	case "return":
		p.next()
		expr, err := p.parseExpression(true)
		if err != nil {
			return nil, err
		}
		return &tReturn{expr}, nil

	default:
		panic(fmt.Sprintf("nextAndChoice returned '%s'", choice))
	}
}

// parseExpression
//
//  := parseLiteral
//   | var
//   | exp (+ - * /) exp
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
		"ident",
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

	case "ident":
		path := []string{p.next()}
		for p.peek(pDot) {
			p.next()
			name, err := p.nextAndCheck(pName)
			if err != nil {
				return nil, err
			}
			path = append(path, name)
		}
		first = tSelector(path)

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
			pGreaterThan,
			pGreaterThanOrEqual,
			pLessThan,
			pLessThanOrEqual,
			pAnd,
			pOr,
		}, []string{
			string(opPlus),
			string(opMinus),
			string(opMult),
			string(opDiv),
			string(opEqual),
			string(opNotEqual),
			string(opGreaterThan),
			string(opGreaterThanOrEqual),
			string(opLessThan),
			string(opLessThanOrEqual),
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
	opAnd:                1,
	opOr:                 1,
	opEqual:              2,
	opNotEqual:           2,
	opGreaterThan:        2,
	opGreaterThanOrEqual: 2,
	opLessThan:           2,
	opLessThanOrEqual:    2,
	opPlus:               3,
	opMinus:              3,
	opMult:               4,
	opDiv:                5,
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
	choice, ok := p.peekWithChoice([]*tokenPattern{
		pName,
		pLbracket,
	}, []string{
		"base",
		"slice",
	})
	if !ok {
		return nil, fmt.Errorf("expecting type")
	}

	switch choice {
	case "base":
		name, err := p.nextAndCheck(pName)
		if err != nil {
			return nil, err
		}

		switch name {
		case "text":
			return &TextType{}, nil
		case "bool":
			return &BoolType{}, nil
		case "undefined":
			return &UndefinedType{}, nil
		case "number":
			_, err := p.nextAndCheck(pLbracket)
			if err != nil {
				return nil, err
			}
			sScale, err := p.nextAndCheck(pIndex)
			if err != nil {
				return nil, err
			}
			var scale int = 32 + 1
			if len(sScale) <= len("32") {
				scale, err = strconv.Atoi(sScale)
				if err != nil {
					// unexpected since sScale should conform to pIndex
					panic(err)
				}
			}
			if scale > 32 {
				return nil, fmt.Errorf("scale cannot be greater than 32")
			}
			_, err = p.nextAndCheck(pRbracket)
			if err != nil {
				return nil, err
			}
			return &NumberType{scale}, nil
		default:
			return &Definition{name: name}, nil
		}

	case "slice":
		p.next()
		_, err := p.nextAndCheck(pRbracket)
		if err != nil {
			return nil, err
		}

		elementType, err := p.parseType()
		if err != nil {
			return nil, err
		}

		return &SliceType{elementType}, nil

	default:
		panic(fmt.Sprintf("unknown choice %s", choice))
	}
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
			if strings.HasSuffix(token, "%") {
				return nil, fmt.Errorf("number must terminate with percent if present")
			}
			token = token + strings.Replace(addToken, "_", "", -1)
		}

		// note whether percent, then remove to keep dot-index calcs correct
		isPct := strings.HasSuffix(token, "%")
		token = strings.TrimRight(token, "%")

		dot := strings.Index(token, ".")
		value, err := strconv.ParseInt("-"+strings.Replace(token, ".", "", 1), 10, 64)
		if err != nil {
			return nil, err
		}
		var scale int
		if dot < 0 {
			scale = 0
		} else {
			scale = len(token) - dot - 1
		}

		if isPct {
			scale += 2
		}
		if !negNumber {
			value = -value
		}
		return &Number{value, &NumberType{scale}}, nil
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
	"<": "=",
	">": "=",
	"&": "&",
	"|": "|",
}

func (p *parser) next() string {
	if len(p.toks) == 0 {
		p.s.Scan()
		token := p.s.TokenText()

		// will need to revisit when we implement mod operator
		if p.s.Peek() == '%' {
			return token + string(p.s.Next())
		}

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
