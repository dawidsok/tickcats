package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Mode == ViewCreate {
		return m.updateCreate(msg)
	}
	if m.Mode == ViewConfig {
		return m.updateConfig(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global exits — always honoured outside create/config views.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && m.InteractionMode != InteractionQuitConfirm {
			return m.enterQuitConfirm()
		}
		if m.InteractionMode == InteractionQuitConfirm {
			return m.updateQuitConfirm(msg)
		}
		if m.Mode == ViewDetail {
			return m.updateDetail(msg)
		}
		if m.InteractionMode == InteractionPostCreate {
			return m.updatePostCreate(msg)
		}
		if m.InteractionMode == InteractionDeleteConfirm {
			return m.updateDeleteConfirm(msg)
		}
		if m.InteractionMode == InteractionSortPrompt {
			return m.updateSortPrompt(msg)
		}
		if m.InteractionMode == InteractionMove {
			return m.updateMove(msg)
		}
		return m.updateBoard(msg)
	case clearNotificationMsg:
		if m.notification != nil && m.notification.gen == msg.gen {
			m.notification = nil
		}
		return m, nil
	case msgFileChanged:
		m.reloadBoard()
		return m, waitForWatchEvent(m.watchCh)
	case editorFinishedMsg:
		return m, m.handleEditorFinished(msg)
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}
	return m, nil
}

func (m Model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m.moveColumn(-1)
	case "l", "right":
		m.moveColumn(1)
	case "j", "down":
		m.moveRow(1)
	case "k", "up":
		m.moveRow(-1)
	case "d":
		m.pageRows(1)
	case "u":
		m.pageRows(-1)
	case "enter", "o":
		if m.selectedTicket() != nil {
			m.Mode = ViewDetail
			m.DetailScroll = 0
		}
	case "v":
		m.toggleSelection()
		if n := m.totalSelected(); n > 0 {
			m.Status = fmt.Sprintf("%d selected", n)
		} else {
			m.Status = ""
		}
	case "m":
		m.InteractionMode = InteractionMove
		if n := m.totalSelected(); n > 0 {
			m.Status = fmt.Sprintf("Move mode (%d selected): h/l col, H first, L last, esc cancel", n)
		} else {
			m.Status = "Move mode: h/l col, H first, L last, esc cancel"
		}
	case "n":
		return m.enterCreate()
	case "p":
		return m, m.moveSelected(1)
	case "b":
		return m, m.moveSelected(-1)
	case "e":
		return m.editSelected()
	case "x":
		m.enterDeleteConfirm()
	case "r":
		if m.reloadBoard() {
			return m, m.notify("Board reloaded", notifSuccess)
		}
	case "s":
		m.cycleSortMode()
	case "c":
		return m.enterConfig()
	}
	return m, nil
}

func (m Model) updateMove(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.InteractionMode = InteractionBoard
		m.Status = "Board mode"
	case "h", "left":
		return m, m.moveAllSelectedBy(-1)
	case "l", "right":
		return m, m.moveAllSelectedBy(1)
	case "H":
		return m, m.moveAllSelectedTo(0)
	case "L":
		return m, m.moveAllSelectedTo(len(columnOrder) - 1)
	case "j", "down":
		if m.SortMode == store.SortManual {
			m.moveSelectedInColumn(1)
		} else {
			m.InteractionMode = InteractionSortPrompt
			m.Status = "Switch to manual sort to reorder? y/n"
		}
	case "k", "up":
		if m.SortMode == store.SortManual {
			m.moveSelectedInColumn(-1)
		} else {
			m.InteractionMode = InteractionSortPrompt
			m.Status = "Switch to manual sort to reorder? y/n"
		}
	}
	return m, nil
}

func (m Model) updateSortPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "y":
		m.SortMode = store.SortManual
		m.syncManualOrder()
		m.applySortToBoard()
		m.saveSortConfig()
		m.InteractionMode = InteractionMove
		m.Status = "Switched to manual sort. Use j/k to reorder."
	case "n", "esc":
		m.InteractionMode = InteractionMove
		m.Status = "Move mode: h left, l right, esc cancel"
	}
	return m, nil
}

func (m Model) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "n", "esc":
		m.InteractionMode = InteractionBoard
		m.Status = "Delete cancelled"
	case "y":
		return m, m.deleteSelected()
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.Mode = ViewBoard
		m.DetailScroll = 0
	case "j", "down":
		m.moveDetailScroll(1)
	case "k", "up":
		m.moveDetailScroll(-1)
	case "d":
		m.moveDetailScroll(m.detailHalfPage())
	case "u":
		m.moveDetailScroll(-m.detailHalfPage())
	case "e":
		return m.editSelected()
	case "c":
		return m.enterConfig()
	}
	return m, nil
}

func (m Model) View() string {
	if m.Mode == ViewCreate {
		return m.renderCreate()
	}
	if m.Mode == ViewConfig {
		return m.renderConfig()
	}
	if m.InteractionMode == InteractionPostCreate {
		return m.renderPostCreateDialog()
	}
	if m.Mode == ViewDetail {
		return m.renderDetail()
	}

	var b strings.Builder
	b.WriteString(m.renderPickNext())
	b.WriteString("\n")
	b.WriteString(m.renderHScrollIndicator())
	b.WriteString(m.renderBoard())
	b.WriteString("\n")
	b.WriteString(m.renderFooter())
	return b.String()
}

func (m Model) enterQuitConfirm() (tea.Model, tea.Cmd) {
	m.prevMode = m.Mode
	m.prevInteractionMode = m.InteractionMode
	m.InteractionMode = InteractionQuitConfirm
	return m, nil
}

func (m Model) updateQuitConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "q":
		return m, tea.Quit
	case "n", "esc":
		m.Mode = m.prevMode
		m.InteractionMode = m.prevInteractionMode
	}
	return m, nil
}

