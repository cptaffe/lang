// Parser for lisp like language

package parser

import (
	"fmt"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/ast"
	"github.com/cptaffe/lang/lexer"
	"log"
	"strconv"
	"strings"
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*parser) stateFn

// lexer holds the state of the scanner.
type parser struct {
	name string // file name
	state      stateFn          // the next lexing function to enter
	input string
	items      chan token.Token // channel of scanned items
	buff []token.Token // buffer is an array of tokens
	pos int // pos is the location in buff
	tree       *ast.Tree            // tree position
	Root       *ast.Tree            // tree position
	parenDepth int              // nesting depth of ( ) exprs
}

func Parse(s string, name string) *ast.Tree {
	l := lexer.Lex(s, name)
	tree := new(ast.Tree)
	p := &parser{
		name: l.Name,
		input: s,
		items: l.Items,
		tree:  tree,
		Root:  tree,
	}
	return p.run()
}

// next
func (p *parser) next() (token.Token) {
	if len(p.buff) -1 <= p.pos {
		p.buff = append(p.buff, <-p.items)
	}
	tok := p.buff[p.pos]
	p.pos++
	return tok
}

// backup
func (p *parser) backup() {
	p.pos -= 1
}

// peekBack
func (p *parser) peekBack() token.Token {
	p.backup()
	return p.next()
}

// get line number
func (p *parser) lineNumber(tok token.Token) int {
	return 1 + strings.Count(p.input[:tok.Pos], "\n")
}

// get character number
func (p *parser) charNumber(tok token.Token) int {
	return int(tok.Pos) - strings.LastIndex(p.input[:tok.Pos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (p *parser) errorf(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	tok := p.peekBack()
	// print error message
	fmt.Printf("\033[1m%s: %s:%d:%d: \033[31merror:\033[0m\033[1m %s\033[0m\n", "parse", p.name, p.lineNumber(tok), p.charNumber(tok), msg)
	return nil
}

// run runs the state machine for the lexer.
func (p *parser) run() *ast.Tree {
	for p.state = parseAll; p.state != nil; {
		p.state = p.state(p)
	}
	return p.Root
}

// Handles EOF, Errors, sends list to parse inside list.
func parseAll(p *parser) stateFn {
	for {
		tok := p.next()
		switch {
		case isException(tok):
			p.backup()
			return nil
		case tok.Typ == token.ItemBeginList:
			p.parenDepth++
			return parseInsideList
		}
	}
}

// Inside a list
// everything happens here.
func parseInsideList(p *parser) stateFn {
	//print("Is parsing list\n")
	for {
		tok := p.next()
		switch {
		// keyword at beginning of list
		// only at beginning because lexer has checked that.
		case isException(tok):
			p.backup()
			return nil
		// Cases with subs 
		case token.Keyword(tok.Typ):
			p.tree = p.tree.Append(&ast.Node{
				Typ: ast.ItemKey,
				Key: tok.Typ,
				Var: tok.Val,
			})
			return parseInsideList
		case token.Constant(tok.Typ) || tok.Typ == token.ItemVariable:
			var node = new(ast.Node)
			switch{
			case tok.Typ == token.ItemVariable:
				node.Typ = ast.ItemVar
				node.Var = tok.Val
			case tok.Typ == token.ItemString:
				node.Typ = ast.ItemString
				node.Str = tok.Val[1:len(tok.Val)-1]
			case tok.Typ == token.ItemNumber:
				node.Typ = ast.ItemNum
				num, err := strconv.ParseInt(tok.Val, 10, 32)
				if err != nil {
					log.Fatal(err)
				}
				node.Num = int32(num)
			case tok.Typ == token.ItemBool:
				node.Typ = ast.ItemNum
				if tok.Val == "true" {
					node.Num = 1
				} else {
					node.Num = 0
				}
			}
			p.tree.Append(node)
		case tok.Typ == token.ItemEndList:
			p.parenDepth--
			tree, err := p.Root.Walk(p.parenDepth)
			if err != nil {
				log.Fatal(err)
			}
			p.tree = tree
			if p.parenDepth == 0 {
				return parseAll
			}
		case tok.Typ == token.ItemBeginList:
			p.parenDepth++
		}
	}
}

func isException(tok token.Token) bool {
	if tok.Typ == token.ItemEOF || tok.Typ == token.ItemError {
		return true
	}
	return false
}
