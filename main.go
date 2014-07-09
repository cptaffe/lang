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

type Program struct {
	Str string // string of input
	Len int    // length of tree last time
}

func Compute(s *Program) string {
	ch := lexer.Lex(s.Str)
	done := make(chan *parser.Tree)
	parser.Parse(ch, done)
	tree := <-done
	//fmt.Printf("%s\n", tree)
	t := optim.Eval(tree)
	if t == nil {
		return "error..."
	}
	str := "result: "
	for i := 0; i < (len(t.Sub) - s.Len); i++ {
		str += fmt.Sprintf("%s", t.Sub[s.Len+i])
	}
	s.Len = len(t.Sub) - 1 // set new len
	return str
}

// Read input from stdin & output result to stdout
func main() {
	r := bufio.NewReader(os.Stdin)
	p := new(Program)
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
		p.Str = string(b)
		ans := Compute(p)
		fmt.Println(ans)
	}
}
