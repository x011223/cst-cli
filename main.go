package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "mvn", "maven":
		if err := RunMvnBuild(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("cst-cli %s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`cst-cli - a collection of developer commands

Usage:
  cst-cli <command> [arguments]

Available Commands:
  mvn     List Maven projects in the current directory and build them
          (clean → compile → package, parallel across projects, serial per project)
  help    Show this help message
  version Show the version

Examples:
  cst-cli mvn        Run the interactive Maven build selector
  cst-cli version    Print the version
`)
}
