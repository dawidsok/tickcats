package main

import (
	"fmt"
	"os"

	"github.com/dawidsok/tickcats/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "init":
		if err := store.Init("."); err != nil {
			return err
		}
		fmt.Println("Initialized .tickcats")
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println("TickCats")
	fmt.Println()
	fmt.Println("Usage: tickcats <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init       create .tickcats board folders and ignore them in git")
	fmt.Println()
	fmt.Println("Commands coming soon: new, list, move, pick-next")
}
