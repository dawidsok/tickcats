// editor.go builds the exec.Cmd used to open a ticket file in an external
// editor. The editor is resolved in order: user config → $EDITOR env var → vi.
package tui

import (
	"os"
	"os/exec"
)

func editorCommand(path, preferred string) *exec.Cmd {
	editor := preferred
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	return exec.Command(editor, path)
}
