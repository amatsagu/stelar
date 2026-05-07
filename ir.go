package main

import (
	"fmt"
)

type Operation struct {
	Type     TokenType
	Value    string
	JumpTo   int // Index of the target operation in the IR array
	Line     int
	Column   int
	IsGreedy bool // When operator consumes arguments until ';' (stop mark)
}

type Parser struct {
	lexer *Lexer
	ir    []Operation
}

func NewParser(lexer *Lexer) *Parser {
	return &Parser{
		lexer: lexer,
		ir:    make([]Operation, 0),
	}
}

func (p *Parser) Parse() ([]Operation, []CompilerError) {
	stack := []int{}
	errors := []CompilerError{}

	for {
		tok := p.lexer.NextToken()
		if tok.Type == EOF_TOKEN {
			break
		}

		if tok.Type == ILLEGAL_TOKEN {
			errors = append(errors, CompilerError{tok.Line, tok.Column, tok.Literal, fmt.Sprintf("unrecognized token: '%s'", tok.Literal)})
			continue
		}

		op := Operation{
			Type:   tok.Type,
			Value:  tok.Literal,
			Line:   tok.Line,
			Column: tok.Column,
			JumpTo: 0,
		}

		switch tok.Type {
		case PUT_TOKEN, PRINT_TOKEN, BIND_TOKEN, REQUIRE_TOKEN:
			op.IsGreedy = true
		case IF_TOKEN, FOR_TOKEN, SWITCH_TOKEN, PROC_TOKEN, UNSAFE_PROC_TOKEN, WITH_TOKEN:
			stack = append(stack, len(p.ir))
		case ELSE_TOKEN:
			if len(stack) == 0 {
				errors = append(errors, CompilerError{tok.Line, tok.Column, tok.Literal, "unexpected 'else' without a preceding 'if' operation"})
				break // Using break allows the token to still be appended to IR, keeping IR and source somewhat synced for later error recovery
			}

			openingIdx := stack[len(stack)-1]
			if p.ir[openingIdx].Type != IF_TOKEN {
				errors = append(errors, CompilerError{tok.Line, tok.Column, tok.Literal, fmt.Sprintf("unexpected 'else' inside '%s' block (can only be used after 'if' operation)", p.ir[openingIdx].Value)})
				break
			}

			// IF condition fails -> Jump over the ELSE instruction into the else body
			p.ir[openingIdx].JumpTo = len(p.ir) + 1

			// Replace IF on the stack with ELSE, so END can resolve the ELSE jump
			stack[len(stack)-1] = len(p.ir)
		case END_TOKEN:
			if len(stack) == 0 {
				errors = append(errors, CompilerError{tok.Line, tok.Column, tok.Literal, "unexpected 'end' operation (no open block to close)"})
				break
			}

			openingIdx := stack[len(stack)-1]
			stack = stack[:len(stack)-1] // Pop from stack
			openingOp := &p.ir[openingIdx]

			switch openingOp.Type {
			case IF_TOKEN, ELSE_TOKEN, SWITCH_TOKEN, PROC_TOKEN, UNSAFE_PROC_TOKEN, WITH_TOKEN:
				openingOp.JumpTo = len(p.ir)
			case FOR_TOKEN:
				openingOp.JumpTo = len(p.ir) + 1
				op.JumpTo = openingIdx
			default:
				errors = append(errors, CompilerError{tok.Line, tok.Column, tok.Literal, "compiler bug - unknown control block on stack"})
			}
		}

		p.ir = append(p.ir, op)
	}

	for _, idx := range stack {
		unclosed := p.ir[idx]
		errors = append(errors, CompilerError{unclosed.Line, unclosed.Column, unclosed.Value, fmt.Sprintf("unclosed block: '%s' has no matching 'end' operation", unclosed.Value)})
	}

	return p.ir, errors
}

func DebugPrintIR(ir []Operation) {
	fmt.Printf("%-4s | %-17s | %-15s | %-6s\n", "ADDR", "TYPE", "VALUE", "JUMP")
	fmt.Println("---------------------------------------------------------------")
	for i, op := range ir {
		jumpStr := ""
		if op.JumpTo != 0 {
			jumpStr = fmt.Sprintf("-> %04d", op.JumpTo)
		}
		fmt.Printf("%04d | %-17v | %-15s | %s\n", i, op.Type, op.Value, jumpStr)
	}
}
