// internal/ui/editor_alias_test.go
//
// Fork-only: Ctrl+G is an alias for upstream's Ctrl+E external-editor
// binding, matching Claude Code's key for the same gesture. The alias
// lives inline in handleInsertMode, so nothing else pins it.
package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestApp_CtrlGOpensEditorFromInsertMode(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.SetMode(ModeInsert)
	a.focusedPanel = PanelMessages
	a.composeEditor = []string{"true"} // Cmd is never invoked
	_ = a.compose.Focus()
	a.compose.SetValue("draft before editor")

	_, cmd := a.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("want a non-nil Cmd from Ctrl+G in insert mode")
	}
	if !a.compose.EditingExternally() {
		t.Fatal("want compose locked while the editor is open")
	}
}

func TestApp_CtrlGUsesTheFocusedThreadCompose(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.SetMode(ModeInsert)
	a.threadVisible = true
	a.focusedPanel = PanelThread
	a.composeEditor = []string{"true"}
	_ = a.threadCompose.Focus()

	if cmd := a.handleKey(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}); cmd == nil {
		t.Fatal("want a non-nil Cmd from Ctrl+G in thread insert mode")
	}
	if !a.threadCompose.EditingExternally() {
		t.Error("want the thread compose locked")
	}
	if a.compose.EditingExternally() {
		t.Error("channel compose must stay unlocked when the thread compose opened the editor")
	}
}