func (m Model) footerText() string {
	if m.InteractionMode == InteractionQuitConfirm {
		return "QUIT? y/q confirm  n/esc cancel"
	}
	if m.InteractionMode == InteractionPostCreate {
		return "y open editor  n/esc stay  d don't ask again"
	}
	if m.InteractionMode == InteractionDeleteConfirm {
		return "DELETE? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionSortPrompt {
		return "Switch to manual sort? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionMove {
		sel := m.totalSelected()
		if sel > 0 {
			return fmt.Sprintf("MOVE MODE (%d selected): h/l col  H first  L last  j/k reorder  esc board  q quit", sel)
		}
		return "MOVE MODE: h/l col  H first  L last  j/k reorder (manual)  v select  esc board  q quit"
	}
	if m.Mode == ViewDetail {
		return "DETAIL MODE: j/k scroll  d/u half-page  e edit  c config  esc board  q quit"
	}
	return fmt.Sprintf("BOARD MODE: h/l col  j/k/d/u ticket  v select  m move  s sort(%s)  p progress  b back  o/enter detail  e edit  n new  x del  r reload  c config  q quit", m.SortMode)
}

func (m *Model) moveColumn(delta int) {
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(columnOrder)-1)
	m.ensureColVisible()
	m.ensureSelectedVisible(columnOrder[m.SelectedCol])
}

func (m *Model) ensureColVisible() {
	visible := m.visibleColumnCount()
	if m.SelectedCol < m.ColScrollOffset {
		m.ColScrollOffset = m.SelectedCol
	}
	if m.SelectedCol >= m.ColScrollOffset+visible {
		m.ColScrollOffset = m.SelectedCol - visible + 1
	}
	m.ColScrollOffset = clamp(m.ColScrollOffset, 0, max(0, len(columnOrder)-visible))
}

func (m *Model) moveRow(delta int) {
	state := columnOrder[m.SelectedCol]
	rows := len(m.Board.Columns[state])
	if rows == 0 {
		m.SelectedRows[state] = 0
		m.ColumnScroll[state] = 0
		return
	}
	m.SelectedRows[state] = clamp(m.SelectedRows[state]+delta, 0, rows-1)
	m.ensureSelectedVisible(state)
}

func (m *Model) ensureSelectedVisible(state store.State) {
	rows := len(m.Board.Columns[state])
	if rows == 0 {
		m.SelectedRows[state] = 0
		m.ColumnScroll[state] = 0
		return
	}

	selected := clamp(m.SelectedRows[state], 0, rows-1)
	m.SelectedRows[state] = selected
	scroll := clamp(m.ColumnScroll[state], 0, rows-1)
	if scroll > selected {
		scroll = selected
	}

	budget := m.columnLineBudget()
	innerWidth := m.columnInnerWidth()
	for scroll < selected && !m.columnRangeFits(state, scroll, selected, budget, innerWidth) {
		scroll++
	}
	for scroll > 0 && m.columnRangeFits(state, scroll-1, selected, budget, innerWidth) {
		scroll--
	}

	m.ColumnScroll[state] = clamp(scroll, 0, rows-1)
}

func (m Model) columnRangeFits(state store.State, start int, selected int, budget int, innerWidth int) bool {
	if budget <= 0 {
		return false
	}
	used := 0
	if start > 0 {
		used++ // "above" indicator
	}
	for row := start; row <= selected; row++ {
		if row > start {
			used++ // separator before this ticket
		}
		used += len(m.ticketColumnLines(state, row, innerWidth))
		if used > budget {
			return false
		}
	}
	return true
}

func (m *Model) moveDetailScroll(delta int) {
	maxScroll := len(m.detailLines()) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.DetailScroll = clamp(m.DetailScroll+delta, 0, maxScroll)
}

func (m *Model) pageRows(dir int) {
	half := max(1, m.visibleTicketRows()/2)
	for range half {
		m.moveRow(dir)
	}
}

func (m Model) detailHalfPage() int {
	return max(1, m.detailPanelInnerHeight()/2)
}

func (m *Model) enterDeleteConfirm() {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return
	}
	m.InteractionMode = InteractionDeleteConfirm
	m.Status = fmt.Sprintf("Delete %s?", stored.Name)
}

func (m *Model) deleteSelected() tea.Cmd {
	stored := m.selectedTicket()
	if stored == nil {
		m.InteractionMode = InteractionBoard
		m.Status = "No ticket selected"
		return nil
	}

	name := stored.Name
	if _, err := store.Trash(m.Root, name, stored.State); err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Delete failed: " + err.Error()
		return nil
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Reload failed: " + err.Error()
		return nil
	}

	m.Board = board
	m.InteractionMode = InteractionBoard
	m.Status = ""
	return m.notify("Deleted "+name, notifSuccess)
}

func (m Model) editSelected() (tea.Model, tea.Cmd) {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return m, nil
	}

	cmd := editorCommand(stored.Path, m.Config.Editor)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

type editorFinishedMsg struct {
	err error
}

func (m *Model) handleEditorFinished(msg editorFinishedMsg) tea.Cmd {
	if msg.err != nil {
		m.Status = "Edit failed: " + msg.err.Error()
		return nil
	}
	m.reloadBoard()
	return m.notify("Edited ticket", notifSuccess)
}

