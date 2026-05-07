package main

import (
	"fmt"
	"strings"
)

func ReportError(filepath string, source string, line int, column int, literal string, message string) {
	lines := strings.Split(source, "\n")
	fmt.Printf("%s:%d:%d: error: %s\n", filepath, line, column, message)

	// Context window (1 line before, 1 line after)
	start := max(line-2, 0)
	end := min(line+1, len(lines))

	for i := start; i < end; i++ {
		lNum := i + 1
		lContent := lines[i]

		gutter := fmt.Sprintf("%4d | ", lNum)
		fmt.Printf("%s%s\n", gutter, lContent)

		if lNum == line {
			padding := strings.Repeat(" ", len(gutter))
			for j := 1; j < column; j++ {
				if j <= len(lContent) && lContent[j-1] == '\t' {
					padding += "\t"
				} else {
					padding += " "
				}
			}

			length := len(literal)
			if length == 0 {
				length = 1
			}

			fmt.Printf("%s%s\n", padding, strings.Repeat("^", length))
		}
	}
}

type CompilerError struct {
	Line    int
	Column  int
	Literal string
	Message string
}

func (e *CompilerError) Error() string {
	return fmt.Sprintf("%d:%d: error: %s", e.Line, e.Column, e.Message)
}
