// fmtcheck reports Go files that are not formatted.
// Keeping this check in Go makes the Makefile work on Windows and Unix.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("gofmt", "-l", ".")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt check failed: %v\n", err)
		os.Exit(1)
	}

	files := strings.TrimSpace(string(output))
	if files == "" {
		return
	}

	fmt.Println("these files need formatting:")
	fmt.Println(files)
	os.Exit(1)
}
