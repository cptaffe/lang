// Parser for lisp like language

package parser

import (
	"../token"
	"fmt"
	"log"
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*parser) stateFn

// lexer holds the state of the scanner.
type parser struct {
	state      stateFn          // the next lexing function to enter
	items      chan token.Token // channel of scanned items
	done       chan *Tree       // signals Parse is done
	tree       *Tree            // tree position
	Root       *Tree            // tree position
	parenDepth int              // nesting depth of ( ) exprs
}

func Parse(ch chan token.Token, done chan *Tree) {
	tree := new(Tree)
	p := &parser{
		items: ch,
		done:  done,
		tree:  tree,
		Root:  tree,
	}
	go p.run()
	return
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (p *parser) errorf(tok token.Token) stateFn {
	fmt.Printf("Error: %s\n", tok.Val)
	return nil
}

// run runs the state machine for the lexer.
func (p *parser) run() {
	for p.state = parseAll; p.state != nil; {
		p.state = p.state(p)
	}
	close(p.items)
	p.done <- p.Root
}

// Handles EOF, Errors, sends list to parse inside list.
func parseAll(p *parser) stateFn {
	//print("Is parsing\n")
	p.tree = p.Root
	for {
		tok := <-p.items
		switch {
		case isException(tok):
			return handleException(tok, p)
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
		tok := <-p.items
		switch {
		// keyword at beginning of list
		// only at beginning because lexer has checked that.
		case isException(tok):
			return handleException(tok, p)
		case token.Keyword(tok.Typ):
			p.tree = p.tree.Append(&Node{
				Tok: tok,
			})
			return parseInsideAction
		case token.Constant(tok.Typ) || tok.Typ == token.ItemVariable:
			p.tree.Append(&Node{
				Tok: tok,
			})
		case tok.Typ == token.ItemEndList:
			p.parenDepth--
			if p.parenDepth == 0 {
				return parseAll
			}
		case tok.Typ == token.ItemBeginList:
			p.parenDepth++
		case isException(tok):
			return handleException(tok, p)
		}
	}
}

// An action is a list which does something.
// actions have the rest of the list as args.
func parseInsideAction(p *parser) stateFn {
	//print("Is parsing action\n")
	for {
		tok := <-p.items
		switch {
		case isException(tok):
			return handleException(tok, p)
		case token.Constant(tok.Typ) || tok.Typ == token.ItemVariable:
			p.tree.Append(&Node{
				Tok: tok,
			})
		case tok.Typ == token.ItemEndList:
			p.parenDepth--
			tree, err := p.Root.Walk(p.parenDepth)
			if err != nil {
				log.Fatal(err)
			}
			p.tree = tree
			if p.parenDepth == 0 {
				return parseAll
			} else {
				return parseInsideList
			}
		case tok.Typ == token.ItemBeginList:
			p.parenDepth++
			return parseInsideList
		}
	}
}

func isException(tok token.Token) bool {
	if tok.Typ == token.ItemEOF || tok.Typ == token.ItemError {
		return true
	}
	return false
}

func handleException(tok token.Token, p *parser) stateFn {
	//print("Is parsing exception\n")
	if tok.Typ == token.ItemEOF {
		return nil
	}
	return p.errorf(tok)
}
