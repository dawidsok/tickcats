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
