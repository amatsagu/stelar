package main

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Debug         bool
	Silent        bool
	Run           bool
	CompilerName  string
	CompilerFlags []string
}

func printUsage() {
	fmt.Print(`Usage: stelar [OPTIONS] <file> [COMPILER_ARGS...]

OPTIONS:
  -debug              Enable debug mode. Keeps the generated output.c for inspection 
                      instead of using a hidden temporary path.
  -silent             Ignore warnings. Warnings won't stop compilation but hint at 
                      ineffective code or potentially unhandled errors.
  -run                Run the application immediately after successful compilation.
  -compiler <name>    Name of the C compiler to use. 
                      (Default fallback chain: clang -> msvc -> cc)
  -help               Print this help to stdout and exit.

Note: Any unknown flags (e.g., -O2, -Wall) are passed directly to the underlying C compiler.
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	config := Config{
		CompilerFlags: make([]string, 0),
	}

	var inputFileName string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-debug", "-d":
				config.Debug = true
			case "-silent", "-s":
				config.Silent = true
			case "-run", "-r":
				config.Run = true
			case "-compiler", "-c":
				if i+1 < len(os.Args) {
					config.CompilerName = os.Args[i+1]
					i++
				} else {
					fmt.Println("ERROR :: -compiler flag requires a compiler name argument.")
					os.Exit(1)
				}
			case "-help", "-h":
				printUsage()
				os.Exit(0)
			default:
				fmt.Printf("ERROR :: Unknown flag: %s\n", arg)
				os.Exit(1)
			}
		} else if inputFileName == "" {
			inputFileName = arg
		} else {
			config.CompilerFlags = append(config.CompilerFlags, arg)
		}
	}

	if inputFileName == "" {
		fmt.Printf("ERROR :: No input file specified. Run \"%s -help\" to see more.\n", os.Args[0])
		os.Exit(1)
	}

	processFile(inputFileName, &config)
}

func processFile(filepath string, config *Config) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("ERROR :: Failed to read file \"%s\": %v\n", filepath, err)
		os.Exit(1)
	}

	sourceCode := string(content)

	lexer := NewLexer(sourceCode)
	parser := NewParser(lexer)

	fmt.Printf("INFO :: Parsing \"%s\" file...\n", filepath)
	ir, errs := parser.Parse()
	if len(errs) != 0 {
		fmt.Printf("ERROR :: Detected %d error(s) while generating IR:\n", len(errs))

		for _, err := range errs {
			ReportError(filepath, sourceCode, err.Line, err.Column, err.Literal, err.Message)
		}

		os.Exit(1)
	}

	if config.Debug {
		fmt.Printf("DEBUG :: Generated IR for \"%s\":\n", filepath)
		DebugPrintIR(ir)
	}

	// TODO: Do something with IR (lol)
}
