package main

import (
	"fmt"
)

type ShadowStack struct {
	Types []TokenType
}

func (s *ShadowStack) Push(t TokenType) {
	s.Types = append(s.Types, t)
}

func (s *ShadowStack) Pop() (TokenType, bool) {
	if len(s.Types) == 0 {
		return ILLEGAL_TOKEN, false
	}
	t := s.Types[len(s.Types)-1]
	s.Types = s.Types[:len(s.Types)-1]
	return t, true
}

func (s *ShadowStack) Peek(index int) (TokenType, bool) {
	if index < 0 || index >= len(s.Types) {
		return ILLEGAL_TOKEN, false
	}
	return s.Types[len(s.Types)-1-index], true
}

func (s *ShadowStack) Clone() *ShadowStack {
	newTypes := make([]TokenType, len(s.Types))
	copy(newTypes, s.Types)
	return &ShadowStack{Types: newTypes}
}

type BindInfo struct {
	StackName string
	PeekIndex int
	Type      TokenType
}

type Analyzer struct {
	filepath string
	source   string
	ir       []Operation
	pos      int
	stacks   map[string]*ShadowStack
	defaultS []string
	binds    map[string]BindInfo
	errors   []CompilerError
	inUnsafe bool
}

func NewAnalyzer(filepath string, source string, ir []Operation) *Analyzer {
	return &Analyzer{
		filepath: filepath,
		source:   source,
		ir:       ir,
		stacks: map[string]*ShadowStack{
			"main":   {Types: []TokenType{}},
			"errors": {Types: []TokenType{}},
		},
		defaultS: []string{"main"},
		binds:    make(map[string]BindInfo),
		inUnsafe: false,
	}
}

func (a *Analyzer) currentDefault() string {
	return a.defaultS[len(a.defaultS)-1]
}

func (a *Analyzer) Analyze() []CompilerError {
	for a.pos < len(a.ir) {
		op := a.ir[a.pos]
		err := a.analyzeOp(op)
		if err != nil {
			a.errors = append(a.errors, *err)
		}
		a.pos++
	}
	return a.errors
}

func (a *Analyzer) checkPushAccess(targetStack string, op Operation) *CompilerError {
	if targetStack == "errors" && !a.inUnsafe {
		return &CompilerError{op.Line, op.Column, op.Value, "restricted access: cannot add values to the 'errors' stack outside of an unsafe procedure (~proc)"}
	}
	return nil
}

func (a *Analyzer) analyzeOp(op Operation) *CompilerError {
	targetStack := a.currentDefault()

	if op.Type == IDENT_TOKEN && a.pos+1 < len(a.ir) {
		next := a.ir[a.pos+1]
		switch next.Type {
		case DOT_TOKEN:
			targetStack = op.Value
			a.pos += 2
			if a.pos >= len(a.ir) {
				return &CompilerError{op.Line, op.Column, op.Value, "trailing stack prefix"}
			}
			op = a.ir[a.pos]
		case PEEK_TOKEN:
			targetStack = op.Value
			a.pos += 1
			op = a.ir[a.pos]
		}
	}

	if _, ok := a.stacks[targetStack]; !ok {
		a.stacks[targetStack] = &ShadowStack{Types: []TokenType{}}
	}

	switch op.Type {
	case INTEGER_TOKEN, FLOAT_TOKEN, STRING_TOKEN:
		if err := a.checkPushAccess(targetStack, op); err != nil {
			return err
		}
		a.stacks[targetStack].Push(op.Type)

	case PUT_TOKEN, PRINT_TOKEN:
		if op.Type == PUT_TOKEN {
			if err := a.checkPushAccess(targetStack, op); err != nil {
				return err
			}
		}
		return a.handleGreedy(targetStack, op)

	case PLUS_TOKEN, MINUS_TOKEN, ASTERISK_TOKEN, SLASH_TOKEN:
		if err := a.checkPushAccess(targetStack, op); err != nil {
			return err
		}
		return a.handleMathGreedy(targetStack, op)

	case GT_TOKEN, LT_TOKEN, EQ_TOKEN:
		return a.handleMathGreedy(targetStack, op)

	case IF_TOKEN, ELSE_TOKEN, FOR_TOKEN, SWITCH_TOKEN, CASE_TOKEN, DEFAULT_TOKEN:
		return nil

	case PROC_TOKEN:
		a.inUnsafe = false
		return nil
	case UNSAFE_PROC_TOKEN:
		a.inUnsafe = true
		return nil

	case POP_TOKEN:
		_, ok := a.stacks[targetStack].Pop()
		if !ok {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("stack underflow on '%s'", targetStack)}
		}

	case CLONE_TOKEN:
		if err := a.checkPushAccess(targetStack, op); err != nil {
			return err
		}
		t, ok := a.stacks[targetStack].Peek(0)
		if !ok {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("cannot clone from empty stack '%s'", targetStack)}
		}
		a.stacks[targetStack].Push(t)

	case WITH_TOKEN:
		a.pos++
		if a.pos >= len(a.ir) || a.ir[a.pos].Type != IDENT_TOKEN {
			return &CompilerError{op.Line, op.Column, op.Value, "'with' requires stack identifier"}
		}
		a.defaultS = append(a.defaultS, a.ir[a.pos].Value)

	case END_TOKEN:
		for i := 0; i < a.pos; i++ {
			if a.ir[i].JumpTo == a.pos {
				if a.ir[i].Type == WITH_TOKEN {
					a.defaultS = a.defaultS[:len(a.defaultS)-1]
					break
				}
				if a.ir[i].Type == PROC_TOKEN || a.ir[i].Type == UNSAFE_PROC_TOKEN {
					a.inUnsafe = false
					break
				}
			}
		}

	case PEEK_TOKEN:
		index := 0
		fmt.Sscanf(op.Value, ".%d", &index)
		_, ok := a.stacks[targetStack].Peek(index)
		if !ok {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("peek out of bounds on '%s' (size: %d)", targetStack, len(a.stacks[targetStack].Types))}
		}

	case BIND_TOKEN:
		return a.handleBind(op)

	case REQUIRE_TOKEN:
		return a.handleRequire(targetStack, op)

	case IDENT_TOKEN:
		if bind, ok := a.binds[op.Value]; ok {
			_, ok := a.stacks[bind.StackName].Peek(bind.PeekIndex)
			if !ok {
				return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("accessing bind '%s' failed: peek out of bounds on '%s'", op.Value, bind.StackName)}
			}
		}
	}

	return nil
}

