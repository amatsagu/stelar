**Stelar** is a strongly-typed, prefix-arity concatenative programming language. Created as a study project to learn more about how programming languages are being created and with the goal of building a **self-hosted, compiled language**. It is heavily inspired by the philosophy of [Porth](https://gitlab.com/tsoding/porth) (and by extension: Forth), but modernized with a **Prefix-Arity** evaluation model, **Multi-Stack Support**, and **C-Transpilation** to leverage industry-standard optimizations.

## Evaluation Rules
1. **Scanner:** Tokenizes input into symbols, literals, and keywords.
2. **Linear IR Generation:** Converts tokens into a flat, contiguous array of `Operation` structs. This stage resolves control flow (jumps for `if`, `else`, `for`, `switch`) and performs early syntax validation.
3. **C-Transpilation:** Generates optimized C code from the Linear IR, leveraging the pre-calculated jump targets and validated structure.

## Compilation Pipeline
The compiler follows a multi-stage architecture:
- **Frontend:** Lexical analysis and Linear IR construction.
- **Middle-end:** (Planned) IR-level optimizations and type checking.
- **Backend:** C code generation and native binary compilation via `clang`/`gcc`/`msvc`.

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