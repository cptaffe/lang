// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Lexer for a lisp like language

package lexer

import (
	"../token"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string           // the name of the input; used only for error reports
	input      string           // the string being scanned
	state      stateFn          // the next lexing function to enter
	pos        token.Pos        // current position in the input
	start      token.Pos        // start position of this item
	width      token.Pos        // width of last rune read from input
	lastPos    token.Pos        // position of most recent item returned by nextItem
	items      chan token.Token // channel of scanned items
	parenDepth int              // nesting depth of ( ) exprs
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return token.Eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = token.Pos(w)
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t token.ItemType) {
	l.items <- token.Token{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- token.Token{token.ItemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

// lex creates a new scanner for the input string.
func Lex(input string) chan token.Token {
	l := &lexer{
		input: input,
		items: make(chan token.Token),
	}
	go l.run()
	return l.items
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexAll; l.state != nil; {
		l.state = l.state(l)
	}
}

// state functions
const (
	// call deliminators
	leftList  = '('
	rightList = ')'
	// comment deliminators
	leftComment  = "/*"
	rightComment = "*/"
)

// lexAll scans until it runs into a list
func lexAll(l *lexer) stateFn {
	for {
		r := l.next()
		if r == token.Eof {
			break
		} else if isSpace(r) || r == '\n' {
			// consume
		} else if r == leftList {
			l.backup()
			if l.start < l.pos {
				l.emit(token.ItemSpace)
			}
			return lexList
		} else {
			// r is not a list
			return l.errorf("unexpected nonlist item: %#U", r)
		}
	}
	// Correctly reached EOF.
	l.emit(token.ItemEOF)
	return nil
}

// lexList scans deliminators for a list
func lexList(l *lexer) stateFn {
	r := l.next()
	if r == leftList {
		l.parenDepth++
		l.emit(token.ItemBeginList)
		return checkKeyword
	} else if r == rightList {
		l.parenDepth--
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren: %#U", r)
		} else {
			l.emit(token.ItemEndList)
		}
		if l.parenDepth == 0 {
			return lexAll // could be anywhere
		} else {
			return lexInsideList // still inside a list
		}
	}
	// r is not a list
	return l.errorf("unexpected nonlist item: %#U", r)
}

// lexInsideList scans the elements inside list delimiters.
func lexInsideList(l *lexer) stateFn {
	// Either number, quoted string, or Variable.
	// Spaces separate arguments; runs of spaces turn into itemSpace.
	r := l.next()
	switch {
	case r == token.Eof:
		return l.errorf("unclosed list")
	case isSpace(r):
		return lexSpace
	case isEndOfLine(r):
		return lexEndOfLine
	case r == leftList || r == rightList:
		l.backup()
		return lexList
	case r == '/':
		return lexComment
	case r == '"':
		return lexQuote
	case r == '`':
		return lexRawQuote
	case r == '\'':
		return lexChar
	case r == '-' || r == '+' || ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexVariable
	default:
		return l.errorf("unrecognized character in list: %#U", r)
	}
	return lexInsideList
}

func checkKeyword(l *lexer) stateFn {
	// list can have operation as first element
	s := l.readWord() // reads word
	if token.IsKeyword(s) {
		return lexKeyword
	} else {
		if l.start < l.pos {
			l.pos = l.start
		}
		return lexInsideList
	}
}

// lexOperation scans operations
func lexKeyword(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case !isSpace(r):
			// consume
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			switch {
			case token.IsKeyword(word):
				l.emit(token.Lookup(word))
				return lexInsideList
			default:
				return l.errorf("unexpected inoperative list: %#U", r)
			}
		}
	}
}

// lexSpace scans a run of space characters.
// One space has already been seen.
func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(token.ItemSpace)
	return lexInsideList
}

// lexVariable scans an alphanumeric.
func lexVariable(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			switch {
			case word == "true", word == "false":
				l.emit(token.ItemBool)
			default:
				l.emit(token.ItemVariable)
			}
			break Loop
		}
	}
	return lexInsideList
}

// lexChar scans a character constant. The initial quote is already
// scanned. Syntax checking is done by the parser.
func lexChar(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != token.Eof && r != '\n' {
				break
			}
			fallthrough
		case token.Eof, '\n':
			return l.errorf("unterminated character constant")
		case '\'':
			break Loop
		}
	}
	l.emit(token.ItemChar)
	return lexInsideList
}

// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This
// isn't a perfect number scanner - for instance it accepts "." and "0x0.2"
// and "089" - but when it's wrong the input is invalid and the parser (via
// strconv) will notice.
func lexNumber(l *lexer) stateFn {
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	if sign := l.peek(); sign == '+' || sign == '-' {
		// Complex: 1+2i. No spaces, must end in 'i'.
		if !l.scanNumber() || l.input[l.pos-1] != 'i' {
			return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
		}
		l.emit(token.ItemComplex)
	} else {
		l.emit(token.ItemNumber)
	}
	return lexInsideList
}

func (l *lexer) scanNumber() bool {
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Is it imaginary?
	l.accept("i")
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return false
	}
	return true
}

// lexQuote scans a quoted string.
func lexQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != token.Eof && r != '\n' {
				break
			}
			fallthrough
		case token.Eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}
	l.emit(token.ItemString)
	return lexInsideList
}

// lexRawQuote scans a raw quoted string.
func lexRawQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case token.Eof, '\n':
			return l.errorf("unterminated raw quoted string")
		case '`':
			break Loop
		}
	}
	l.emit(token.ItemRawString)
	return lexInsideList
}

// lexEndOfLine is called when on a newline
func lexEndOfLine(l *lexer) stateFn {
	l.emit(token.ItemNewline)
	return lexInsideList
}

// lexComment lexes a comment, it is on the first character of one.
func lexComment(l *lexer) stateFn {
	r := l.next()
	if r == '/' {
		for {
			r := l.next()
			switch {
			// consume until newline
			case isEndOfLine(r) || r == token.Eof:
				l.emit(token.ItemLineComment)
				return lexInsideList
			}
		}
	} else if r == '*' {
		for {
			r := l.next()
			switch {
			// consume until newline
			case r == '*':
				if l.next() == '/' {
					l.emit(token.ItemLineComment)
					return lexInsideList
				} else {
					return l.errorf("unterminated comment")
				}
			case r == token.Eof:
				return l.errorf("unterminated comment")
			}
		}
	} else {
		return l.errorf("unexpected noncomment in list: %#U", r)
	}
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// lexOperation scans operations
func (l *lexer) readWord() string {
	for {
		switch r := l.next(); {
		case !isSpace(r):
			// consume
		default:
			l.backup()
			return l.input[l.start:l.pos]
		}
	}
}
