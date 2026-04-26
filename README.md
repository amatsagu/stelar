**Stelar** is a strongly-typed, prefix-arity concatenative programming language. Created as a study project to learn more about how programming languages are being created and with the goal of building a **self-hosted, compiled language**. It is heavily inspired by the philosophy of [Porth](https://gitlab.com/tsoding/porth) (and by extension: Forth), but modernized with a **Prefix-Arity** evaluation model, **Multi-Stack Support**, and **C-Transpilation** to leverage industry-standard optimizations.

## Evaluation Rules
1. **Scanner:** Tokenizes input into symbols, literals, and keywords.
2. **Arity Lookup:** Every token has a defined arity N.
3. **Recursive Descent:** When the compiler hits a token with arity N, it recursively resolves the next N sub-expressions before applying the current operation.

## Transpilation Strategy (Go -> C)
Stelar is designed for a single-pass transpilation to C.
- **Stacks:** Implemented as dynamic arrays in C (`realloc` strategy) with a 2x growth factor and 0.25x shrink hysteresis to prevent memory bloat.
- **Heap Elision:** Local stacks defined inside `proc` blocks are transpiled to local C structs. `clang -O3` will typically optimize these entirely into registers.
- **Type Checking:** The compiler maintains a shadow `TypeStack` during the parsing phase to validate `require` statements and struct field access at compile-time.

## Implementation Roadmap
Since the end goal is a self-hosted compiler, the project will follow a strict bootstrapping sequence:

1. **Phase 1 (Go-based Compiler):** - Lexer, Parser, and Type-Checker written in Go.
   - AST to C-Emitter.
2. **Phase 2 (Standard Library):**
   - Implement core Syscalls (Open, Read, Write, Close) in Stelar using `foreign` C-bindings.
3. **Phase 3 (Self-Hosting):**
   - Rewrite the Lexer, Parser, and C-Emitter entirely in Stelar.
   - Compile the Stelar-written compiler using the Go-written compiler.
   - Stelar is now self-hosted.

```bash
# Example compilation pipeline
stelar build main.stelar -> main.c -> clang -O3 -> main
```