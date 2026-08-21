// internal/ui/editor_tty_test.go
//
// The Ctrl+G editor must be handed the real terminal. tea.ExecProcess
// fills any nil stream with the Program's own input/output, and slk's
// output is the sixel FrameOutput wrapper rather than an *os.File — so
// os/exec would give the child a pipe instead of the tty. See
// editorExecCmd for the full reasoning.
package ui

import (
	"os"
	"testing"
)

func TestEditorExecCmd_BindsRealTerminalFiles(t *testing.T) {
	c := editorExecCmd("/tmp/draft.md")

	// *os.File specifically: os/exec only passes a real fd through when
	// the field's concrete type is *os.File. An io.Writer that merely
	// exposes Fd() still gets a pipe.
	for _, tc := range []struct {
		name string
		got  any
		want *os.File
	}{
		{"Stdin", c.Stdin, os.Stdin},
		{"Stdout", c.Stdout, os.Stdout},
		{"Stderr", c.Stderr, os.Stderr},
	} {
		f, ok := tc.got.(*os.File)
		if !ok {
			t.Errorf("%s is %T, want *os.File — ExecProcess would substitute a pipe", tc.name, tc.got)
			continue
		}
		if f != tc.want {
			t.Errorf("%s = %v, want the process's own %s", tc.name, f.Name(), tc.name)
		}
	}
}

func TestEditorExecCmd_PassesThePathToTheEditor(t *testing.T) {
	c := editorExecCmd("/tmp/draft.md")
	if n := len(c.Args); n < 2 || c.Args[n-1] != "/tmp/draft.md" {
		t.Errorf("Args = %v, want the draft path last", c.Args)
	}
}
