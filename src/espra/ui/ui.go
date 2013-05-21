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
)

var bodyNode = &html.Node{Data: "body", DataAtom: atom.Body, Type: html.ElementNode}

const (
	leftDelim  = "{{"
	rightDelim = "}}"
)

const (
	ItemText lex.ItemType = 2 + iota
	ItemIdentifier
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

func Builtin(l *lex.Lexer) lex.StateFn {
	for {
		if !IsValidIdentifierChar(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemBuiltin)
	return InsideAction
}

func Identifier(l *lex.Lexer) lex.StateFn {
	for {
		if !IsValidIdentifierChar(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemIdentifier)
	return InsideAction
}

func Number(l *lex.Lexer) lex.StateFn {
	//
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
	// first identifier may be a func or a var --
	// then everything is a var unless its the first identifier inside parens the its a func

	// add keyword key:var
	// add espraURI func or attr
	// Number

	// add if and for -- HTML5 parser ordered?

	// deal with errors nicely
	// reuse/reset the lexer

	if strings.HasPrefix(l.Input[l.Pos:], rightDelim) {
		if l.IntState[ActionParenDepth] > 0 {
			return l.Errorf("unmatched parentheses")
		}
		return RightDelim
	}

	switch r := l.Next(); {
	case r == lex.EOF || lex.IsEndOfLine(r):
		// if reach eof throw while still in action throw error
		return l.Errorf("unclosed action")
	case lex.IsSpace(r):
		return Space
	case unicode.IsLetter(r): //variable and function must begin with a letter
		return Identifier
	case r == '!':
		if unicode.IsLetter(l.Peek()) {
			return Builtin
		}
		return l.Errorf("invalid character in builtin")
	case r == '#' || r == '+':
		if unicode.IsLetter(l.Peek()) {
			return EspraURI
		}
		return l.Errorf("invalid character in URI")
	case unicode.IsDigit(r):
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

// Space scans a run of space characters.
// One space has already been seen.
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
	ItemString:     "STR",
	ItemNumber:     "INT",
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

func ParseTextNode(input string) (db.Domly, error) {

	l := createLexer("LextTextNode", input, LexTextNode)
	for item := range l.Items {
		fmt.Printf("Type: %20s\t %q\n", debugNames[item.Typ], item.Val)
		if item.Typ == lex.ItemEOF {
			break
		}
		if item.Typ == lex.ItemError {
			return db.Domly{}, error(fmt.Errorf("Failed: %s", item.Val))
		}
	}
	return db.Domly{}, nil

}

func GenDomlyNode(domNode *html.Node) (db.Domly, error) {
	data := db.Domly{}
	var textNodeDomly db.Domly
	var node db.Domly

	var errr error
	errr = nil
	if domNode.Type == html.ElementNode {
		data = append(data, domNode.Data)
		if len(domNode.Attr) != 0 { // != nil {
			attrs := db.DomlyAttrs{}
			for _, attr := range domNode.Attr {
				key := attr.Key
				if len(replacements[key]) > 0 {
					key = replacements[key]
				}
				if domNode.Data == "label" && key == "for" {
					key = "htmlFor"
				}

				attrs[key] = attr.Val
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
				textNodeDomly, errr = ParseTextNode(c.Data) // for each token parse attribute values and text node
				if errr != nil {
					return data, errr
				}
				data = append(data, textNodeDomly)
			}
		}
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
