package main

import (
	"./ast"
	"./interp"
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
	if tree == nil {
		os.Exit(1)
	}
	//fmt.Printf("%s\n", tree)
	num := interp.Exec(tree)
	fmt.Printf("result: ")
	for i := 0; i < len(num); i++ {
		if i != len(num)-1 {
			fmt.Printf("%d, ", num[i])
		} else {
			fmt.Printf("%d\n", num[i])
		}
	}
}