func (a *Analyzer) resolveExpression(targetStack string) (TokenType, *CompilerError) {
	if a.pos+1 >= len(a.ir) {
		return ILLEGAL_TOKEN, &CompilerError{0, 0, "", "unexpected end of IR"}
	}

	next := a.ir[a.pos+1]
	switch next.Type {
	case IF_TOKEN, ELSE_TOKEN, FOR_TOKEN, SWITCH_TOKEN, CASE_TOKEN, DEFAULT_TOKEN, END_TOKEN, PROC_TOKEN, UNSAFE_PROC_TOKEN, WITH_TOKEN, SEMICOLON_TOKEN:
		return ILLEGAL_TOKEN, &CompilerError{next.Line, next.Column, next.Value, fmt.Sprintf("unexpected %s where expression was expected", next.Value)}
	}

	a.pos++
	exprOp := a.ir[a.pos]

	exprTarget := targetStack
	if exprOp.Type == IDENT_TOKEN && a.pos+1 < len(a.ir) {
		nn := a.ir[a.pos+1]
		switch nn.Type {
		case DOT_TOKEN:
			exprTarget = exprOp.Value
			a.pos += 2
			if a.pos >= len(a.ir) {
				return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, "trailing stack prefix"}
			}
			exprOp = a.ir[a.pos]
		case PEEK_TOKEN:
			exprTarget = exprOp.Value
			a.pos += 1
			exprOp = a.ir[a.pos]
		}
	}

	switch exprOp.Type {
	case INTEGER_TOKEN, FLOAT_TOKEN, STRING_TOKEN:
		return exprOp.Type, nil
	case POP_TOKEN, UNDERSCORE_TOKEN:
		t, ok := a.stacks[exprTarget].Pop()
		if !ok {
			return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("stack underflow on '%s'", exprTarget)}
		}
		return t, nil
	case SIZE_TOKEN:
		return INTEGER_TOKEN, nil
	case PEEK_TOKEN:
		index := 0
		fmt.Sscanf(exprOp.Value, ".%d", &index)
		t, ok := a.stacks[exprTarget].Peek(index)
		if !ok {
			return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("peek out of bounds on '%s'", exprTarget)}
		}
		return t, nil
	case IDENT_TOKEN:
		if bind, ok := a.binds[exprOp.Value]; ok {
			if bind.PeekIndex == -1 {
				return bind.Type, nil
			}
			t, ok := a.stacks[bind.StackName].Peek(bind.PeekIndex)
			if !ok {
				return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("bind '%s' out of bounds", exprOp.Value)}
			}
			return t, nil
		}
	case PLUS_TOKEN, MINUS_TOKEN, ASTERISK_TOKEN, SLASH_TOKEN, GT_TOKEN, LT_TOKEN, EQ_TOKEN:
		t1, err := a.resolveExpression(exprTarget)
		if err != nil {
			return ILLEGAL_TOKEN, err
		}
		if t1 == STRING_TOKEN {
			return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("type error: math operator '%s' does not support string arguments", exprOp.Value)}
		}
		t2, err := a.resolveExpression(exprTarget)
		if err != nil {
			return ILLEGAL_TOKEN, err
		}
		if t2 == STRING_TOKEN {
			return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("type error: math operator '%s' does not support string arguments", exprOp.Value)}
		}

		if isComparisonOp(exprOp.Type) {
			return INTEGER_TOKEN, nil
		}
		if t1 == FLOAT_TOKEN || t2 == FLOAT_TOKEN {
			return FLOAT_TOKEN, nil
		}
		return INTEGER_TOKEN, nil
	}

	return ILLEGAL_TOKEN, &CompilerError{exprOp.Line, exprOp.Column, exprOp.Value, fmt.Sprintf("cannot resolve expression type for %v", exprOp.Type)}
}

