// Parser for lisp like language

package parser

import (
	"fmt"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/ast"
	"github.com/cptaffe/lang/lexer"
	"log"
	"strconv"
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*parser) stateFn

// lexer holds the state of the scanner.
type parser struct {
	state      stateFn          // the next lexing function to enter
	items      chan token.Token // channel of scanned items
	buff []token.Token // buffer is an array of tokens
	pos int // pos is the location in buff
	tree       *ast.Tree            // tree position
	Root       *ast.Tree            // tree position
	parenDepth int              // nesting depth of ( ) exprs
}

func Parse(s string) *ast.Tree {
	ch := lexer.Lex(s)
	tree := new(ast.Tree)
	p := &parser{
		items: ch,
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

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (p *parser) errorf(tok token.Token) stateFn {
	fmt.Printf("Error: %s\n", tok.Val)
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
			return parseException
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
			return parseException
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
				node.Str = tok.Val
			case tok.Typ == token.ItemNumber:
				node.Typ = ast.ItemNum
				num, err := strconv.ParseFloat(tok.Val, 64)
				if err != nil {
					log.Fatal(err)
				}
				node.Num = num
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

func parseException(p *parser) stateFn {
	tok := p.next()
	switch {
	case tok.Typ == token.ItemEOF:
		return nil
	default:
		return p.errorf(tok)
	}
}
