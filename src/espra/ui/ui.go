// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package ui

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"encoding/json"
	"espra/db"
	"espra/lex"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

var bodyNode = &html.Node{Data: "body", DataAtom: atom.Body, Type: html.ElementNode}

const (
	leftDelim  = "{{"
	rightDelim = "}}"
)

const (
	ItemText lex.ItemType = 2 + iota
	ItemIdentifier
	ItemArgKey
	ItemString
	ItemNumber
	ItemSpace
	ItemLeftDelim
	ItemRightDelim
	ItemBuiltin
	ItemEspraURI
	ItemPipe
	ItemLeftParen
	ItemRightParen
	ItemOpenQuote
	ItemCloseQuote
	ItemIf
	ItemIn
	ItemFor
)
const (
	ActionParenDepth = iota
)

var replacements = map[string]string{
	"cellpadding":     "cellPadding",
	"cellspacing":     "cellSpacing",
	"class":           "className",
	"colspan":         "colSpan",
	"contenteditable": "contentEditable",
	"frameborder":     "frameBorder",
	"maxlength":       "maxLength",
	"readonly":        "readOnly",
	"rowspan":         "rowSpan",
	"tabindex":        "tabIndex",
	"usemap":          "useMap",
}

func ParseHTML5(filename string) ([]*html.Node, error) {
	reader, err := os.Open(filename)
	frag, err := html.ParseFragment(reader, bodyNode)
	return frag, err
}

func createLexer(name, input string, startfn lex.StateFn) *lex.Lexer {
	// conf = map{}  -- add conf map to the lexer to allow lexer configuration
	l := &lex.Lexer{
		Name:  name,
		Input: input,
		Items: make(chan lex.Item),
		IntState: lex.IntState{
			ActionParenDepth: 0,
		},
	}
	go l.Run(startfn)
	return l
}

func IsValidIdentifierChar(r rune) bool {
	if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' || r == '-' || r == '.' {
		return true
	}
	return false
}

func EspraURI(l *lex.Lexer) lex.StateFn {
	l.Emit(ItemEspraURI)
	return InsideAction
}

func Identifier(l *lex.Lexer) lex.StateFn {
	typ := ItemIdentifier
	r, _ := utf8.DecodeRuneInString(l.Input[l.Start:])
	if r == '!' {
		l.Ignore()
		typ = ItemBuiltin
	}
	for {
		// should not appear in the first identifier of an expression of sub-expression (since it must be a function)
		r = l.Next()

		if r == ':' {
			typ = ItemArgKey
			l.Backup()
			l.Emit(typ)
			l.Next()
			l.Ignore()
			return InsideAction
		}
		if !IsValidIdentifierChar(r) {
			l.Backup()
			break
		}
	}
	l.Emit(typ)
	return InsideAction
}

func Number(l *lex.Lexer) lex.StateFn {
	if !l.ScanNumber() {
		return l.Errorf("bad number syntax: %q", l.Input[l.Start:l.Pos])
	}
	l.Emit(ItemNumber)
	return InsideAction
}

// Quote scans a quoted string.
func String(l *lex.Lexer) lex.StateFn {
Loop:
	for {
		switch l.Next() {
		case '\\':
			// ???
			if r := l.Next(); r != lex.EOF && r != '\n' {
				break
			}
			fallthrough
		case lex.EOF, '\n':
			return l.Errorf("unterminated quoted string")
		case '"', '\'':
			l.Backup()
			break Loop
		}
	}
	l.Emit(ItemString)
	l.Next()
	l.Emit(ItemCloseQuote)
	return InsideAction
}

