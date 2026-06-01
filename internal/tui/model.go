package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

var columnOrder = []store.State{store.StateBacklog, store.StateReady, store.StateDoing, store.StateDone, store.StateWontDo}

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

type ViewMode int

const (
	ViewBoard ViewMode = iota
	ViewDetail
	ViewCreate
	ViewConfig
)

type InteractionMode int

const (
	InteractionBoard InteractionMode = iota
	InteractionMove
	InteractionDeleteConfirm
	InteractionPostCreate
	InteractionSortPrompt
	InteractionQuitConfirm
	InteractionHelp
)

var createKinds = []ticket.Kind{ticket.KindFeature, ticket.KindTask, ticket.KindBug}
var createPriorities = []ticket.Priority{ticket.PriorityP0, ticket.PriorityP1, ticket.PriorityP2, ticket.PriorityP3}

// editorPresets are the selectable preset editor commands; empty string means "use $EDITOR".
var editorPresets = []string{"", "nvim", "vim", "nano", "code", "hx"}

type colorTheme struct {
	name   string
	colors []lipgloss.Color // indexed by columnOrder: backlog, ready, doing, done, wont-do
}

var colorThemes = []colorTheme{
	{name: "mono", colors: []lipgloss.Color{"#679", "#f8d", "#f8d", "#f8d", "#8a8a8a"}},
	{name: "gradient", colors: []lipgloss.Color{"#679", "#5fd787", "#f8d", "#679", "#8a8a8a"}},
	{name: "ocean", colors: []lipgloss.Color{"#679", "#0ff", "#0af", "#14a", "#6c7a89"}},
	{name: "fire", colors: []lipgloss.Color{"#679", "#fa0", "#f40", "#a00", "#875f5f"}},
	{name: "forest", colors: []lipgloss.Color{"#679", "#5fd787", "#5faf87", "#098a08", "#6b7d68"}},
}

type Model struct {
	Root            string
	Board           store.Board
	SelectedCol     int
	ColScrollOffset int
	SelectedRows    map[store.State]int
	ColumnScroll    map[store.State]int
	MultiSelected   map[store.State]map[string]bool
	Mode            ViewMode
	InteractionMode InteractionMode
	DetailScroll    int
	HelpScroll      int
	Status          string
	Width           int
	Height          int

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
	configField       int // 0=editor, 1=theme
	configEditorIdx   int
	configEditorInput textinput.Model

	watchCh <-chan struct{}

	notification *notification
	notifGen     int
}

func NewModel(board store.Board) Model {
	return NewModelWithRoot(".", board)
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
	m.syncManualOrder()
	m.applySortToBoard()
	m.Config, _ = store.LoadConfig(root)
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
