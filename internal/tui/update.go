// update.go contains the top-level Update dispatcher and the View function.
// Update routes each incoming message to the correct mode-specific handler,
// with global keys (ctrl+c, ?, q) intercepted before mode dispatch.
// View selects which render function to call based on the current Mode and
// InteractionMode.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Mode == ViewCreate {
		return m.updateCreate(msg)
	}
	if m.Mode == ViewConfig {
		return m.updateConfig(msg)
	}
	if m.InteractionMode == InteractionSearch {
		return m.updateSearch(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global exits — always honoured outside create/config views.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "?" && m.InteractionMode != InteractionHelp && m.InteractionMode != InteractionQuitConfirm {
			return m.enterHelp()
		}
		if msg.String() == "q" && m.InteractionMode != InteractionQuitConfirm {
			return m.enterQuitConfirm()
		}
		if m.InteractionMode == InteractionQuitConfirm {
			return m.updateQuitConfirm(msg)
		}
		if m.InteractionMode == InteractionHelp {
			return m.updateHelp(msg)
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
		if stored := m.selectedTicket(); stored != nil {
			m.detailTicketName = stored.Name
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
	case "/":
		return m.enterSearch()
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
		m.resolveDetailCursor()
		m.detailTicketName = ""
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
	if m.InteractionMode == InteractionHelp {
		return m.renderHelpDialog()
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
	if m.InteractionMode == InteractionSearch {
		b.WriteString(m.renderSearchBar())
		b.WriteString("\n")
	}
	b.WriteString(m.renderHScrollIndicator())
	b.WriteString(m.renderBoard())
	b.WriteString("\n")
	b.WriteString(m.renderFooter())
	return b.String()
}
