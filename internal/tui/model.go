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

var columnOrder = []store.State{store.StateBacklog, store.StateReady, store.StateDoing, store.StateDone}

var (
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bannerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
)

type ViewMode int

const (
	ViewBoard ViewMode = iota
	ViewDetail
	ViewCreate
)

type InteractionMode int

const (
	InteractionBoard InteractionMode = iota
	InteractionMove
	InteractionDeleteConfirm
	InteractionPostCreate
	InteractionSortPrompt
)

var createKinds = []ticket.Kind{ticket.KindFeature, ticket.KindTask, ticket.KindBug}
var createPriorities = []ticket.Priority{ticket.PriorityP0, ticket.PriorityP1, ticket.PriorityP2, ticket.PriorityP3}

type Model struct {
	Root            string
	Board           store.Board
	SelectedCol     int
	SelectedRows    map[store.State]int
	ColumnScroll    map[store.State]int
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

	SortMode    store.SortMode
	ManualOrder map[store.State][]string

	watchCh <-chan struct{}
}

func NewModel(board store.Board) Model {
	return NewModelWithRoot(".", board)
}

func NewModelWithRoot(root string, board store.Board) Model {
	m := Model{
		Root:         root,
		Board:        board,
		SelectedRows: make(map[store.State]int),
		ColumnScroll: make(map[store.State]int),
	}
	cfg, _ := store.LoadSortConfig(root)
	m.SortMode = cfg.Mode
	m.ManualOrder = cfg.ManualOrder
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

func (m Model) Init() tea.Cmd {
	return waitForWatchEvent(m.watchCh)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Mode == ViewCreate {
		return m.updateCreate(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	case msgFileChanged:
		m.reloadBoard()
		return m, waitForWatchEvent(m.watchCh)
	case editorFinishedMsg:
		m.handleEditorFinished(msg)
		return m, nil
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
	case "enter", "o":
		if m.selectedTicket() != nil {
			m.Mode = ViewDetail
			m.DetailScroll = 0
		}
	case "m":
		m.InteractionMode = InteractionMove
		m.Status = "Move mode: h left, l right, esc cancel"
	case "n":
		return m.enterCreate()
	case "p":
		m.promoteToReady()
	case "e":
		return m.editSelected()
	case "x":
		m.enterDeleteConfirm()
	case "r":
		m.reloadBoard()
	case "s":
		m.cycleSortMode()
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
		m.moveSelectedLeft()
	case "l", "right":
		m.moveSelectedRight()
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
		m.deleteSelected()
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
	}
	return m, nil
}

func (m Model) View() string {
	if m.Mode == ViewCreate {
		return m.renderCreate()
	}
	if m.Mode == ViewDetail {
		return m.renderDetail()
	}

	var b strings.Builder
	b.WriteString(m.renderPickNext())
	b.WriteString("\n")
	b.WriteString(m.renderBoard())
	b.WriteString("\n")
	b.WriteString(m.renderWarnings())
	b.WriteString(m.renderStatus())
	b.WriteString(m.renderFooter())
	return b.String()
}

func (m Model) footerText() string {
	if m.InteractionMode == InteractionPostCreate {
		return "Open in editor? y yes  n/esc skip  q quit"
	}
	if m.InteractionMode == InteractionDeleteConfirm {
		return "DELETE? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionSortPrompt {
		return "Switch to manual sort? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionMove {
		return "MOVE MODE: h left  l right  j/k reorder (manual)  esc board  q quit"
	}
	return fmt.Sprintf("BOARD MODE: h/l col  j/k ticket  m move  s sort(%s)  p ready  o/enter detail  e edit  n new  x del  r reload  q quit", m.SortMode)
}

func (m *Model) moveColumn(delta int) {
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(columnOrder)-1)
	m.ensureSelectedVisible(columnOrder[m.SelectedCol])
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
		m.ColumnScroll[state] = 0
		return
	}
	visible := m.visibleTicketRows()
	selected := clamp(m.SelectedRows[state], 0, rows-1)
	scroll := clamp(m.ColumnScroll[state], 0, max(0, rows-visible))
	if selected < scroll {
		scroll = selected
	}
	if selected >= scroll+visible {
		scroll = selected - visible + 1
	}
	m.ColumnScroll[state] = clamp(scroll, 0, max(0, rows-visible))
}

func (m *Model) moveDetailScroll(delta int) {
	maxScroll := len(m.detailLines()) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.DetailScroll = clamp(m.DetailScroll+delta, 0, maxScroll)
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

func (m *Model) deleteSelected() {
	stored := m.selectedTicket()
	if stored == nil {
		m.InteractionMode = InteractionBoard
		m.Status = "No ticket selected"
		return
	}

	if _, err := store.Trash(m.Root, stored.Name, stored.State); err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Delete failed: " + err.Error()
		return
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Reload failed: " + err.Error()
		return
	}

	m.Board = board
	m.InteractionMode = InteractionBoard
	m.Status = fmt.Sprintf("Deleted %s", stored.Name)
}

func (m *Model) promoteToReady() {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return
	}
	if stored.State == store.StateReady {
		m.Status = fmt.Sprintf("%s is already in ready", stored.Name)
		return
	}

	if _, err := store.Move(m.Root, stored.Name, stored.State, store.StateReady); err != nil {
		m.Status = "Move failed: " + err.Error()
		return
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return
	}

	m.Board = board
	readyIdx := 1 // columnOrder index for StateReady
	m.SelectedCol = readyIdx
	m.SelectedRows[store.StateReady] = findTicketRow(m.Board.Columns[store.StateReady], stored.Name)
	m.ensureSelectedVisible(store.StateReady)
	m.Status = fmt.Sprintf("Moved %s to ready", stored.Name)
}

