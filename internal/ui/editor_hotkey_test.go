// internal/ui/editor_hotkey_test.go
//
// The external-editor round trip is bound to Ctrl+G in insert mode,
// matching Claude Code's binding for the same gesture. The binding
// lives inline in handleInsertMode rather than in keys.go, so nothing
// else pins it — these tests do.
package ui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func insertKey(a *App, r rune, mod tea.KeyMod) tea.Cmd {
	return handleInsertMode(a, tea.KeyPressMsg{Code: r, Mod: mod})
}

// cleanupEditorTemp removes the draft file beginEditorCompose leaves
// behind when the returned cmd is never run.
func cleanupEditorTemp(t *testing.T, a *App) {
	t.Helper()
	if a.editorTempPath != "" {
		os.Remove(a.editorTempPath)
	}
}

func TestInsertMode_CtrlGStartsTheEditorRoundTrip(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.SetMode(ModeInsert)
	a.compose.SetValue("draft text")
	defer cleanupEditorTemp(t, a)

	if cmd := insertKey(a, 'g', tea.ModCtrl); cmd == nil {
		t.Fatal("ctrl+g should return the ExecProcess cmd")
	}
	if a.editorTempPath == "" {
		t.Error("ctrl+g should have written the draft to a temp file")
	}
}

// Ctrl+E is no longer the binding. It must fall through to the compose
// box as an ordinary key rather than opening an editor.
func TestInsertMode_CtrlENoLongerOpensTheEditor(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.SetMode(ModeInsert)
	defer cleanupEditorTemp(t, a)

	insertKey(a, 'e', tea.ModCtrl)
	if a.editorTempPath != "" {
		t.Error("ctrl+e must not start an editor round trip any more")
	}
}

// The thread compose is the other target: with the thread panel
// focused, ctrl+g must seed from that draft, not the channel one.
func TestInsertMode_CtrlGUsesTheFocusedCompose(t *testing.T) {
	a := NewApp()
	a.threadVisible = true
	a.focusedPanel = PanelThread
	a.SetMode(ModeInsert)
	a.compose.SetValue("channel draft")
	a.threadCompose.SetValue("thread draft")
	defer cleanupEditorTemp(t, a)

	if cmd := insertKey(a, 'g', tea.ModCtrl); cmd == nil {
		t.Fatal("ctrl+g should return the ExecProcess cmd")
	}
	data, err := os.ReadFile(a.editorTempPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "thread draft" {
		t.Errorf("temp file = %q, want the thread compose's draft", data)
	}
}

// Ctrl+U still clears the compose — the neighbouring shortcut in the
// same block, cheap insurance that the rebind did not shift anything.
func TestInsertMode_CtrlUStillClearsCompose(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.SetMode(ModeInsert)
	a.compose.SetValue("throw this away")

	insertKey(a, 'u', tea.ModCtrl)
	if got := a.compose.Value(); got != "" {
		t.Errorf("compose = %q, want cleared", got)
	}
}
