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
	t := optim.Eval(tree)
	if t == nil {
		return "error..."
	}
	return fmt.Sprintf("result: %s", t.Sub[len(t.Sub)-1])
}

// Read input from stdin & output result to stdout
func main() {
	r := bufio.NewReader(os.Stdin)
	var str string
	keep := true
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
		if keep {
			str += string(b)
		} else {
			str = string(b)
		}
		if string(b) == "exit" {
			os.Exit(0)
		}
		ans := Compute(str)
		fmt.Println(ans)
	}
}
