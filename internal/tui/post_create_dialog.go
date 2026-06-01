// post_create_dialog.go renders the "open in editor?" prompt shown immediately
// after a ticket is created. The user can open the file in their configured
// editor (y), dismiss the prompt (n/esc), or permanently suppress it (d).
package tui

import (
	"path/filepath"
	"strings"
)

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

	box := dialogBoxStyle(48, 0).Render(content)
	return m.placeDialog("Ticket Created", box, "", m.safeHeight(24))
}