func (m *Model) reloadBoard() bool {
	state := columnOrder[m.SelectedCol]
	focusedName := ""
	if tickets := m.Board.Columns[state]; m.SelectedRows[state] < len(tickets) {
		focusedName = tickets[m.SelectedRows[state]].Name
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return false
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	m.syncMultiSelected()

	if focusedName == "" {
		return true
	}
	newTickets := m.Board.Columns[state]
	for i, t := range newTickets {
		if t.Name == focusedName {
			m.SelectedRows[state] = i
			m.ensureSelectedVisible(state)
			return true
		}
	}
	if m.SelectedRows[state] >= len(newTickets) && len(newTickets) > 0 {
		m.SelectedRows[state] = len(newTickets) - 1
	}
	m.ensureSelectedVisible(state)
	return true
}

func (m *Model) cycleSortMode() {
	for i, mode := range store.SortModes {
		if mode == m.SortMode {
			m.SortMode = store.SortModes[(i+1)%len(store.SortModes)]
			m.syncManualOrder()
			m.applySortToBoard()
			m.saveSortConfig()
			m.Status = "Sort: " + string(m.SortMode)
			return
		}
	}
	m.SortMode = store.SortPriority
}

func (m *Model) saveSortConfig() {
	_ = store.SaveSortConfig(m.Root, store.SortConfig{
		Mode:        m.SortMode,
		ManualOrder: m.ManualOrder,
	})
}

func (m *Model) syncManualOrder() {
	if m.ManualOrder == nil {
		m.ManualOrder = make(map[store.State][]string)
	}
	for state, tickets := range m.Board.Columns {
		existing := m.ManualOrder[state]
		existingSet := make(map[string]bool, len(existing))
		for _, name := range existing {
			existingSet[name] = true
		}
		// Append new tickets not yet in manual order
		for _, t := range tickets {
			if !existingSet[t.Name] {
				m.ManualOrder[state] = append(m.ManualOrder[state], t.Name)
			}
		}
		// Remove tickets that no longer exist
		ticketSet := make(map[string]bool, len(tickets))
		for _, t := range tickets {
			ticketSet[t.Name] = true
		}
		filtered := m.ManualOrder[state][:0]
		for _, name := range m.ManualOrder[state] {
			if ticketSet[name] {
				filtered = append(filtered, name)
			}
		}
		m.ManualOrder[state] = filtered
	}
}

func (m *Model) applySortToBoard() {
	for _, state := range columnOrder {
		tickets := m.Board.Columns[state]
		if len(tickets) <= 1 {
			continue
		}
		sorted := make([]store.StoredTicket, len(tickets))
		copy(sorted, tickets)
		switch m.SortMode {
		case store.SortPriority:
			sort.SliceStable(sorted, func(i, j int) bool {
				ri, rj := sorted[i].Ticket.Priority.Rank(), sorted[j].Ticket.Priority.Rank()
				if ri != rj {
					return ri < rj
				}
				return sorted[i].Name < sorted[j].Name
			})
		case store.SortTitle:
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Ticket.Title < sorted[j].Ticket.Title
			})
		case store.SortDate:
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Ticket.Created.Before(sorted[j].Ticket.Created)
			})
		case store.SortManual:
			order := m.ManualOrder[state]
			idx := make(map[string]int, len(order))
			for i, name := range order {
				idx[name] = i
			}
			sort.SliceStable(sorted, func(i, j int) bool {
				ii, iok := idx[sorted[i].Name]
				ji, jok := idx[sorted[j].Name]
				if iok && jok {
					return ii < ji
				}
				if iok {
					return true
				}
				if jok {
					return false
				}
				return sorted[i].Name < sorted[j].Name
			})
		}
		m.Board.Columns[state] = sorted
	}
}

func (m *Model) moveSelectedInColumn(delta int) {
	state := columnOrder[m.SelectedCol]
	stored := m.selectedTicket()
	if stored == nil {
		return
	}
	order := m.ManualOrder[state]
	for i, name := range order {
		if name == stored.Name {
			newI := i + delta
			if newI < 0 || newI >= len(order) {
				return
			}
			order[i], order[newI] = order[newI], order[i]
			m.ManualOrder[state] = order
			m.applySortToBoard()
			m.SelectedRows[state] = findTicketRow(m.Board.Columns[state], stored.Name)
			m.ensureSelectedVisible(state)
			m.saveSortConfig()
			return
		}
	}
}

func (m Model) totalSelected() int {
	n := 0
	for _, s := range m.MultiSelected {
		n += len(s)
	}
	return n
}

func (m *Model) toggleSelection() {
	stored := m.selectedTicket()
	if stored == nil {
		return
	}
	state := columnOrder[m.SelectedCol]
	if m.MultiSelected[state] == nil {
		m.MultiSelected[state] = make(map[string]bool)
	}
	if m.MultiSelected[state][stored.Name] {
		delete(m.MultiSelected[state], stored.Name)
		if len(m.MultiSelected[state]) == 0 {
			delete(m.MultiSelected, state)
		}
	} else {
		m.MultiSelected[state][stored.Name] = true
	}
}

func (m *Model) syncMultiSelected() {
	for state, names := range m.MultiSelected {
		ticketSet := make(map[string]bool, len(m.Board.Columns[state]))
		for _, t := range m.Board.Columns[state] {
			ticketSet[t.Name] = true
		}
		for name := range names {
			if !ticketSet[name] {
				delete(names, name)
			}
		}
		if len(names) == 0 {
			delete(m.MultiSelected, state)
		}
	}
}

type selectedRef struct {
	name   string
	state  store.State
	colIdx int
}

func (m *Model) allSelectedRefs() []selectedRef {
	var refs []selectedRef
	for colIdx, state := range columnOrder {
		for name := range m.MultiSelected[state] {
			refs = append(refs, selectedRef{name, state, colIdx})
		}
	}
	return refs
}

func (m *Model) moveAllSelectedBy(delta int) tea.Cmd {
	if m.totalSelected() == 0 {
		return m.moveSelected(delta)
	}

	refs := m.allSelectedRefs()
	if delta > 0 {
		sort.Slice(refs, func(i, j int) bool { return refs[i].colIdx > refs[j].colIdx })
	} else {
		sort.Slice(refs, func(i, j int) bool { return refs[i].colIdx < refs[j].colIdx })
	}

	moved := 0
	for _, r := range refs {
		toIdx := r.colIdx + delta
		if toIdx < 0 || toIdx >= len(columnOrder) {
			continue
		}
		if _, err := store.Move(m.Root, r.name, r.state, columnOrder[toIdx]); err != nil {
			m.Status = "Move failed: " + err.Error()
			return nil
		}
		moved++
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return nil
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()

	newSelected := make(map[store.State]map[string]bool)
	for _, r := range refs {
		newIdx := r.colIdx + delta
		if newIdx < 0 || newIdx >= len(columnOrder) {
			newIdx = r.colIdx
		}
		newState := columnOrder[newIdx]
		if newSelected[newState] == nil {
			newSelected[newState] = make(map[string]bool)
		}
		newSelected[newState][r.name] = true
	}
	m.MultiSelected = newSelected
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(columnOrder)-1)
	m.ensureColVisible()

	if moved == 0 {
		if delta > 0 {
			m.Status = "Ticket(s) already at last column"
		} else {
			m.Status = "Ticket(s) already at first column"
		}
		return nil
	}
	return m.notify(fmt.Sprintf("Moved %d ticket(s)", moved), notifSuccess)
}

