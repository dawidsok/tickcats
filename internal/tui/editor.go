// editor.go builds the exec.Cmd used to open a ticket file in an external
// editor. The editor is resolved in order: user config → $EDITOR env var → vi.
package tui

import (
	"os"
	"os/exec"
	"strings"
)

func editorCommand(path, preferred string) *exec.Cmd {
	editor := preferred
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	bin, err := exec.LookPath(parts[0])
	if err != nil {
		bin = parts[0]
	}
	args := append(parts[1:], path)
	return exec.Command(bin, args...)
}
