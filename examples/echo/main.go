package main

import (
	"fmt"
	"os"

	"github.com/tomasbasham/grpctest/examples/echo/cmd"
)

func main() {
	if err := cmd.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
