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
)

var bodyNode = &html.Node{Data: "body", DataAtom: atom.Body, Type: html.ElementNode}

const (
	leftDelim  = "{{"
	rightDelim = "}}"
)

const (
	ItemText lex.ItemType = 2 + iota
	ItemVar
	ItemSpace
	ItemLeftDelim
	ItemRightDelim
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
		/* IntState: lex.IntState{
			StateLeftDelimLen: len(leftDelim),
		}, */
	}
	go l.Run(startfn)
	return l
}

func Variable(l *lex.Lexer) lex.StateFn {
	for {
		if !lex.IsAlphaNumeric(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemVar)
	return InsideAction
}

func InsideAction(l *lex.Lexer) lex.StateFn {
	if strings.HasPrefix(l.Input[l.Pos:], rightDelim) {
		return RightDelim
	}

	switch r := l.Next(); {
	case r == lex.EOF || lex.IsEndOfLine(r):
		// if reach eof throw while still in action throw error
		return l.Errorf("unclosed action")
	case lex.IsSpace(r):
		return Space
	case lex.IsAlphaNumeric(r):
		l.Backup()
		return Variable
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
	ItemVar:        "VAR",
	ItemSpace:      "SPACE",
	ItemLeftDelim:  "LEFT DELIM",
	ItemRightDelim: "RIGHT DELIM",
}

func ParseTextNode(input string) db.Domly {

	l := createLexer("LextTextNode", input, LexTextNode)
	for item := range l.Items {
		fmt.Printf("Type: %20s\t %q\n", debugNames[item.Typ], item.Val)
		if item.Typ == lex.ItemEOF {
			break
		}
	}
	return db.Domly{}

}

func GenDomlyNode(domNode *html.Node) db.Domly {
	data := db.Domly{}
	var textNodeDomly db.Domly
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
				data = append(data, GenDomlyNode(c))
			} else if c.Type == html.TextNode {
				textNodeDomly = ParseTextNode(c.Data) // for each token parse attribute values and text node
				data = append(data, textNodeDomly)
			}
		}
	}

	return data
}

func GenDomly(domfrag []*html.Node) db.Domly {
	// The db.Domly format looks like: [tagName, attr1:val1, attr2:val2..., 'Text' | ChildNodes ]
	// what to do with TextNodes mixed with nested tagged content e.g. "sdfds ds <b>sdgtd </b>". Is there an explicit TextNode domly expression?

	data := db.Domly{}
	for _, node := range domfrag {
		if node.Type == html.ElementNode {
			data = append(data, GenDomlyNode(node))
		}
	}
	return data
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

	data := GenDomly(DOMTree)
	fmt.Printf("%s", data)

	json, err := GenJSON(data)
	if err != nil {
		return json, err
	}
	return json, nil
}