func (m *Model) moveAllSelectedTo(targetCol int) tea.Cmd {
	if m.totalSelected() == 0 {
		return m.moveSelected(targetCol - m.SelectedCol)
	}

	refs := m.allSelectedRefs()
	moved := 0
	for _, r := range refs {
		if r.colIdx == targetCol {
			continue
		}
		if _, err := store.Move(m.Root, r.name, r.state, columnOrder[targetCol]); err != nil {
			m.Status = "Move failed: " + err.Error()
			return nil
		}
		moved++
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return nil
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()

	newSelected := make(map[store.State]map[string]bool)
	targetState := columnOrder[targetCol]
	newSelected[targetState] = make(map[string]bool)
	for _, r := range refs {
		newSelected[targetState][r.name] = true
	}
	m.MultiSelected = newSelected
	m.SelectedCol = targetCol
	m.ensureColVisible()

	if moved == 0 {
		m.Status = fmt.Sprintf("Ticket(s) already at %s", columnOrder[targetCol])
		return nil
	}
	return m.notify(fmt.Sprintf("Moved %d ticket(s) to %s", moved, columnOrder[targetCol]), notifSuccess)
}

func (m Model) enterCreate() (tea.Model, tea.Cmd) {
	input := textinput.New()
	input.Placeholder = "ticket title"
	input.CharLimit = 200
	m.createInput = input
	m.createKind = ticket.KindFeature
	m.createPriority = ticket.PriorityP2
	m.createToRefine = true
	m.createField = 1
	m.Mode = ViewCreate
	m.Status = ""
	cmd := m.createInput.Focus()
	return m, cmd
}

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.Mode = ViewBoard
			m.Status = ""
			return m, nil
		case "tab":
			m.createField = (m.createField + 1) % 4
			return m, m.syncCreateFocus()
		case "shift+tab":
			m.createField = (m.createField + 3) % 4
			return m, m.syncCreateFocus()
		case "h", "left":
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 0 {
				m.cycleKind(-1)
			} else if m.createField == 2 {
				m.cyclePriority(-1)
			}
			return m, nil
		case "l", "right":
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 0 {
				m.cycleKind(1)
			} else if m.createField == 2 {
				m.cyclePriority(1)
			}
			return m, nil
		case "enter":
			if strings.TrimSpace(m.createInput.Value()) == "" {
				m.Status = "Title required"
				return m, nil
			}
			return m.submitCreate()
		default:
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 3 && keyMsg.String() == " " {
				m.createToRefine = !m.createToRefine
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.createInput, cmd = m.createInput.Update(msg)
	return m, cmd
}

func (m *Model) syncCreateFocus() tea.Cmd {
	if m.createField == 1 {
		return m.createInput.Focus()
	}
	m.createInput.Blur()
	return nil
}

func (m *Model) cycleKind(delta int) {
	for i, k := range createKinds {
		if k == m.createKind {
			m.createKind = createKinds[(i+len(createKinds)+delta)%len(createKinds)]
			return
		}
	}
	m.createKind = createKinds[0]
}

func (m *Model) cyclePriority(delta int) {
	for i, p := range createPriorities {
		if p == m.createPriority {
			m.createPriority = createPriorities[(i+len(createPriorities)+delta)%len(createPriorities)]
			return
		}
	}
	m.createPriority = ticket.PriorityP2
}

func (m Model) submitCreate() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.createInput.Value())
	var labels []string
	if m.createToRefine {
		labels = []string{ticket.LabelToRefine}
	}
	path, err := store.Create(m.Root, m.createKind, title, labels, m.createPriority, time.Now().UTC())
	if err != nil {
		m.Status = "Create failed: " + err.Error()
		return m, nil
	}
	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return m, nil
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	m.Mode = ViewBoard
	m.createPending = path
	if m.Config.SkipEditorPrompt {
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(path), notifSuccess)
	}
	m.InteractionMode = InteractionPostCreate
	m.Status = ""
	return m, nil
}

func (m Model) updatePostCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y":
		cmd := editorCommand(m.createPending, m.Config.Editor)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		m.InteractionMode = InteractionBoard
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	case "n", "esc":
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(m.createPending), notifSuccess)
	case "d":
		m.Config.SkipEditorPrompt = true
		_ = store.SaveConfig(m.Root, m.Config)
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(m.createPending)+" (won't ask again)", notifSuccess)
	}
	return m, nil
}

