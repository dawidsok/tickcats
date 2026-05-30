package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	if len(args) == 0 {
		return runTUI()
	}

	switch args[0] {
	case "init":
		return runInit()
	case "new":
		return runNew(args[1:])
	case "list":
		return runList()
	case "move":
		return runMove(args[1:])
	case "pick-next":
		return runPickNext()
	case "tui":
		return runTUI()
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit() error {
	if err := store.Init("."); err != nil {
		return err
	}
	fmt.Println("Initialized .tickcats")
	return nil
}

func runNew(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: tickcats new feat|task|bug <title>")
	}

	kind, err := parseNewKind(args[0])
	if err != nil {
		return err
	}
	titleParts, acceptance := splitTitleAndAcceptance(args[1:])
	titleText := strings.Join(titleParts, " ")
	if strings.TrimSpace(titleText) == "" {
		return fmt.Errorf("ticket title cannot be empty")
	}

	if err := store.Init("."); err != nil {
		return err
	}

	now := time.Now().UTC()
	content := ticket.NewMarkdown(kind, titleText, ticket.PriorityP2, now, acceptance)
	name := filename(now, titleText)
	path := filepath.Join(store.StateDir(store.StateBacklog), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write ticket %q: %w", path, err)
	}

	fmt.Println(path)
	return nil
}

func runList() error {
	board, err := store.LoadBoard(".")
	if err != nil {
		return err
	}
	printWarnings(board.Warnings)

	for _, state := range store.ValidStates {
		fmt.Printf("%s\n", state)
		for _, stored := range board.Columns[state] {
			fmt.Printf("  %s  [%s] %s\n", stored.Name, stored.Ticket.Priority, stored.Ticket.Title)
		}
	}
	return nil
}

func runMove(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: tickcats move <ticket.md> <from-state> <to-state>")
	}

	from, err := store.ParseState(args[1])
	if err != nil {
		return err
	}
	to, err := store.ParseState(args[2])
	if err != nil {
		return err
	}

	path, err := store.Move(".", args[0], from, to)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func runPickNext() error {
	board, err := store.LoadBoard(".")
	if err != nil {
		return err
	}
	printWarnings(board.Warnings)

	result := store.PickNext(board)
	if !result.HasPick {
		fmt.Println("No ready ticket found")
		return nil
	}
	if result.NeedsChoice {
		fmt.Println("Tie candidates:")
		for _, tied := range result.Tied {
			fmt.Printf("  %s  [%s] %s\n", tied.Name, tied.Ticket.Priority, tied.Ticket.Title)
		}
		return nil
	}

	picked := result.Ticket
	fmt.Printf("%s  [%s] %s\n", picked.Name, picked.Ticket.Priority, picked.Ticket.Title)
	return nil
}

func splitTitleAndAcceptance(args []string) ([]string, string) {
	for i, arg := range args {
		if arg == "--ac" || arg == "--acceptance" {
			return args[:i], strings.Join(args[i+1:], " ")
		}
	}
	return args, ""
}

func runTUI() error {
	board, err := store.LoadBoard(".")
	if err != nil {
		return err
	}
	program := tea.NewProgram(tui.NewModel(board))
	_, err = program.Run()
	return err
}

func parseNewKind(raw string) (ticket.Kind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "feat", "feature":
		return ticket.KindFeature, nil
	case "task":
		return ticket.KindTask, nil
	case "bug", "fix":
		return ticket.KindBug, nil
	default:
		return "", fmt.Errorf("unknown ticket kind %q", raw)
	}
}

func filename(now time.Time, title string) string {
	return now.UTC().Format("20060102-1504") + "-" + slug(title) + ".md"
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func slug(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	slug := nonSlugChars.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "ticket"
	}
	return slug
}

func printWarnings(warnings []store.Warning) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: skipped %s: %v\n", warning.Path, warning.Err)
	}
}

func printHelp() {
	fmt.Println("TickCats")
	fmt.Println()
	fmt.Println("Usage: tickcats <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init                         create .tickcats board folders and ignore them in git")
	fmt.Println("  new feat|task|bug <title> [--ac text]  create ticket in backlog")
	fmt.Println("  list                         list tickets grouped by state")
	fmt.Println("  move <ticket> <from> <to>    move ticket between states")
	fmt.Println("  pick-next                    print next ready ticket")
	fmt.Println("  tui                          open terminal board")
}
