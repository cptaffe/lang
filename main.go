package main

import (
	"./lexer"
	"./optim"
	"./parser"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

func Compute(s string) string {
	ch := lexer.Lex(s)
	done := make(chan *parser.Tree)
	parser.Parse(ch, done)
	tree := <-done
	//fmt.Printf("%s\n", tree)
	t := optim.Eval(tree)
	if t == nil {
		return "error..."
	}
	return fmt.Sprintf("result: %s", t)
}

// Read input from stdin & output result to stdout
func main() {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(": ")
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Print("exit\n")
				return
			}
			log.Print(err)
		}
		if string(b) == "exit" {
			os.Exit(0)
		}
		ans := Compute(string(b))
		fmt.Println(ans)
	}
}
