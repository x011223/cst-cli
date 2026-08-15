package main

import (
	"os"

	"github.com/wujunqiang/cst-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
