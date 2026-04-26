package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <filename>")
		return
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	l := NewLexer(string(content))

	for {
		tok := l.NextToken()
		if tok.Type == EOF_TOKEN {
			break
		}

		fmt.Printf("%s:%d:%d Type: %v, Literal: %q\n",
			filename,
			tok.Line,
			tok.Column,
			tok.Type,
			tok.Literal,
		)
	}
}
