package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorCommandPrefersVisualThenEditor(t *testing.T) {
	t.Setenv("VISUAL", "myvisual")
	t.Setenv("EDITOR", "myeditor")
	if got := editorCommand(); got != "myvisual" {
		t.Errorf("editorCommand = %q, want myvisual", got)
	}
	t.Setenv("VISUAL", "")
	if got := editorCommand(); got != "myeditor" {
		t.Errorf("editorCommand = %q, want myeditor", got)
	}
	t.Setenv("EDITOR", "")
	if got := editorCommand(); got != "vi" {
		t.Errorf("editorCommand = %q, want vi fallback", got)
	}
}

// applyEditorResult reads the temp file the editor wrote, replaces the
// recorded compose's draft, and removes the file. A trailing newline
// (added by virtually every editor on save) is stripped so sends don't
// carry a phantom blank line.
func TestApplyEditorResultUpdatesComposeAndCleansUp(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages

	path := filepath.Join(t.TempDir(), "slk-compose.md")
	if err := os.WriteFile(path, []byte("edited draft\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	a.applyEditorResult(editorFinishedMsg{Path: path, Panel: PanelMessages})

	if got := a.compose.Value(); got != "edited draft" {
		t.Errorf("compose value = %q, want %q", got, "edited draft")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file should be removed, stat err = %v", err)
	}
}

func TestApplyEditorResultThreadPanel(t *testing.T) {
	a := NewApp()

	path := filepath.Join(t.TempDir(), "slk-compose.md")
	if err := os.WriteFile(path, []byte("thread reply"), 0o600); err != nil {
		t.Fatal(err)
	}

	a.applyEditorResult(editorFinishedMsg{Path: path, Panel: PanelThread})

	if got := a.threadCompose.Value(); got != "thread reply" {
		t.Errorf("threadCompose value = %q, want %q", got, "thread reply")
	}
}

// Editor exited non-zero (e.g. :cq) — draft untouched, file removed.
func TestApplyEditorResultErrorLeavesDraft(t *testing.T) {
	a := NewApp()
	a.compose.SetValue("original")

	path := filepath.Join(t.TempDir(), "slk-compose.md")
	if err := os.WriteFile(path, []byte("should not apply"), 0o600); err != nil {
		t.Fatal(err)
	}

	a.applyEditorResult(editorFinishedMsg{Path: path, Panel: PanelMessages, Err: os.ErrClosed})

	if got := a.compose.Value(); got != "original" {
		t.Errorf("compose value = %q, want untouched %q", got, "original")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file should be removed even on error, stat err = %v", err)
	}
}

// beginEditorCompose seeds the temp file with the current draft so the
// editor opens with what the user already typed.
func TestBeginEditorComposeSeedsDraft(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.compose.SetValue("half-typed message")

	cmd := a.beginEditorCompose()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	if a.editorTempPath == "" {
		t.Fatal("expected editorTempPath recorded")
	}
	defer os.Remove(a.editorTempPath)
	data, err := os.ReadFile(a.editorTempPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "half-typed message") {
		t.Errorf("temp file = %q, want seeded draft", data)
	}
}