func InsideAction(l *lex.Lexer) lex.StateFn {

	if l.Name == "LexTextNode" && strings.HasPrefix(l.Input[l.Pos:], rightDelim) {
		if l.IntState[ActionParenDepth] > 0 {
			return l.Errorf("unmatched parentheses")
		}
		return RightDelim
	}

	switch r := l.Next(); {
	case (r == lex.EOF || lex.IsEndOfLine(r)):
		if l.Name == "LexIfExpr" {
			return LexIfExpr
		}
		if l.Name == "LexForExpr" {
			return LexForExpr
		}
		// if reach eof throw while still in action throw error
		return l.Errorf("unclosed action")
	case lex.IsSpace(r):
		return Space
	case unicode.IsLetter(r): //variable and function must begin with a letter
		return Identifier
	case r == '!':
		if unicode.IsLetter(l.Peek()) {
			return Identifier
		}
		return l.Errorf("invalid character in builtin")
	case r == '#' || r == '+':
		if unicode.IsLetter(l.Peek()) {
			return EspraURI
		}
		return l.Errorf("invalid character in URI")
	case r == '-' || unicode.IsDigit(r):
		l.Backup()
		return Number
	case r == '\'' || r == '"':
		l.Emit(ItemOpenQuote)
		return String
	case r == '(':
		l.IntState[ActionParenDepth] += 1
		l.Emit(ItemLeftParen)
	case r == ')':
		l.IntState[ActionParenDepth] -= 1
		l.Emit(ItemRightParen)
	case r == '|':
		l.Emit(ItemPipe)
	default:
		return l.Errorf("Unexpected Character '%s'", string(r))
	}
	return InsideAction
}

func Space(l *lex.Lexer) lex.StateFn {
	for lex.IsSpace(l.Peek()) {
		l.Next()
	}
	l.Emit(ItemSpace)

	return InsideAction
}

// RightDelim scans the right delimiter, which is known to be present.
func RightDelim(l *lex.Lexer) lex.StateFn {
	l.Pos += lex.Pos(len(rightDelim))
	l.Emit(ItemRightDelim)
	return LexTextNode
}

const (
	StateLeftDelimLen = iota
)

// LeftDelim scans the left delimiter, which is known to be present.
func LeftDelim(l *lex.Lexer) lex.StateFn {
	l.Pos += lex.Pos(len(leftDelim))
	l.Emit(ItemLeftDelim)
	return InsideAction
}

func LexIfExpr(l *lex.Lexer) lex.StateFn {
	if int(l.Pos) >= len(l.Input) {
		l.Emit(lex.ItemEOF)
		return nil
	}
	return InsideAction

}

func LoopIdentifier(l *lex.Lexer) lex.StateFn {
	typ := ItemIdentifier
	for {
		r := l.Next()
		if !IsValidIdentifierChar(r) {
			l.Backup()
			break
		}
	}
	l.Emit(typ)
	return LexForExpr
}

func LexForExpr(l *lex.Lexer) lex.StateFn {
	if int(l.Pos) >= len(l.Input) {
		l.Emit(lex.ItemEOF)
		return nil
	}
	if strings.HasPrefix(l.Input[l.Pos:], "in") {
		l.Pos += lex.Pos(2)
		l.Emit(ItemIn)
		return InsideAction
	}

	// any list of space separated identifiers
	switch r := l.Next(); {
	case lex.IsSpace(r):
		for lex.IsSpace(l.Peek()) {
			l.Next()
		}
		l.Emit(ItemSpace)
		return LexForExpr
	case unicode.IsLetter(r):
		return LoopIdentifier
	default:
		return l.Errorf("Unexpected Character '%s'", string(r))
	}
}

func LexTextNode(l *lex.Lexer) lex.StateFn {
	for {
		if strings.HasPrefix(l.Input[l.Pos:], leftDelim) {
			if l.Pos > l.Start {
				l.Emit(ItemText)
			}
			return LeftDelim
		}
		if l.Next() == lex.EOF {
			break
		}
	}
	// Correctly reached EOF.
	if l.Pos > l.Start {
		l.Emit(ItemText)
	}
	l.Emit(lex.ItemEOF)
	return nil
}

