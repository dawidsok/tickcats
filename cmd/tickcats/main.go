// Package main is the CLI entry point for TickCats.
// It parses the optional --path flag and dispatches to subcommand handlers.
// When invoked without arguments it opens the interactive TUI; all other
// subcommands (init, new, list, move, pick-next, ids) are scriptable and
// print results to stdout.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
	"github.com/dawidsok/tickcats/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	boardPath := store.RootDir

	for i := 0; i < len(args); i++ {
		if args[i] == "--path" {
			if i+1 >= len(args) {
				return fmt.Errorf("--path requires an argument")
			}
			boardPath = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	if len(args) == 0 {
		return runTUI(boardPath)
	}

	switch args[0] {
	case "init":
		return runInit(boardPath)
	case "new":
		return runNew(args[1:], boardPath)
	case "list":
		return runList(boardPath)
	case "move":
		return runMove(args[1:], boardPath)
	case "pick-next":
		return runPickNext(args[1:], boardPath)
	case "ids":
		return runIDs(args[1:], boardPath)
	case "__complete":
		return runComplete(args[1:], boardPath)
	case "tui":
		return runTUI(boardPath)
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit(boardPath string) error {
	if err := store.Init(boardPath); err != nil {
		return err
	}
	fmt.Println("Initialized " + boardPath)
	return nil
}

func runNew(args []string, boardPath string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: tickcats new feat|task|bug <title>")
	}

	kind, err := ticket.ParseKind(args[0])
	if err != nil {
		return err
	}
	titleParts, acceptance := splitTitleAndAcceptance(args[1:])
	titleText := strings.Join(titleParts, " ")
	if strings.TrimSpace(titleText) == "" {
		return fmt.Errorf("ticket title cannot be empty")
	}

	path, err := store.Create(boardPath, kind, titleText, nil, ticket.PriorityP2, time.Now().UTC(), acceptance)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func runList(boardPath string) error {
	cfg, err := store.LoadConfig(boardPath)
	if err != nil {
		return err
	}

	board, err := store.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	printWarnings(board.Warnings)

	for _, col := range cfg.GetColumns() {
		state := store.State(col.ID)
		fmt.Printf("%s\n", col.DisplayName)
		for _, stored := range board.Columns[state] {
			fmt.Printf("  %s  %s  [%s] %s\n", stored.Name, displayID(stored.Ticket.ID), stored.Ticket.Priority, stored.Ticket.Title)
		}
	}
	return nil
}

func runMove(args []string, boardPath string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: tickcats move <ticket.md> <from-column> <to-column>")
	}

	cfg, err := store.LoadConfig(boardPath)
	if err != nil {
		return err
	}
	from, err := store.ResolveColumn(cfg, args[1])
	if err != nil {
		return err
	}
	to, err := store.ResolveColumn(cfg, args[2])
	if err != nil {
		return err
	}

	path, err := store.Move(boardPath, args[0], from, to)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func runPickNext(args []string, boardPath string) error {
	pathOnly, err := parsePickNextArgs(args)
	if err != nil {
		return err
	}

	board, err := store.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	printWarnings(board.Warnings)

	result := store.PickNext(board)
	if pathOnly {
		return printPickNextPath(result)
	}

	if !result.HasPick {
		fmt.Println("No ready ticket found")
		return nil
	}
	if result.NeedsChoice {
		fmt.Println("Tie candidates:")
		for _, tied := range result.Tied {
			fmt.Printf("  %s  %s  [%s] %s\n", tied.Name, displayID(tied.Ticket.ID), tied.Ticket.Priority, tied.Ticket.Title)
		}
		return nil
	}

	picked := result.Ticket
	fmt.Printf("%s  %s  [%s] %s\n", picked.Name, displayID(picked.Ticket.ID), picked.Ticket.Priority, picked.Ticket.Title)
	return nil
}

func runIDs(args []string, boardPath string) error {
	if len(args) != 1 || args[0] != "migrate" {
		return fmt.Errorf("usage: tickcats ids migrate")
	}
	result, err := store.MigrateIDs(boardPath)
	if err != nil {
		return err
	}
	fmt.Printf("Migrated %d ticket(s)\n", len(result.Migrated))
	for _, migrated := range result.Migrated {
		fmt.Printf("  %s  %s -> %s\n", migrated.ID, migrated.OldPath, migrated.NewPath)
	}
	return nil
}

func displayID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "—"
	}
	return id
}

func parsePickNextArgs(args []string) (bool, error) {
	pathOnly := false
	for _, arg := range args {
		switch arg {
		case "--path":
			pathOnly = true
		default:
			return false, fmt.Errorf("usage: tickcats pick-next [--path]")
		}
	}
	return pathOnly, nil
}

func printPickNextPath(result store.PickResult) error {
	if !result.HasPick {
		return fmt.Errorf("no ready ticket found")
	}
	if result.NeedsChoice {
		fmt.Fprintln(os.Stderr, "Tie candidates:")
		for _, tied := range result.Tied {
			fmt.Fprintln(os.Stderr, tied.Path)
		}
		return fmt.Errorf("multiple ready tickets tied for next pick")
	}
	fmt.Println(result.Ticket.Path)
	return nil
}

// splitTitleAndAcceptance splits CLI args at the "--ac" / "--acceptance" flag.
// Everything before the flag is the title words; everything after is joined as
// the acceptance-criteria string.
func splitTitleAndAcceptance(args []string) ([]string, string) {
	for i, arg := range args {
		if arg == "--ac" || arg == "--acceptance" {
			return args[:i], strings.Join(args[i+1:], " ")
		}
	}
	return args, ""
}

func runTUI(boardPath string) error {
	board, err := store.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	program := tea.NewProgram(tui.NewModelWithRoot(boardPath, board), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func runComplete(args []string, boardPath string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tickcats __complete tickets|columns")
	}

	switch args[0] {
	case "tickets":
		return completeTickets(boardPath)
	case "columns":
		return completeColumns(boardPath)
	default:
		return fmt.Errorf("usage: tickcats __complete tickets|columns")
	}
}

func completeTickets(boardPath string) error {
	cfg, err := store.LoadConfig(boardPath)
	if err != nil {
		return err
	}
	board, err := store.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	for _, col := range cfg.GetColumns() {
		for _, stored := range board.Columns[store.State(col.ID)] {
			fmt.Println(stored.Name)
		}
	}
	return nil
}

func completeColumns(boardPath string) error {
	cfg, err := store.LoadConfig(boardPath)
	if err != nil {
		return err
	}
	for _, col := range cfg.GetColumns() {
		fmt.Println(col.ID)
	}
	return nil
}

func printWarnings(warnings []store.Warning) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s: %v\n", warning.Path, warning.Err)
	}
}

func printHelp() {
	fmt.Println("TickCats")
	fmt.Println()
	fmt.Println("Usage: tickcats [--path <dir>] <command>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --path <dir>                 board directory (default: .tickcats)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init                         create board folders and ignore them in git")
	fmt.Println("  new feat|task|bug <title> [--ac text]  create ticket in backlog")
	fmt.Println("  list                         list tickets grouped by column")
	fmt.Println("  move <ticket> <from> <to>    move ticket between columns")
	fmt.Println("  pick-next [--path]           print next ready ticket")
	fmt.Println("  ids migrate                  add IDs to existing tickets and rename files")
	fmt.Println("  tui                          open terminal board")
}
