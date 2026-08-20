// internal/ui/gotop_chord_test.go
//
// `gg` top-jump chord tests. The first `g` arms a pending sub-state in
// normal mode; only a second `g` acts. Anything else — Esc included —
// cancels silently and is swallowed; any mode change disarms (SetMode).
// Mirrors windows_chord_test.go, which covers the same shape for ctrl+w.
package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/sidebar"
)

// sidebarApp returns a test App focused on a seeded sidebar, with the
// cursor parked on the last row so a top-jump is observable.
func sidebarApp(t *testing.T) *App {
	t.Helper()
	a := newWideTestApp(t)
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "C2", Name: "random", Type: "channel"},
		{ID: "C3", Name: "eng-widgets", Type: "channel"},
	})
	a.focusedPanel = PanelSidebar
	a.sidebar.GoToBottom()
	if a.sidebar.SelectedID() == "C1" {
		t.Fatal("setup: cursor should start away from the top row")
	}
	return a
}

func TestChord_FirstGArmsPendingState(t *testing.T) {
	a := sidebarApp(t)
	_ = press(a, 'g')
	if !a.pendingGoTop {
		t.Fatal("first g should arm the pending top-jump state")
	}
	// The first g must not move anything on its own — that is the
	// whole difference between `g` and `gg`.
	if a.sidebar.SelectedID() == "C1" {
		t.Fatal("a single g must not jump; it only arms the chord")
	}
}

// The top row of the sidebar is the Threads entry, which is the whole
// point of the chord: it is the fast path to Threads without going
// through the channel finder.
func TestChord_GGJumpsSidebarToThreadsRow(t *testing.T) {
	a := sidebarApp(t)
	_ = press(a, 'g')
	_ = press(a, 'g')
	if a.pendingGoTop {
		t.Fatal("second g should disarm the pending state")
	}
	if !a.sidebar.IsThreadsSelected() {
		t.Fatalf("gg should select the top row (Threads); SelectedID = %q", a.sidebar.SelectedID())
	}
}

// From a middle row, not just from the bottom — GoToTop is absolute,
// not a one-step move.
func TestChord_GGJumpsFromMiddleRow(t *testing.T) {
	a := sidebarApp(t)
	a.sidebar.SelectByID("C2")
	_ = press(a, 'g')
	_ = press(a, 'g')
	if !a.sidebar.IsThreadsSelected() {
		t.Fatalf("gg from C2 should select the top row; SelectedID = %q", a.sidebar.SelectedID())
	}
}

func TestChord_GThenOtherKeyCancelsSilently(t *testing.T) {
	a := sidebarApp(t)
	before := a.sidebar.SelectedID()
	_ = press(a, 'g')
	_ = press(a, 'x')
	if a.pendingGoTop {
		t.Fatal("an unmapped key should disarm the pending state")
	}
	if got := a.sidebar.SelectedID(); got != before {
		t.Fatalf("SelectedID = %q, want %q — g+x must not jump", got, before)
	}
}

// The cancelling key is swallowed, not re-dispatched: `g` is a prefix,
// so `gj` must not also move down the way a bare `j` would.
func TestChord_CancellingKeyIsSwallowed(t *testing.T) {
	a := sidebarApp(t)
	a.sidebar.GoToTop()
	before := a.sidebar.SelectedID()
	_ = press(a, 'g')
	_ = press(a, 'j')
	if got := a.sidebar.SelectedID(); got != before {
		t.Fatalf("SelectedID = %q, want %q — the key after g is consumed", got, before)
	}
}

func TestChord_EscCancelsGoTop(t *testing.T) {
	a := sidebarApp(t)
	before := a.sidebar.SelectedID()
	_ = press(a, 'g')
	_ = handleNormalMode(a, tea.KeyPressMsg{Code: tea.KeyEscape})
	if a.pendingGoTop {
		t.Fatal("Esc should disarm the pending state")
	}
	if got := a.sidebar.SelectedID(); got != before {
		t.Fatalf("SelectedID = %q, want %q — Esc must not jump", got, before)
	}
}

// A global intercept (ctrl+c quit-confirm) changes mode without going
// through handleGoTopChord. Left armed, the next key in normal mode
// would be swallowed as the chord's second half.
func TestChord_ModeChangeDisarmsGoTop(t *testing.T) {
	a := sidebarApp(t)
	_ = press(a, 'g')
	a.SetMode(ModeInsert)
	if a.pendingGoTop {
		t.Fatal("a mode change must disarm the pending top-jump state")
	}
}

func TestGoToTop_MessagePane(t *testing.T) {
	a := newWideTestApp(t)
	a.focusedPanel = PanelMessages
	// No messages loaded: the assertion is that the chord routes to
	// the pane and returns cleanly rather than panicking on an empty
	// buffer, which is the same contract handleGoToBottom has.
	_ = press(a, 'g')
	if cmd := press(a, 'g'); cmd != nil {
		t.Fatal("gg on the message pane should not emit a command")
	}
	if a.pendingGoTop {
		t.Fatal("gg should disarm the pending state")
	}
}