var debugNames = map[lex.ItemType]string{
	lex.ItemError:  "ERROR",
	lex.ItemEOF:    "EOF",
	ItemText:       "TEXT",
	ItemIdentifier: "IDENTIFIER",
	ItemArgKey:     "KEY",
	ItemString:     "STRING",
	ItemNumber:     "NUMBER",
	ItemSpace:      "SPACE",
	ItemLeftDelim:  "LEFT DELIM",
	ItemRightDelim: "RIGHT DELIM",
	ItemBuiltin:    "BUILTIN",
	ItemEspraURI:   "URI IDENTIFIER",
	ItemPipe:       "PIPE",
	ItemLeftParen:  "LEFT PARENS",
	ItemRightParen: "RIGHT PARENS",
	ItemOpenQuote:  "QUOTE OPEN",
	ItemCloseQuote: "QUOTE CLOSE",
}

func isTerminal(typ lex.ItemType) bool {
	if typ == ItemString || typ == ItemNumber || typ == ItemIdentifier || typ == ItemBuiltin || typ == ItemEspraURI {
		return true
	}
	return false
}

type EOFError error

func ParseArgKeyVal(l *lex.Lexer) (db.Domly, error) {
	// get the value of a key-value argument of the form {{A B:C}}
	for {
		item := <-l.Items

		switch typ := item.Typ; {
		case typ == ItemLeftParen:
			return ParseExpr(l, ItemRightParen)

		case isTerminal(typ):
			return db.Domly{typ, item.Val}, nil
		case typ == ItemSpace:
			continue
		default:
			return db.Domly{}, error(fmt.Errorf("Unexpected token %v in keyword value", item.Typ))
		}
	}
}

func ParseExpr(l *lex.Lexer, endDelimToken lex.ItemType) (db.Domly, error) {
	// Opcodes are given as ['expr', {'key':val}, [ItemIdentifier, X],  [ItemIdentifier, Y], [ItemString, "string"] within an action without parens (or pipe) all identifiers are sequentially added to the same list

	var item lex.Item
	attrs := db.DomlyAttrs{}         // key-value arguments of the form {{A B:C}} are added to the Domly Attrs of the node
	domly := db.Domly{"expr", attrs} //domly list for this level,
	seen_argkey := false
	first := true

	for {
		item = <-l.Items
		fmt.Printf("Type: %20s\t %q\n", debugNames[item.Typ], item.Val)

		switch typ := item.Typ; {
		case typ == ItemLeftParen:
			expr, err := ParseExpr(l, ItemRightParen)
			if err != nil {
				return domly, err
			}
			domly = append(domly, expr)

		case isTerminal(typ):
			// each terminal is appended as a domly sub-node [OpCodeInt, item.Val] except for
			// if there are more than one terminals in an expression, the first must be an identifier since it must be a function (in the future this may change with the addition of binary operators)
			if first != true {
				first_domly := domly[2].(db.Domly)
				first_typ := first_domly[0].(lex.ItemType)
				first_val := first_domly[1].(string)
				if first_typ == ItemString || first_typ == ItemNumber {
					return domly, error(fmt.Errorf("Value '%v' Type '%v' cannot be evaluated as a function with arguments", first_val, debugNames[first_typ]))
				}
			}
			if seen_argkey {
				return domly, error(fmt.Errorf("argument cannot follow keyword argument"))
			} else {
				first = false
				domly = append(domly, db.Domly{typ, item.Val})
			}

		case typ == ItemArgKey:
			// the value to a keyword arg can either be a terminal or the outcome of an expression in which case it must be put in parentheses.
			// the key:val is added to the Domly Attrs of the node
			seen_argkey = true
			if first == true {
				return domly, error(fmt.Errorf("keyword arguments with no function"))
			}
			val, err := ParseArgKeyVal(l)
			if err != nil {
				return domly, err
			}
			attrs, boolean := domly[1].(db.DomlyAttrs)
			if boolean {
				attrs[item.Val] = val
			}

		case typ == endDelimToken:
			// return the domly ast for the subexpression
			return domly, nil

		case typ == ItemRightDelim || typ == ItemRightParen || typ == lex.ItemEOF:
			// expression ended wrong
			return domly, error(fmt.Errorf("expected '%v' but got '%v'", debugNames[endDelimToken], debugNames[typ]))

		case typ == ItemPipe:
			// A|B === B(A) So when we see a pipe signal we put the previous identifer list as an argument to the current
			pipe_receiver, err := ParseExpr(l, endDelimToken)
			if err != nil {
				return domly, err
			}
			// the domly expression up to now becomes an argument to the pipe_receiver
			//return db.Domly{pipe_receiver, domly}, nil
			return append(pipe_receiver, domly), nil

		case typ == ItemLeftDelim || typ == ItemText:
			return domly, error(fmt.Errorf("Unexpected %s inside an expression", debugNames[typ]))

		case typ == lex.ItemError:
			return db.Domly{}, error(fmt.Errorf("Lexer Failed: %s", item.Val))

		}
	}
}

