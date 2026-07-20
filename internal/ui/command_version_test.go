package ui

import (
	"strings"
	"testing"
)

// :version shows the running binary's build info in the status-bar
// toast — the only way to tell WHICH slk you're in from inside the
// TUI (dev build vs an installed release).

func TestVersionCommandShowsBuildInfo(t *testing.T) {
	a := NewApp()
	a.SetBuildInfo("slk dev (abc1234)")

	cmd := executeCommand(a, "version")
	if cmd == nil {
		t.Fatal("expected non-nil cmd (toast clear tick)")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "slk dev (abc1234)") {
		t.Errorf("statusbar should show build info, got %q", out)
	}
}

func TestVersionCommandUnsetBuildInfo(t *testing.T) {
	a := NewApp()

	_ = executeCommand(a, "version")
	if out := a.statusbar.View(120); !strings.Contains(out, "slk (unknown build)") {
		t.Errorf("statusbar should show unknown-build fallback, got %q", out)
	}
}