func (m Model) editSelected() (tea.Model, tea.Cmd) {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return m, nil
	}

	cmd := editorCommand(stored.Path)
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

func (m *Model) handleEditorFinished(msg editorFinishedMsg) {
	if msg.err != nil {
		m.Status = "Edit failed: " + msg.err.Error()
		return
	}
	m.reloadBoard()
	m.Status = "Edited ticket"
}

func (m *Model) reloadBoard() {
	state := columnOrder[m.SelectedCol]
	focusedName := ""
	if tickets := m.Board.Columns[state]; m.SelectedRows[state] < len(tickets) {
		focusedName = tickets[m.SelectedRows[state]].Name
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()

	if focusedName == "" {
		return
	}
	newTickets := m.Board.Columns[state]
	for i, t := range newTickets {
		if t.Name == focusedName {
			m.SelectedRows[state] = i
			m.ensureSelectedVisible(state)
			return
		}
	}
	if m.SelectedRows[state] >= len(newTickets) && len(newTickets) > 0 {
		m.SelectedRows[state] = len(newTickets) - 1
	}
	m.ensureSelectedVisible(state)
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
	m.Mode = ViewBoard
	m.InteractionMode = InteractionPostCreate
	m.createPending = path
	m.Status = filepath.Base(path) + " created. Open in editor? y/n"
	return m, nil
}

func (m Model) updatePostCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "y":
		cmd := editorCommand(m.createPending)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		m.InteractionMode = InteractionBoard
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	case "n", "esc":
		m.InteractionMode = InteractionBoard
		m.Status = "Created " + filepath.Base(m.createPending)
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

func (m *Model) moveSelectedRight() {
	m.moveSelected(1)
}

func (m *Model) moveSelectedLeft() {
	m.moveSelected(-1)
}

func (m *Model) moveSelected(delta int) {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return
	}

	from := columnOrder[m.SelectedCol]
	toIndex := m.SelectedCol + delta
	if toIndex < 0 {
		m.Status = "Ticket already in backlog"
		return
	}
	if toIndex >= len(columnOrder) {
		m.Status = "Ticket already done"
		return
	}
	to := columnOrder[toIndex]

	if _, err := store.Move(m.Root, stored.Name, from, to); err != nil {
		m.Status = "Move failed: " + err.Error()
		return
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return
	}

	m.Board = board
	m.SelectedCol = toIndex
	m.SelectedRows[to] = findTicketRow(m.Board.Columns[to], stored.Name)
	m.ensureSelectedVisible(to)
	m.Status = fmt.Sprintf("Moved %s to %s", stored.Name, to)
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
	columns := make([]string, 0, len(columnOrder))
	for i, state := range columnOrder {
		columns = append(columns, m.renderColumn(i, state))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderColumn(index int, state store.State) string {
	var b strings.Builder
	header := strings.ToUpper(string(state))
	if index == m.SelectedCol {
		header = selectedStyle.Render(header)
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
	visibleRows := m.visibleTicketRows()
	scroll := clamp(m.ColumnScroll[state], 0, max(0, len(tickets)-visibleRows))
	innerWidth := m.columnInnerWidth()
	maxLines := m.boardColumnInnerHeight() - 1
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
	end := min(len(tickets), scroll+visibleRows)
	separator := mutedStyle.Render(strings.Repeat("─", innerWidth))
	for row := scroll; row < end; row++ {
		stored := tickets[row]
		prefix := "  "
		if index == m.SelectedCol && row == selectedRow {
			prefix = "> "
		}
		wrapped := wrapText(fmt.Sprintf("%s[%s] %s", prefix, stored.Ticket.Priority, stored.Ticket.Title), innerWidth)
		for _, line := range wrapped {
			if index == m.SelectedCol && row == selectedRow {
				line = selectedStyle.Render(line)
			}
			if !appendLine(line) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row)
			}
		}
		if row < end-1 {
			if !appendLine(separator) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row-1)
			}
		}
	}

	below := len(tickets) - end
	if below > 0 {
		appendColumnOverflow(lines, innerWidth, below)
	}
	return lines
}

func appendColumnOverflow(lines []string, width int, below int) []string {
	if below <= 0 || len(lines) == 0 {
		return lines
	}
	lines[len(lines)-1] = mutedStyle.Render(fmt.Sprintf("  ↓ %d below", below))
	return lines
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
	b.WriteString("Metadata\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", stored.Ticket.Title))
	b.WriteString(fmt.Sprintf("State: %s\n", stored.State))
	b.WriteString(fmt.Sprintf("Priority: %s\n", stored.Ticket.Priority))
	b.WriteString(fmt.Sprintf("File: %s\n", stored.Name))
	if len(stored.Ticket.ParsedTitle.Labels) > 0 {
		b.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(stored.Ticket.ParsedTitle.Labels, ", ")))
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
	maxBodyLines := m.detailPanelHeight() - 2
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

func (m Model) columnWidth() int {
	if m.Width <= 0 {
		return 32
	}
	width := (m.Width / len(columnOrder)) - 2
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
	line := strings.Repeat("─", m.fullWidth())
	return mutedStyle.Render(line) + "\n" + mutedStyle.Render(m.footerText()) + "\n"
}

func (m Model) renderWarnings() string {
	if len(m.Board.Warnings) == 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("Warnings: %d malformed ticket(s) skipped", len(m.Board.Warnings))) + "\n"
}

func (m Model) renderStatus() string {
	if m.Status == "" {
		return ""
	}
	return mutedStyle.Render(m.Status) + "\n"
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
