package main

import (
	"./ast"
	"./interp"
	"./lexer"
	"./parser"
	"bufio"
	"fmt"
	"log"
	"os"
)

func Compute(s string) string {
	ch := lexer.Lex(s)
	done := make(chan *ast.Tree)
	parser.Parse(ch, done)
	tree := <-done
	//fmt.Printf("%s\n", tree)
	num := interp.Exec(tree)
	if num == nil {
		return "error"
	}
	str := "result: "
	for i := 0; i < len(num); i++ {
		if i != len(num)-1 {
			str += fmt.Sprintf("%d, ", num[i])
		} else {
			str += fmt.Sprintf("%d", num[i])
		}
	}
	return str
}

// Read input from stdin & output result to stdout
func main() {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(": ")
		b, _, err := r.ReadLine()
		if err != nil {
			log.Print(err)
		}
		//fmt.Printf("%s\n", string(b))
		if string(b) == "exit" {
			os.Exit(0)
		}
		ans := Compute(string(b))
		fmt.Println(ans)
	}
}
