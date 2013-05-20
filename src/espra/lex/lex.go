//Changes to this file by The Espra Authors are in the Public Domain.
//See the Espra UNLICENSE file for details.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ItemType int

const EOF = -1

const (
	ItemError ItemType = iota // error occurred; value is text of error
	ItemEOF
)

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// Item represents a token or text string returned from the scanner.
type Item struct {
	Typ ItemType // The type of this Item.
	Pos Pos      // The starting position, in bytes, of this Item in the input string.
	Val string   // The valVCue of this Item.
}

func (i Item) String() string {
	switch {
	case i.Typ == ItemEOF:
		return "EOF"
	case i.Typ == ItemError:
		return i.Val
	case len(i.Val) > 10:
		return fmt.Sprintf("%.10q...", i.Val)
	}
	return fmt.Sprintf("%q", i.Val)
}

type IntState [16]int

type StrState [16]string

// stateFn represents the state of the scanner as a function that returns the next state.
type StateFn func(*Lexer) StateFn

// Lexer holds the state of the scanner.
type Lexer struct {
	Name       string  // the name of the input; used only for error reports
	Input      string  // the string being scanned
	StateFn    StateFn // the next lexing function to enter
	StrState   StrState
	IntState   IntState
	Pos        Pos       // current position in the input
	Start      Pos       // start position of this item
	Width      Pos       // width of last rune read from input
	LastPos    Pos       // position of most recent item returned by nextItem
	Items      chan Item // channel of scanned Items
	ParenDepth int       // nesting depth of ( ) exprs
}

// next returns the next rune in the input.
func (l *Lexer) Next() rune {
	if int(l.Pos) >= len(l.Input) {
		l.Width = 0
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.Input[l.Pos:])
	l.Width = Pos(w)
	l.Pos += l.Width
	return r
}

// Peek returns but does not consume the next rune in the input.
func (l *Lexer) Peek() rune {
	r := l.Next()
	l.Backup()
	return r
}

// Backup steps back one rune. Can only be called once per call of next.
func (l *Lexer) Backup() {
	l.Pos -= l.Width
}

// emit passes an Item back to the client.
func (l *Lexer) Emit(t ItemType) {
	l.Items <- Item{t, l.Start, l.Input[l.Start:l.Pos]}
	l.Start = l.Pos
}

// ignore skips over the pending input before this point.
func (l *Lexer) Ignore() {
	l.Start = l.Pos
}

// accept consumes the next rune if it's from the valid set.
func (l *Lexer) Accept(valid string) bool {
	if strings.IndexRune(valid, l.Next()) >= 0 {
		return true
	}
	l.Backup()
	return false
}

// AcceptRun consumes a run of runes from the valid set.
func (l *Lexer) AcceptRun(valid string) {
	for strings.IndexRune(valid, l.Next()) >= 0 {
	}
	l.Backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous Item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *Lexer) LineNumber() int {
	return 1 + strings.Count(l.Input[:l.LastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *Lexer) Errorf(format string, args ...interface{}) StateFn {
	l.Items <- Item{ItemError, l.Start, fmt.Sprintf(format, args...)}
	return nil
}

// nextItem returns the next Item from the input.
func (l *Lexer) NextItem() Item {
	item := <-l.Items
	l.LastPos = item.Pos
	return item
}

func (l *Lexer) ScanNumber() bool {
	// Optional leading sign.
	l.Accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.Accept("0") && l.Accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.AcceptRun(digits)
	if l.Accept(".") {
		l.AcceptRun(digits)
	}
	if l.Accept("eE") {
		l.Accept("+-")
		l.AcceptRun("0123456789")
	}
	// Is it imaginary?
	l.Accept("i")
	// Next thing mustn't be alphanumeric.
	if IsAlphaNumeric(l.Peek()) {
		l.Next()
		return false
	}
	return true
}

// isSpace reports whether r is a space character.
func IsSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func IsEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func IsAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// run runs the state machine for the Lexer.
func (l *Lexer) Run(startfn StateFn) {
	for l.StateFn = startfn; l.StateFn != nil; {
		l.StateFn = l.StateFn(l)
	}
}