func isMathOp(t TokenType) bool {
	return t == PLUS_TOKEN || t == MINUS_TOKEN || t == ASTERISK_TOKEN || t == SLASH_TOKEN || t == GT_TOKEN || t == LT_TOKEN || t == EQ_TOKEN
}

func isComparisonOp(t TokenType) bool {
	return t == GT_TOKEN || t == LT_TOKEN || t == EQ_TOKEN
}

func (a *Analyzer) skipGreedyBlock(pos int) int {
	for i := pos + 1; i < len(a.ir); i++ {
		t := a.ir[i].Type
		if t == PUT_TOKEN || t == PRINT_TOKEN || t == BIND_TOKEN || t == REQUIRE_TOKEN {
			i = a.skipGreedyBlock(i)
			if i == -1 {
				return -1
			}
		} else if isMathOp(t) {
			if a.isMathGreedy(i) {
				i = a.skipGreedyBlock(i)
				if i == -1 {
					return -1
				}
			}
		} else if t == SEMICOLON_TOKEN {
			return i
		}
	}
	return -1
}

func (a *Analyzer) isMathGreedy(pos int) bool {
	if pos+1 >= len(a.ir) {
		return false
	}
	if a.ir[pos+1].Type == SEMICOLON_TOKEN {
		return true
	}

	for i := pos + 1; i < len(a.ir); i++ {
		t := a.ir[i].Type
		if t == PUT_TOKEN || t == PRINT_TOKEN || t == BIND_TOKEN || t == REQUIRE_TOKEN {
			i = a.skipGreedyBlock(i)
			if i == -1 {
				return false
			}
		} else if isMathOp(t) {
			if a.isMathGreedy(i) {
				i = a.skipGreedyBlock(i)
				if i == -1 {
					return false
				}
			}
		} else if t == SEMICOLON_TOKEN {
			return true
		} else if t == END_TOKEN || t == PROC_TOKEN || t == IF_TOKEN || t == FOR_TOKEN {
			return false
		}
	}
	return false
}

func (a *Analyzer) handleMathGreedy(targetStack string, op Operation) *CompilerError {
	if !a.isMathGreedy(a.pos) {
		t1, ok1 := a.stacks[targetStack].Pop()
		t2, ok2 := a.stacks[targetStack].Pop()
		if !ok1 || !ok2 {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("legacy operation '%s' requires 2 elements on '%s'", op.Value, targetStack)}
		}

		if t1 == STRING_TOKEN || t2 == STRING_TOKEN {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("type error: math operator '%s' does not support string arguments", op.Value)}
		}

		resType := INTEGER_TOKEN
		if !isComparisonOp(op.Type) && (t1 == FLOAT_TOKEN || t2 == FLOAT_TOKEN) {
			resType = FLOAT_TOKEN
		}
		a.stacks[targetStack].Push(resType)
		return nil
	}

	if a.pos+1 < len(a.ir) && a.ir[a.pos+1].Type == SEMICOLON_TOKEN {
		return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("operator '%s' with 0 arguments (greedy mode requires at least 1, or use legacy mode by removing ';')", op.Value)}
	}

	argCount := 0
	var finalType TokenType = INTEGER_TOKEN

	for {
		if a.pos+1 >= len(a.ir) {
			return &CompilerError{op.Line, op.Column, op.Value, "operator missing ';' terminator"}
		}
		if a.ir[a.pos+1].Type == SEMICOLON_TOKEN {
			a.pos++
			break
		}
		typ, err := a.resolveExpression(targetStack)
		if err != nil {
			return err
		}

		if typ == STRING_TOKEN {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("type error: math operator '%s' does not support string arguments", op.Value)}
		}

		if argCount == 0 {
			finalType = typ
		} else if typ == FLOAT_TOKEN {
			finalType = FLOAT_TOKEN
		}
		argCount++
	}

	if isComparisonOp(op.Type) {
		finalType = INTEGER_TOKEN
	} else if argCount == 1 {
		t2, ok := a.stacks[targetStack].Pop()
		if !ok {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("operation '%s' with 1 arg requires 1 more element on '%s'", op.Value, targetStack)}
		}
		if t2 == STRING_TOKEN {
			return &CompilerError{op.Line, op.Column, op.Value, fmt.Sprintf("type error: math operator '%s' does not support string arguments", op.Value)}
		}
		if t2 == FLOAT_TOKEN {
			finalType = FLOAT_TOKEN
		}
	}

	a.stacks[targetStack].Push(finalType)
	return nil
}

