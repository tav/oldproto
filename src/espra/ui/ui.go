// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package ui

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"encoding/json"
	"espra/db"
	"fmt"
	"os"
	"text/template/parse"
)

var bodyNode = &html.Node{Data: "body", DataAtom: atom.Body, Type: html.ElementNode}

//var replacements = map[string]string

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
	"for":             "htmlFor"}

func ParseHTML5(filename string) ([]*html.Node, error) {
	reader, err := os.Open(filename)
	frag, err := html.ParseFragment(reader, bodyNode)
	return frag, err
}

func GenDomlyNode(domNode *html.Node) db.Domly {
	data := db.Domly{}
	if domNode.Type == html.ElementNode {
		data = append(data, domNode.Data)
		if len(domNode.Attr) != 0 { // != nil {
			attrs := db.DomlyAttrs{}
			for _, attr := range domNode.Attr {
				attrs[attr.Key] = attr.Val
			}
			data = append(data, attrs)
		}
		for c := domNode.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				data = append(data, GenDomlyNode(c))
			} else if c.Type == html.TextNode {
				_ = parse.Parse // for each token parse attribute values and text nodes
				data = append(data, c.Data)
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
