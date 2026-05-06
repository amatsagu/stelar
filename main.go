package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: stelar <filename>")
		return
	}

	// content, err := os.ReadFile(os.Args[1])
	// if err != nil {
	// 	fmt.Printf("Error: %v\n", err)
	// 	return
	// }

}