func ParseIfExpr(iftext string) (db.Domly, error) {
	// <a if="expr">
	l := createLexer("LexIfExpr", iftext, LexIfExpr)
	return ParseExpr(l, lex.ItemEOF)
}

func ParseForExpr(fortext string) (db.Domly, error) {
	// e.g. <a for="x y z in expr">{{x.name}}</a>
	l := createLexer("LexForExpr", fortext, LexForExpr)
	loopVars := db.Domly{}
	domly := db.Domly{}
	var err error
	var item lex.Item
	for {
		item = <-l.Items
		switch typ := item.Typ; {
		case typ == ItemIdentifier:
			loopVars = append(loopVars, item.Val)
		case typ == ItemIn:
			domly, err = ParseExpr(l, lex.ItemEOF)
			return db.Domly{loopVars, domly}, err
		case typ == ItemSpace:
			continue
		// should always hit the EOF in ParseExpr
		case typ == lex.ItemEOF:
			return db.Domly{}, error(fmt.Errorf("Unexpected EOF in : %s", fortext))
		case typ == lex.ItemError:
			return db.Domly{}, error(fmt.Errorf("Lexer Failed: %s", item.Val))
		default:
			return db.Domly{}, error(fmt.Errorf("Unexpected token in : %s", fortext))
		}
	}
}

func ParseTextNode(input string) (db.Domly, error) {
	// in TextNode use <span> in attribute use <strip> tag that strips itself - js execution framework

	l := createLexer("LexTextNode", input, LexTextNode)
	var item lex.Item
	var domly = db.Domly{} //domly list for this level,

	for {
		item = <-l.Items
		switch typ := item.Typ; {
		case typ == ItemText:
			//ItemText is inserted directly as a string
			domly = append(domly, item.Val)
			fmt.Printf("val %v\n", domly)
		case typ == ItemLeftDelim:
			expr, err := ParseExpr(l, ItemRightDelim)
			if err != nil {
				return domly, err
			}
			domly = append(domly, expr)
		case typ == lex.ItemEOF:
			if len(domly) == 1 {
				return domly, nil
			}
			return domly, nil
		case typ == lex.ItemError:
			return db.Domly{}, error(fmt.Errorf("Lexer Failed: %s", item.Val))

		}
	}
}

func extractVal(parsedTextNode db.Domly) interface{} {
	inner := ""
	if len(parsedTextNode) == 1 {
		str, is_str := parsedTextNode[0].(string)
		if is_str {
			inner = str
		}
	}
	if inner != "" {
		return inner
	}
	return parsedTextNode
}

