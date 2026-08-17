// internal/ui/editor.go
//
// External-editor compose: Ctrl+E in insert mode suspends the TUI,
// opens $VISUAL/$EDITOR on a temp file seeded with the current draft,
// and on a clean exit replaces the draft with the file's contents.
//
// The temp file lives in os.TempDir and is removed as soon as the
// editor returns (success or failure) so drafts don't accumulate on
// disk. A non-zero editor exit (e.g. vim's :cq) aborts the round-trip
// and leaves the in-app draft untouched.
package ui

import (
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// editorFinishedMsg is delivered by the ExecProcess callback when the
// external editor exits. Path is the temp file to read; Panel records
// which compose (channel or thread) initiated the round-trip so the
// result lands in the right draft even if focus changed meanwhile.
type editorFinishedMsg struct {
	Path  string
	Panel Panel
	Err   error
}

// editorCommand returns the external editor to launch: $VISUAL, then
// $EDITOR, then vi (present on every POSIX system).
func editorCommand() string {
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}

// beginEditorCompose writes the focused compose's draft to a temp
// file and returns the ExecProcess cmd that suspends the TUI and
// launches the editor on it. Returns nil if the temp file can't be
// created (the draft is never lost — it stays in the textarea).
func (a *App) beginEditorCompose() tea.Cmd {
	panel := PanelMessages
	target := &a.compose
	if a.focusedPanel == PanelThread && a.threadVisible {
		panel = PanelThread
		target = &a.threadCompose
	}

	f, err := os.CreateTemp("", "slk-compose-*.md")
	if err != nil {
		return toastWithClear(a, "Editor failed: "+err.Error(), 3*time.Second)
	}
	path := f.Name()
	_, werr := f.WriteString(target.Value())
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(path)
		return toastWithClear(a, "Editor failed: could not write draft", 3*time.Second)
	}
	a.editorTempPath = path

	c := exec.Command(editorCommand(), path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{Path: path, Panel: panel, Err: err}
	})
}

// applyEditorResult finishes the round-trip: on a clean editor exit
// the temp file's contents (minus the trailing newline editors append
// on save) replace the recorded compose's draft. The temp file is
// always removed, error or not.
func (a *App) applyEditorResult(m editorFinishedMsg) {
	defer func() {
		os.Remove(m.Path)
		a.editorTempPath = ""
	}()
	if m.Err != nil {
		return
	}
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return
	}
	text := strings.TrimRight(string(data), "\n")

	target := &a.compose
	if m.Panel == PanelThread {
		target = &a.threadCompose
	}
	target.SetValue(text)
}
