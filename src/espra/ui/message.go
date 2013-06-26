// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package ui

import (
	"espra/db"
	"espra/lex"
	"strings"
	"unicode"
)

const (
	TextMarker = iota
)

func createMsgLexer(name, input string, startfn lex.StateFn) *lex.Lexer {
	// conf = map{}  -- add conf map to the lexer to allow lexer configuration
	l := &lex.Lexer{
		Name:  name,
		Input: input,
		Items: make(chan lex.Item),
		IntState: lex.IntState{
			Marker: 0,
		},
	}
	go l.Run(startfn)
	return l
}

// utilities
func IsWordChar(r) bool {
	if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '-' {
		return true
	}
	return false
}

func IsURIPathChar(r) bool {
	if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '-' || r == '/' || r == '&' || r == '?' || r == '=' || r == '.' || r == '' {
		return true
	}
	return false
}

//StateFns

func EspraURIMsg(l *lex.Lexer) lex.StateFn {
	for {
		if !IsURIPathChar(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemEspraURI)
	l.IntState[TextMarker] = l.Start
	return LexMsg
}

func URI(l *lex.Lexer) lex.StateFn {
	l.Pos = l.Start
	for {
		if !IsURIPathChar(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemURI)
	l.IntState[TextMarker] = l.Start
	return LexMsg
}

func HashTag(l *lex.Lexer) lex.StateFn {
	for {
		if !IsWordChar(l.Next()) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemHashTag)
	l.IntState[TextMarker] = l.Start
	return LexMsg
}

func Word(l *lex.Lexer) lex.StateFn {
	// finishing punctation
	for {
		if !IsWordChar(l.Next()) {
			if r == '\'' {
				if !unicode.IsLetter(l.Peek()) {
					break
				}
			} else if r == ':' {
				return URI
			} else {
				break
			}
		}
	}
	l.Emit(ItemWord)
	return LexMsg
}

func SlashTag(l *lex.Lexer) lex.StateFn {
	l.Next() //step over the slash
	l.Ignore()
	// first rune of slashtag must be a letter
	if !unicode.IsLetter(l.Next()) {
		return l.Errorf("Invalid: SlashTag should begin with a letter")
	}

	for {
		if !(IsValidIdentifierChar(l.Next())) {
			l.Backup()
			break
		}
	}
	l.Emit(ItemSlashTag)
	l.IntState[TextMarker] = l.Start
	return LexMsg
}

func EmitText(l *lex.Lexer) {
	l.Start = l.IntState[TextMarker]
	l.Emit(ItemText)
}

func LexMsg(l *lex.Lexer) lex.StateFn {
	first := l.Peek()
	if first == '/' {
		return SlashTag
	}
	switch r := l.Peek(); {
	case r == lex.EOF:
		EmitText(l)
		l.Next()
		l.Emit(lex.ItemEOF)
		return nil
	case r == '+':
		EmitText(l)
		l.Next()
		return EspraURIMsg
	case r == '#':
		EmitText(l)
		l.Next()
		return HashTag
	case lex.IsAlphaNumeric(r):
		return Word
	default:
		//advance and ignore all other characters at the beginning of a word
		l.Ignore()
	}
}

// hostname to lower
// # strip out fragment

// Main Parse routine
// domly <div>["sdfgsd", [a rel="i"]  ]</div>
// i for internal e for external
// to lower

// Returns (domlyAsJSON, references, slashTag, hostURIs)
//`` - code blocks quote uri's and slash tags.  words to lower
func parseMsg(message string, terms []string) ([]byte, []string, string, []*db.WebLink) {
	l := createMsgLexer("msgLex", message, LexMsg)
	slashTag := ""
	weblinks := []*db.WebLink{}
	var seen map[string]bool
	var item lex.Item
	var val string
	var prefix string
	domly := db.Domly{}

	for {
		item = <-l.Items
		switch item.Typ {
		case ItemWord:
			prefix = db.WordTerm
		case ItemEspraURI:
			prefix = db.EspraURITerm
		case ItemHashTag:
			prefix = db.HashTagTerm
		case ItemURI:
			prefix = db.URITerm
		case ItemSlashTag:
			prefix = db.SlashTagTerm
		}

		val = prefix + strings.ToLower(item.Val)
		if !seen[val] {
			seen[val] = true
		}
		terms = append(terms, item.Val)

	}
	return []byte("[]"), terms, slashTag, weblinks
}