func (a *Analyzer) handleGreedy(targetStack string, op Operation) *CompilerError {
	for {
		if a.pos+1 >= len(a.ir) {
			return &CompilerError{op.Line, op.Column, op.Value, "greedy operator missing ';' terminator"}
		}
		if a.ir[a.pos+1].Type == SEMICOLON_TOKEN {
			a.pos++
			break
		}
		typ, err := a.resolveExpression(targetStack)
		if err != nil {
			return err
		}
		if op.Type == PUT_TOKEN {
			a.stacks[targetStack].Push(typ)
		}
	}
	return nil
}

func (a *Analyzer) handleBind(op Operation) *CompilerError {
	a.pos++
	if a.pos >= len(a.ir) || a.ir[a.pos].Type != IDENT_TOKEN {
		return &CompilerError{op.Line, op.Column, op.Value, "bind requires name"}
	}
	name := a.ir[a.pos].Value

	a.pos++
	if a.pos >= len(a.ir) {
		return &CompilerError{op.Line, op.Column, op.Value, "bind missing target"}
	}

	targetStack := a.currentDefault()
	targetOp := a.ir[a.pos]

	if targetOp.Type == IDENT_TOKEN && a.pos+1 < len(a.ir) {
		nn := a.ir[a.pos+1]
		switch nn.Type {
		case DOT_TOKEN:
			targetStack = targetOp.Value
			a.pos += 2
			if a.pos >= len(a.ir) {
				return &CompilerError{targetOp.Line, targetOp.Column, targetOp.Value, "trailing stack prefix in bind"}
			}
			targetOp = a.ir[a.pos]
		case PEEK_TOKEN:
			targetStack = targetOp.Value
			a.pos += 1
			targetOp = a.ir[a.pos]
		}
	}

	if targetOp.Type != PEEK_TOKEN {
		return &CompilerError{targetOp.Line, targetOp.Column, targetOp.Value, "bind must target a peek index (e.g., .0)"}
	}

	index := 0
	fmt.Sscanf(targetOp.Value, ".%d", &index)

	if _, ok := a.stacks[targetStack]; !ok {
		a.stacks[targetStack] = &ShadowStack{Types: []TokenType{}}
	}

	typ, ok := a.stacks[targetStack].Peek(index)
	if !ok {
		return &CompilerError{targetOp.Line, targetOp.Column, targetOp.Value, fmt.Sprintf("bind target out of bounds on '%s' (size: %d)", targetStack, len(a.stacks[targetStack].Types))}
	}

	a.binds[name] = BindInfo{targetStack, index, typ}

	a.pos++
	if a.pos >= len(a.ir) || a.ir[a.pos].Type != SEMICOLON_TOKEN {
		return &CompilerError{op.Line, op.Column, op.Value, "bind missing ';'"}
	}

	return nil
}

func (a *Analyzer) handleRequire(targetStack string, op Operation) *CompilerError {
	newTypes := []TokenType{}
	for {
		a.pos++
		if a.pos >= len(a.ir) {
			return &CompilerError{op.Line, op.Column, op.Value, "require missing ';'"}
		}
		next := a.ir[a.pos]
		if next.Type == SEMICOLON_TOKEN {
			break
		}

		switch next.Value {
		case "int":
			newTypes = append(newTypes, INTEGER_TOKEN)
		case "float":
			newTypes = append(newTypes, FLOAT_TOKEN)
		case "string":
			newTypes = append(newTypes, STRING_TOKEN)
		default:
			return &CompilerError{next.Line, next.Column, next.Value, fmt.Sprintf("unknown type in require: %s", next.Value)}
		}
	}
	a.stacks[targetStack].Types = newTypes
	return nil
}
