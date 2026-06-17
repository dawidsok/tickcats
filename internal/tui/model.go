// Package tui implements the interactive terminal UI for TickCats using the
// Bubble Tea framework (github.com/charmbracelet/bubbletea). The UI follows
// Bubble Tea's Model-View-Update (MVU) pattern: a single Model value holds all
// UI state, Update handles incoming events and returns a new Model, and View
// renders the current state to a string.
//
// model.go defines the Model struct and its constructors, global style
// variables, enum types for view and interaction modes, and per-create/config
// working state fields.
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

const minColumnWidth = 60 // minimum total width per column (including borders/margin)

var (
	selectedStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	mutedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bannerStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	notifSuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd787"))
	notifErrorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff5f5f"))
)

type notifKind int

const (
	notifInfo notifKind = iota
	notifSuccess
	notifError
)

type notification struct {
	text string
	kind notifKind
	gen  int
}

type clearNotificationMsg struct{ gen int }

// ViewMode controls which full-screen view is rendered.
type ViewMode int

const (
	ViewBoard  ViewMode = iota // kanban column layout
	ViewDetail                 // single-ticket detail panel
	ViewCreate                 // new ticket form
	ViewConfig                 // settings form
)

// InteractionMode is an overlay state layered on top of the current ViewMode.
// Most overlays (dialogs, move mode) keep the underlying board visible in the
// footer but suspend normal board key bindings.
type InteractionMode int

const (
	InteractionBoard         InteractionMode = iota // normal board navigation
	InteractionMove                                 // moving ticket(s) between columns
	InteractionDeleteConfirm                        // "delete?" prompt
	InteractionPostCreate                           // "open in editor?" prompt after create
	InteractionSortPrompt                           // "switch to manual sort?" prompt
	InteractionQuitConfirm                          // "quit?" prompt
	InteractionHelp                                 // help dialog overlay
	InteractionSearch                               // fuzzy search bar overlay
)

var createKinds = []ticket.Kind{ticket.KindFeature, ticket.KindTask, ticket.KindBug}
var createPriorities = []ticket.Priority{ticket.PriorityP0, ticket.PriorityP1, ticket.PriorityP2, ticket.PriorityP3}

// editorPresets are the selectable preset editor commands; empty string means "use $EDITOR".
var editorPresets = []string{"", "nvim", "vim", "nano", "code", "hx"}

type colorTheme struct {
	name         string
	backlogColor lipgloss.Color
	startColor   lipgloss.Color
	endColor     lipgloss.Color
}

// Themes define three colors: first/backlog column, gradient start, and
// gradient end. Per-column colors are generated deterministically at render time
// so dynamic boards are not limited to five hard-coded columns.
var colorThemes = []colorTheme{
	{name: "mono", backlogColor: "#88a", startColor: "#f8d", endColor: "#88a"},
	{name: "gradient", backlogColor: "#88a", startColor: "#fad", endColor: "#88a"},
	{name: "ocean", backlogColor: "#88a", startColor: "#8df", endColor: "#88a"},
	{name: "fire", backlogColor: "#88a", startColor: "#fd8", endColor: "#a88"},
	{name: "forest", backlogColor: "#88a", startColor: "#5fd787", endColor: "#8a8"},
	{name: "dim-sum", backlogColor: "#86837a", startColor: "#b77a4a", endColor: "#87965f"},
}

type configAction int

const (
	configActionNone configAction = iota
	configActionAddName
	configActionRename
	configActionDeleteConfirm
)

// Model is the complete UI state passed through every Update/View cycle.
// It is a value type — Update returns a new copy rather than mutating in place,
// except for pointer-receiver helpers that modify fields directly as a
// performance optimisation within the same cycle.
type Model struct {
	Root             string                          // absolute path to the board root directory
	Board            store.Board                     // last loaded board snapshot
	columnOrder      []store.State                   // dynamic list of columns derived from config
	SelectedCol      int                             // index into columnOrder for the focused column
	ColScrollOffset  int                             // first visible column index when board is wider than terminal
	SelectedRows     map[store.State]int             // focused row per column
	ColumnScroll     map[store.State]int             // scroll offset per column
	MultiSelected    map[store.State]map[string]bool // ticket filenames selected for batch move
	Mode             ViewMode
	InteractionMode  InteractionMode
	DetailScroll     int    // scroll offset in the detail body view
	HelpScroll       int    // scroll offset in the help dialog
	detailTicketName string // filename of the ticket open in ViewDetail; used to track it across reloads and column moves
	Status           string
	countPrefix      string
	Width            int
	Height           int

	createInput    textinput.Model
	createKind     ticket.Kind
	createPriority ticket.Priority
	createToRefine bool
	createField    int
	createPending  string

	prevMode            ViewMode
	prevInteractionMode InteractionMode

	SortMode    store.SortMode
	ManualOrder map[store.State][]string

	Config            store.Config
	configField       int // 0=editor, 1=theme, 2=columns
	configEditorIdx   int
	configEditorInput textinput.Model
	configColIdx      int
	configAction      configAction
	configColumnInput textinput.Model

	searchInput   textinput.Model
	searchFocused bool // true = typing in field; false = navigating results

	watchCh <-chan struct{}

	notification *notification
	notifGen     int
}

// NewModel creates a Model with the current working directory as the board root.
func NewModel(board store.Board) Model {
	return NewModelWithRoot(".", board)
}

// NewModelWithRoot creates a Model with an explicit board root path. It loads
// sort config and user config from disk and starts the file watcher goroutine.
func statesFromColumns(columns []store.Column) []store.State {
	states := make([]store.State, len(columns))
	for i, col := range columns {
		states[i] = store.State(col.ID)
	}
	return states
}

func NewModelWithRoot(root string, board store.Board) Model {
	m := Model{
		Root:          root,
		Board:         board,
		SelectedRows:  make(map[store.State]int),
		ColumnScroll:  make(map[store.State]int),
		MultiSelected: make(map[store.State]map[string]bool),
	}
	cfg, _ := store.LoadSortConfig(root)
	m.SortMode = cfg.Mode
	m.ManualOrder = cfg.ManualOrder
	m.Config, _ = store.LoadConfig(root)
	m.columnOrder = statesFromColumns(m.Config.GetColumns())
	m.syncManualOrder()
	m.applySortToBoard()
	if fw, err := newFileWatcher(root); err == nil {
		m.watchCh = fw.ch
	}
	return m
}

type msgFileChanged struct{}

func waitForWatchEvent(ch <-chan struct{}) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		<-ch
		return msgFileChanged{}
	}
}

func (m *Model) notify(text string, kind notifKind) tea.Cmd {
	m.notifGen++
	gen := m.notifGen
	m.notification = &notification{text: text, kind: kind, gen: gen}
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{gen: gen}
	})
}

func (m Model) Init() tea.Cmd {
	return waitForWatchEvent(m.watchCh)
}