func (m Model) renderCreate() string {
	formWidth := m.fullWidth() - 4
	if formWidth > 60 {
		formWidth = 60
	}
	if formWidth < 44 {
		formWidth = 44
	}

	labelW := 9
	labelStyle := mutedStyle.Width(labelW)
	activeLabel := selectedStyle.Width(labelW)

	kindLabel := labelStyle.Render("Kind")
	titleLabel := labelStyle.Render("Title")
	priorityLabel := labelStyle.Render("Priority")
	refineLabel := labelStyle.Render("To Refine")
	switch m.createField {
	case 0:
		kindLabel = activeLabel.Render("Kind")
	case 1:
		titleLabel = activeLabel.Render("Title")
	case 2:
		priorityLabel = activeLabel.Render("Priority")
	case 3:
		refineLabel = activeLabel.Render("To Refine")
	}

	inputWidth := formWidth - labelW - 6
	if inputWidth < 10 {
		inputWidth = 10
	}
	m.createInput.Width = inputWidth

	checkbox := "[ ]"
	if m.createToRefine {
		checkbox = selectedStyle.Render("[x]")
	}

	kindRow := kindLabel + m.renderKindOptions()
	titleRow := titleLabel + m.createInput.View()
	priorityRow := priorityLabel + m.renderPriorityOptions()
	refineRow := refineLabel + checkbox
	helpRow := mutedStyle.Render("tab/shift-tab field  h/l change  space toggle  enter create  esc cancel")

	content := strings.Join([]string{kindRow, "", titleRow, "", priorityRow, "", refineRow, "", helpRow}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(formWidth).
		Render(content)

	var statusLine string
	if m.Status != "" {
		statusLine = "\n" + mutedStyle.Render(m.Status)
	}

	h := m.Height
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(m.fullWidth(), h, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("New Ticket")+"\n\n"+box+statusLine)
}

func (m Model) renderKindOptions() string {
	parts := make([]string, 0, len(createKinds))
	for _, k := range createKinds {
		name := string(k)
		if k == m.createKind {
			if m.createField == 0 {
				parts = append(parts, selectedStyle.Render("["+name+"]"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			parts = append(parts, mutedStyle.Render(name))
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderPriorityOptions() string {
	parts := make([]string, 0, len(createPriorities))
	for _, p := range createPriorities {
		name := string(p)
		if p == m.createPriority {
			if m.createField == 2 {
				parts = append(parts, selectedStyle.Render("["+name+"]"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			parts = append(parts, mutedStyle.Render(name))
		}
	}
	return strings.Join(parts, "  ")
}

func (m *Model) moveSelected(delta int) tea.Cmd {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return nil
	}

	from := columnOrder[m.SelectedCol]
	toIndex := m.SelectedCol + delta
	if toIndex < 0 {
		m.Status = fmt.Sprintf("Ticket already in %s", columnOrder[0])
		return nil
	}
	if toIndex >= len(columnOrder) {
		m.Status = fmt.Sprintf("Ticket already in %s", columnOrder[len(columnOrder)-1])
		return nil
	}
	to := columnOrder[toIndex]

	if _, err := store.Move(m.Root, stored.Name, from, to); err != nil {
		m.Status = "Move failed: " + err.Error()
		return nil
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return nil
	}

	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	m.SelectedCol = toIndex
	m.ensureColVisible()
	m.SelectedRows[to] = findTicketRow(m.Board.Columns[to], stored.Name)
	m.ensureSelectedVisible(to)
	return m.notify(fmt.Sprintf("Moved %s to %s", stored.Name, to), notifSuccess)
}

func (m Model) selectedTicket() *store.StoredTicket {
	state := columnOrder[m.SelectedCol]
	tickets := m.Board.Columns[state]
	if len(tickets) == 0 {
		return nil
	}
	row := clamp(m.SelectedRows[state], 0, len(tickets)-1)
	return &tickets[row]
}

func findTicketRow(tickets []store.StoredTicket, name string) int {
	for i, stored := range tickets {
		if stored.Name == name {
			return i
		}
	}
	return 0
}

func (m Model) renderPickNext() string {
	result := store.PickNext(m.Board)
	text := "Next: none"
	if result.HasPick && result.NeedsChoice {
		text = fmt.Sprintf("Next: %d tied candidates", len(result.Tied))
	} else if result.HasPick {
		text = fmt.Sprintf("Next: [%s] %s", result.Ticket.Ticket.Priority, result.Ticket.Ticket.Title)
	}
	return lipgloss.NewStyle().
		Width(m.fullWidth()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(bannerStyle.Render(text))
}

func (m Model) renderBoard() string {
	visible := m.visibleColumnCount()
	start := clamp(m.ColScrollOffset, 0, max(0, len(columnOrder)-visible))
	end := min(start+visible, len(columnOrder))
	columns := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, columnOrder[i]))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderHScrollIndicator() string {
	visible := m.visibleColumnCount()
	if visible >= len(columnOrder) {
		return ""
	}
	start := clamp(m.ColScrollOffset, 0, max(0, len(columnOrder)-visible))
	leftHidden := start
	rightHidden := len(columnOrder) - (start + visible)
	var parts []string
	if leftHidden > 0 {
		names := make([]string, leftHidden)
		for i := range leftHidden {
			names[i] = columnOrder[i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("← %s", strings.Join(names, ", ")))
	}
	if rightHidden > 0 {
		names := make([]string, rightHidden)
		for i := range rightHidden {
			names[i] = columnOrder[start+visible+i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("%s →", strings.Join(names, ", ")))
	}
	return mutedStyle.Render(strings.Join(parts, "  ")) + "\n"
}

func (m Model) renderColumn(index int, state store.State) string {
	var b strings.Builder
	header := strings.ToUpper(state.DisplayName())
	if index == m.SelectedCol {
		header = m.colStyle(index).Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n")

	for _, line := range m.renderColumnLines(index, state) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(m.columnWidth()).
		Height(m.boardColumnHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1).
		Render(b.String())
}

func (m Model) renderColumnLines(index int, state store.State) []string {
	tickets := m.Board.Columns[state]
	scroll := clamp(m.ColumnScroll[state], 0, max(0, len(tickets)-1))
	innerWidth := m.columnInnerWidth()
	maxLines := m.columnLineBudget()
	lines := make([]string, 0, maxLines)

	appendLine := func(line string) bool {
		if len(lines) >= maxLines {
			return false
		}
		lines = append(lines, line)
		return true
	}

	if len(tickets) == 0 {
		appendLine(mutedStyle.Render("  empty"))
		return lines
	}

	if scroll > 0 {
		appendLine(mutedStyle.Render(fmt.Sprintf("  ↑ %d above", scroll)))
	}

	selectedRow := m.SelectedRows[state]
	separator := mutedStyle.Render(strings.Repeat("─", innerWidth))
	for row := scroll; row < len(tickets); row++ {
		if row > scroll {
			if !appendLine(separator) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row, row-1, selectedRow)
			}
		}

		for _, line := range m.styledTicketColumnLines(index, state, row, innerWidth) {
			if !appendLine(line) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row, row, selectedRow)
			}
		}
	}

	return lines
}

func (m Model) ticketColumnLines(state store.State, row int, innerWidth int) []string {
	tickets := m.Board.Columns[state]
	if row < 0 || row >= len(tickets) {
		return nil
	}
	stored := tickets[row]
	prefix := "  "
	if m.MultiSelected[state] != nil && m.MultiSelected[state][stored.Name] {
		prefix = "* "
	}
	lines := wrapText(fmt.Sprintf("%s[%s] %s", prefix, stored.Ticket.Priority, stored.Ticket.Title), innerWidth)
	if stored.Ticket.Deadline != nil {
		lines = append(lines, deadlineIndicatorPlain(*stored.Ticket.Deadline, time.Now()))
	}
	return lines
}

func (m Model) styledTicketColumnLines(index int, state store.State, row int, innerWidth int) []string {
	tickets := m.Board.Columns[state]
	if row < 0 || row >= len(tickets) {
		return nil
	}
	stored := tickets[row]
	isFocused := index == m.SelectedCol && row == m.SelectedRows[state]
	isSelected := m.MultiSelected[state] != nil && m.MultiSelected[state][stored.Name]

	var prefix string
	switch {
	case isFocused && isSelected:
		prefix = ">*"
	case isFocused:
		prefix = "> "
	case isSelected:
		prefix = "* "
	default:
		prefix = "  "
	}

	wrapped := wrapText(fmt.Sprintf("%s[%s] %s", prefix, stored.Ticket.Priority, stored.Ticket.Title), innerWidth)
	for i, line := range wrapped {
		switch {
		case isFocused:
			wrapped[i] = m.colStyle(index).Render(line)
		case isSelected:
			wrapped[i] = selectedStyle.Render(line)
		}
	}
	if stored.Ticket.Deadline != nil {
		wrapped = append(wrapped, m.renderDeadlineIndicator(*stored.Ticket.Deadline, time.Now()))
	}
	return wrapped
}

func appendColumnOverflow(lines []string, width int, below int, lastRenderedRow int, selectedRow int) []string {
	if below <= 0 || len(lines) == 0 || lastRenderedRow == selectedRow {
		return lines
	}
	lines[len(lines)-1] = mutedStyle.Render(fmt.Sprintf("  ↓ %d below", below))
	return lines
}

func deadlineIndicatorPlain(deadline time.Time, now time.Time) string {
	return "  SLA " + strings.Repeat("|", len(deadlineBarStates()))
}

func (m Model) renderDeadlineIndicator(deadline time.Time, now time.Time) string {
	activeBars := deadlineBarCount(deadline, now)
	var b strings.Builder
	b.WriteString(mutedStyle.Render("  SLA "))
	b.WriteString(m.renderDeadlineBarSegment(store.StateReady, min(activeBars, 3), 3))
	b.WriteString(m.renderDeadlineBarSegment(store.StateDoing, clamp(activeBars-3, 0, 2), 2))
	b.WriteString(m.renderDeadlineBarSegment(store.StateDone, clamp(activeBars-5, 0, 1), 1))
	return b.String()
}

func (m Model) renderDeadlineBarSegment(state store.State, active int, total int) string {
	active = clamp(active, 0, total)
	var b strings.Builder
	if active > 0 {
		b.WriteString(m.colStyle(stateColIndex(state)).Render(strings.Repeat("|", active)))
	}
	if inactive := total - active; inactive > 0 {
		b.WriteString(mutedStyle.Render(strings.Repeat("|", inactive)))
	}
	return b.String()
}

func deadlineBarStates() []store.State {
	return []store.State{
		store.StateReady,
		store.StateReady,
		store.StateReady,
		store.StateDoing,
		store.StateDoing,
		store.StateDone,
	}
}

func deadlineBarCount(deadline time.Time, now time.Time) int {
	days := daysUntil(deadline, now)
	switch {
	case days <= 0:
		return 6
	case days <= 1:
		return 5
	case days <= 3:
		return 4
	case days <= 7:
		return 3
	case days <= 14:
		return 2
	default:
		return 1
	}
}

func daysUntil(deadline time.Time, now time.Time) int {
	dueDate := dateOnly(deadline.UTC())
	today := dateOnly(now.UTC())
	return int(dueDate.Sub(today).Hours() / 24)
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func (m Model) renderDetail() string {
	stored := m.selectedTicket()
	if stored == nil {
		return "No ticket selected\n\n" + mutedStyle.Render("esc back  q quit") + "\n"
	}

	contentWidth, metadataWidth := m.detailWidths()
	contentInnerWidth := contentWidth - 2
	lines := wrapLines(m.detailLines(), contentInnerWidth)
	visible, above, below := m.visibleDetailLines(lines)
	contentText := strings.Join(visible, "\n")
	if above > 0 {
		contentText = mutedStyle.Render(fitText(fmt.Sprintf("↑ %d lines above", above), contentInnerWidth)) + "\n" + contentText
	}
	if below > 0 {
		contentText += "\n" + mutedStyle.Render(fitText(fmt.Sprintf("↓ %d lines below", below), contentInnerWidth))
	}
	contentText = bannerStyle.Render("CONTENT") + "\n\n" + contentText
	content := lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.detailPanelHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1).
		Render(contentText)
	metadata := lipgloss.NewStyle().
		Width(metadataWidth).
		Height(m.detailPanelHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(wrapLines(strings.Split(m.renderDetailMetadata(*stored), "\n"), metadataWidth-2), "\n"))

	var b strings.Builder
	b.WriteString(bannerStyle.Render(stored.Ticket.Title))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, content, metadata))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("j/k scroll  esc back  q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderDetailMetadata(stored store.StoredTicket) string {
	var b strings.Builder
	b.WriteString(bannerStyle.Render("METADATA"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", stored.Ticket.Title))
	b.WriteString("State: " + m.colStyle(stateColIndex(stored.State)).Render(string(stored.State)) + "\n")
	b.WriteString("Priority: " + priorityStyle(stored.Ticket.Priority).Render(string(stored.Ticket.Priority)) + "\n")
	b.WriteString(fmt.Sprintf("File: %s\n", stored.Name))
	if len(stored.Ticket.ParsedTitle.Labels) > 0 {
		b.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(stored.Ticket.ParsedTitle.Labels, ", ")))
	}
	if stored.Ticket.Deadline != nil {
		b.WriteString(fmt.Sprintf("Deadline: %s\n", stored.Ticket.Deadline.Format(time.DateOnly)))
	} else {
		b.WriteString("Deadline: —\n")
	}
	b.WriteString(fmt.Sprintf("Created: %s\n", stored.Ticket.Created.Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("Updated: %s", stored.Ticket.Updated.Format("2006-01-02 15:04")))
	return b.String()
}

func (m Model) detailWidths() (int, int) {
	width := m.Width
	if width <= 0 {
		width = 96
	}
	metadata := width / 3
	content := width - metadata - 3
	if metadata < 24 {
		metadata = 24
	}
	if content < 40 {
		content = 40
	}
	return content, metadata
}

func (m Model) visibleDetailLines(lines []string) ([]string, int, int) {
	maxBodyLines := m.detailPanelHeight() - 4 // 2 for borders, 2 for "CONTENT\n" header
	if maxBodyLines < 3 {
		maxBodyLines = 3
	}
	start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
	visibleSlots := maxBodyLines
	if start > 0 {
		visibleSlots--
	}
	below := max(0, len(lines)-start-visibleSlots)
	if below > 0 {
		visibleSlots--
	}
	if visibleSlots < 1 {
		visibleSlots = 1
	}
	end := min(len(lines), start+visibleSlots)
	return lines[start:end], start, len(lines) - end
}

func (m Model) detailLines() []string {
	stored := m.selectedTicket()
	if stored == nil {
		return nil
	}
	body := strings.TrimRight(stored.Ticket.Body, "\n")
	if body == "" {
		return []string{mutedStyle.Render("empty body")}
	}
	return strings.Split(body, "\n")
}

func (m Model) boardColumnHeight() int {
	return m.boardColumnInnerHeight() + 2
}

func (m Model) boardColumnInnerHeight() int {
	if m.Height <= 0 {
		return 18
	}
	height := m.Height - 11
	if height < 6 {
		return 6
	}
	return height
}

func (m Model) visibleTicketRows() int {
	rows := m.boardColumnInnerHeight() - 3
	if rows < 1 {
		return 1
	}
	return rows
}

func (m Model) columnLineBudget() int {
	lines := m.boardColumnInnerHeight() - 1 // reserve one line for the column header
	if lines < 1 {
		return 1
	}
	return lines
}

func (m Model) detailPanelHeight() int {
	return m.detailPanelInnerHeight() + 2
}

func (m Model) detailPanelInnerHeight() int {
	if m.Height <= 0 {
		return 18
	}
	height := m.Height - 7
	if height < 6 {
		return 6
	}
	return height
}

func (m Model) fullWidth() int {
	if m.Width <= 0 {
		return 120
	}
	if m.Width < 40 {
		return 40
	}
	return m.Width
}

func (m Model) visibleColumnCount() int {
	if m.Width <= 0 {
		return len(columnOrder)
	}
	count := m.Width / minColumnWidth
	if count < 1 {
		count = 1
	}
	if count > len(columnOrder) {
		count = len(columnOrder)
	}
	return count
}

func (m Model) columnWidth() int {
	visible := m.visibleColumnCount()
	if m.Width <= 0 {
		return 32
	}
	width := (m.Width / visible) - 2
	if width < 20 {
		return 20
	}
	return width
}

func (m Model) columnInnerWidth() int {
	width := m.columnWidth() - 2
	if width < 1 {
		return 1
	}
	return width
}

func (m Model) renderFooter() string {
	line := m.renderFooterSeparator()
	return line + "\n" + mutedStyle.Render(m.footerText()) + "\n"
}

func (m Model) renderFooterSeparator() string {
	line := strings.Repeat("─", m.fullWidth())
	snack := m.renderSnack()
	if snack == "" {
		return mutedStyle.Render(line)
	}
	plainWidth := lipgloss.Width(snack)
	if plainWidth+2 >= m.fullWidth() {
		return snack
	}
	return snack + mutedStyle.Render(strings.Repeat("─", m.fullWidth()-plainWidth))
}

func (m Model) renderWarnings() string {
	if len(m.Board.Warnings) == 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("Warnings: %d malformed ticket(s) skipped", len(m.Board.Warnings))) + "\n"
}

func (m Model) renderStatus() string {
	return m.renderSnack()
}

func (m Model) renderSnack() string {
	if m.notification != nil {
		switch m.notification.kind {
		case notifSuccess:
			return notifSuccessStyle.Render("✓ " + m.notification.text + " ")
		case notifError:
			return notifErrorStyle.Render("✗ " + m.notification.text + " ")
		default:
			return mutedStyle.Render(m.notification.text + " ")
		}
	}
	parts := make([]string, 0, 2)
	if m.Status != "" {
		parts = append(parts, m.Status)
	}
	if len(m.Board.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("Warnings: %d malformed ticket(s) skipped", len(m.Board.Warnings)))
	}
	if len(parts) == 0 {
		return ""
	}
	return mutedStyle.Render(strings.Join(parts, "  •  ") + " ")
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func wrapLines(lines []string, width int) []string {
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapText(line, width)...)
	}
	return wrapped
}

func wrapText(value string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if value == "" {
		return []string{""}
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, 1)
	current := ""
	for _, word := range words {
		for lipgloss.Width(word) > width {
			part, rest := splitToWidth(word, width)
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, part)
			word = rest
		}
		if current == "" {
			current = word
			continue
		}
		candidate := current + " " + word
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitToWidth(value string, width int) (string, string) {
	runes := []rune(value)
	if width >= len(runes) {
		return value, ""
	}
	return string(runes[:width]), string(runes[width:])
}

func fitText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	plain := lipgloss.Width(value)
	if plain <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-1]) + "…"
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func priorityStyle(p ticket.Priority) lipgloss.Style {
	switch p {
	case ticket.PriorityP0:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff5f5f"))
	case ticket.PriorityP1:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffaf5f"))
	case ticket.PriorityP2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#d7d75f"))
	default:
		return mutedStyle
	}
}

func stateColIndex(state store.State) int {
	for i, s := range columnOrder {
		if s == state {
			return i
		}
	}
	return 0
}

func (m Model) colStyle(colIndex int) lipgloss.Style {
	themeIdx := clamp(m.Config.Theme, 0, len(colorThemes)-1)
	colors := colorThemes[themeIdx].colors
	color := colors[clamp(colIndex, 0, len(colors)-1)]
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func (m Model) enterConfig() (tea.Model, tea.Cmd) {
	idx := len(editorPresets) // default to "custom"
	for i, p := range editorPresets {
		if p == m.Config.Editor {
			idx = i
			break
		}
	}
	input := textinput.New()
	input.Placeholder = "editor command"
	input.CharLimit = 200
	m.configField = 0
	m.configEditorIdx = idx
	m.configEditorInput = input
	m.Mode = ViewConfig
	m.Status = ""
	if idx == len(editorPresets) {
		m.configEditorInput.SetValue(m.Config.Editor)
		cmd := m.configEditorInput.Focus()
		return m, cmd
	}
	return m, nil
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.Mode = ViewBoard
			m.Status = ""
			return m, nil
		case "tab", "shift+tab":
			m.configField = (m.configField + 1) % 2
			if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
				cmd := m.configEditorInput.Focus()
				return m, cmd
			}
			m.configEditorInput.Blur()
			return m, nil
		case "h", "left":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme-1, 0, len(colorThemes)-1)
			} else {
				if m.configEditorIdx > 0 {
					m.configEditorIdx--
				}
				m.configEditorInput.Blur()
			}
			return m, nil
		case "l", "right":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme+1, 0, len(colorThemes)-1)
			} else {
				if m.configEditorIdx < len(editorPresets) {
					m.configEditorIdx++
				}
				if m.configEditorIdx == len(editorPresets) {
					cmd := m.configEditorInput.Focus()
					return m, cmd
				}
				m.configEditorInput.Blur()
			}
			return m, nil
		case "enter":
			m.saveConfig()
			m.Mode = ViewBoard
			return m, m.notify("Config saved", notifSuccess)
		default:
			if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
				var cmd tea.Cmd
				m.configEditorInput, cmd = m.configEditorInput.Update(keyMsg)
				return m, cmd
			}
		}
		return m, nil
	}
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		var cmd tea.Cmd
		m.configEditorInput, cmd = m.configEditorInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) saveConfig() {
	if m.configEditorIdx == len(editorPresets) {
		m.Config.Editor = strings.TrimSpace(m.configEditorInput.Value())
	} else {
		m.Config.Editor = editorPresets[m.configEditorIdx]
	}
	_ = store.SaveConfig(m.Root, m.Config)
}

