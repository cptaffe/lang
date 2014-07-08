package main

import (
	"./ast"
	"./lexer"
	"./parser"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan *ast.Tree)
	ch := lexer.Lex(os.Args[1], string(b))
	parser.Parse(ch, done)
	tree := <-done
	fmt.Printf("%s", tree)
}