func GenDomlyNode(domNode *html.Node) (db.Domly, error) {
	data := db.Domly{}
	data_wrappers := []db.Domly{}
	var textNodeDomly db.Domly
	var node db.Domly

	var errr error
	errr = nil

	if domNode.Type == html.ElementNode {
		data = append(data, domNode.Data)
		if len(domNode.Attr) != 0 {
			attrs := db.DomlyAttrs{}
			for _, attr := range domNode.Attr {
				key := attr.Key
				if len(replacements[key]) > 0 {
					key = replacements[key]
				}
				if domNode.Data == "label" && key == "for" {
					key = "htmlFor"
				}
				if key == "if" {
					if_expr, err := ParseIfExpr(attr.Val)
					if err != nil {
						return if_expr, err
					}
					data_wrappers = append(data_wrappers, db.Domly{ItemIf, db.DomlyAttrs{"if": if_expr}})
				} else if key == "for" {
					for_expr, err := ParseForExpr(attr.Val)
					if err != nil {
						return for_expr, err
					}
					data_wrappers = append(data_wrappers, db.Domly{ItemFor, db.DomlyAttrs{"for": for_expr[0], "in": for_expr[1]}})
				} else {
					textNodeDomly, errr = ParseTextNode(attr.Val)
					if errr != nil {
						return textNodeDomly, errr
					}
					attrs[key] = extractVal(textNodeDomly)
				}
			}
			data = append(data, attrs)
		}
		for c := domNode.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				node, errr = GenDomlyNode(c)
				if errr != nil {
					return data, errr
				}
				data = append(data, node)
			} else if c.Type == html.TextNode {
				textNodeDomly, errr = ParseTextNode(c.Data) // for each token parse attribute values and text node)
				if errr != nil {
					return data, errr
				}
				data = append(data, extractVal(textNodeDomly))
			}
		}
	}
	// apply if and for wrappers
	// not ideal as it breaks tail recursion
	for i := len(data_wrappers) - 1; i >= 0; i-- {
		data = append(data_wrappers[i], data)
	}
	return data, nil
}

func GenDomly(domfrag []*html.Node) (db.Domly, error) {
	// The db.Domly format looks like: [tagName, attr1:val1, attr2:val2..., 'Text' | ChildNodes ]
	// what to do with TextNodes mixed with nested tagged content e.g. "sdfds ds <b>sdgtd </b>". Is there an explicit TextNode domly expression?

	data := db.Domly{}
	var node_data db.Domly
	var err error
	for _, node := range domfrag {
		if node.Type == html.ElementNode {
			node_data, err = GenDomlyNode(node)
			data = append(data, node_data)
		}
	}
	return data, err
}

func GenJSON(data db.Domly) ([]byte, error) {
	enc, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("JSON encoding error")
		return []byte{}, err
	}
	return enc, nil
}

/* <img src="htt.." class="{{foo}}" alt="Check out {{blah.name|upper|xx}} today!">

	{"src": "http://...", "className": [[1, "foo"]], "alt": ["Check out ", [1, "blah.name", "upper", "xx"], " today!"]}}

templateData

for attr, val of attrs
  if isArray(val)
    out = []
    for v in val
      if isArray(v)
        k = v[1]
        ctx = templateData
        for splitKey in k.split('.')
          ctx = ctx[splitKey]
        for func in v[2...]
          ctx = builtins[func](ctx)
        out.push(ctx)
      else
        out.push(v)
    val = ''.join(out)
  dom.setAttribute(attr, val)
*/

func DomTree2HTML(DOMTree []*html.Node) {
	HTML5 := bytes.NewBuffer([]byte{})
	for _, node := range DOMTree {
		html.Render(HTML5, node)
	}
	fmt.Printf("Node: %s", HTML5)
}

func ParseTemplate(template_path string) ([]byte, error) {
	DOMTree, err := ParseHTML5(template_path)

	// use crash recover?
	if err != nil {
		fmt.Printf("HTML rendering error")
	}

	DomTree2HTML(DOMTree) //print the parsed HTML

	data, err := GenDomly(DOMTree)
	if err != nil {
		fmt.Printf("%v", err)
		return []byte{}, err
	}
	fmt.Printf("%s", data)

	json, err := GenJSON(data)
	if err != nil {
		return json, err
	}
	return json, nil
}