func (m Model) renderConfig() string {
	formWidth := m.fullWidth() - 4
	if formWidth > 64 {
		formWidth = 64
	}
	if formWidth < 44 {
		formWidth = 44
	}

	labelW := 9
	labelStyle := mutedStyle.Width(labelW)
	activeStyle := func(active bool) lipgloss.Style {
		if active {
			return selectedStyle.Width(labelW)
		}
		return labelStyle
	}

	// Editor row
	editorLabel := activeStyle(m.configField == 0).Render("Editor")
	presetNames := make([]string, 0, len(editorPresets)+1)
	for _, p := range editorPresets {
		name := p
		if name == "" {
			name = "$EDITOR"
		}
		presetNames = append(presetNames, name)
	}
	presetNames = append(presetNames, "custom")
	editorParts := make([]string, 0, len(presetNames))
	for i, name := range presetNames {
		if i == m.configEditorIdx {
			if m.configField == 0 {
				editorParts = append(editorParts, selectedStyle.Render("["+name+"]"))
			} else {
				editorParts = append(editorParts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			editorParts = append(editorParts, mutedStyle.Render(name))
		}
	}
	editorRow := editorLabel + strings.Join(editorParts, "  ")

	// Theme row
	themeLabel := activeStyle(m.configField == 1).Render("Theme")
	themeParts := make([]string, 0, len(colorThemes))
	for i, t := range colorThemes {
		if i == m.Config.Theme {
			if m.configField == 1 {
				themeParts = append(themeParts, selectedStyle.Render("["+t.name+"]"))
			} else {
				themeParts = append(themeParts, lipgloss.NewStyle().Bold(true).Render("["+t.name+"]"))
			}
		} else {
			themeParts = append(themeParts, mutedStyle.Render(t.name))
		}
	}
	themeRow := themeLabel + strings.Join(themeParts, "  ")

	var rows []string
	rows = append(rows, editorRow)

	// Show text input when custom is selected
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		inputWidth := formWidth - labelW - 6
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.configEditorInput.Width = inputWidth
		rows = append(rows, "", labelStyle.Render("")+m.configEditorInput.View())
	}

	rows = append(rows, "", themeRow)
	rows = append(rows, "", mutedStyle.Render("tab field  h/l select  enter save  esc cancel"))
	content := strings.Join(rows, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(formWidth).
		Render(content)

	var statusLine string
	if m.Status != "" {
		statusLine = "\n" + mutedStyle.Render(m.Status)
	}

	h := m.Height
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(m.fullWidth(), h, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("Config")+"\n\n"+box+statusLine)
}

func (m Model) renderPostCreateDialog() string {
	name := filepath.Base(m.createPending)

	content := strings.Join([]string{
		mutedStyle.Render("Created:"),
		"",
		selectedStyle.Render(name),
		"",
		"Open in external editor?",
		"",
		selectedStyle.Render("y") + "  open in editor",
		mutedStyle.Render("n") + "  stay in TickCats",
		mutedStyle.Render("d") + "  don't ask again",
	}, "\n")

	formWidth := 48
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 3).
		Width(formWidth).
		Render(content)

	h := m.Height
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(m.fullWidth(), h, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("Ticket Created")+"\n\n"+box)
}
