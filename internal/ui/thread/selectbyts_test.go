// internal/ui/thread/selectbyts_test.go
//
// Tests for SelectByTS, the timestamp-based selection API added so a
// recorded or marked reply opens the thread at the reply instead of at
// the top. Modelled on messages/selectbyts_test.go.
package thread

import (
	"fmt"
	"testing"

	"github.com/gammons/slk/internal/ui/messages"
)

func TestSelectByTS_ExistingReply(t *testing.T) {
	m := New()
	m.SetThread(
		messages.MessageItem{TS: "1.0", ThreadTS: "1.0", Text: "parent"},
		[]messages.MessageItem{
			{TS: "2.0", ThreadTS: "1.0", Text: "first"},
			{TS: "3.0", ThreadTS: "1.0", Text: "second"},
		},
		"C1", "1.0")
	_ = m.View(40, 60)

	if !m.SelectByTS("2.0") {
		t.Fatal("expected SelectByTS to return true")
	}
	if got := m.SelectedReply(); got == nil || got.TS != "2.0" {
		t.Fatalf("selected reply = %+v, want 2.0", got)
	}
}

func TestSelectByTS_ParentRow(t *testing.T) {
	m := New()
	m.SetThread(
		messages.MessageItem{TS: "1.0", ThreadTS: "1.0", Text: "parent"},
		[]messages.MessageItem{{TS: "2.0", ThreadTS: "1.0", Text: "first"}},
		"C1", "1.0")
	_ = m.View(40, 60)

	// The parent row is a selectable row like any reply: a ts equal to
	// the thread's own ts names it.
	if !m.SelectByTS("1.0") {
		t.Fatal("expected the parent ts to select the parent row")
	}
	if got := m.SelectedReply(); got == nil || got.TS != "1.0" {
		t.Fatalf("selected reply = %+v, want the parent 1.0", got)
	}
}

func TestSelectByTS_AbsentTimestamp(t *testing.T) {
	m := New()
	m.SetThread(
		messages.MessageItem{TS: "1.0", ThreadTS: "1.0", Text: "parent"},
		[]messages.MessageItem{{TS: "2.0", ThreadTS: "1.0", Text: "first"}},
		"C1", "1.0")
	_ = m.View(40, 60) // cursor starts on the newest reply (2.0)

	if m.SelectByTS("9.999999") {
		t.Error("expected false for a missing reply ts")
	}
	if got := m.SelectedReply(); got == nil || got.TS != "2.0" {
		t.Fatalf("selection moved on a miss: %+v", got)
	}
	if m.SelectByTS("") {
		t.Error("expected false for empty ts")
	}
}

// selecting a reply outside the visible window re-snaps the viewport
// so the reply is visible.
func TestSelectByTS_ScrollsIntoView(t *testing.T) {
	m := New()
	parent := messages.MessageItem{TS: "1.0", ThreadTS: "1.0", Text: "parent"}
	var replies []messages.MessageItem
	for i := 2; i <= 30; i++ {
		replies = append(replies, messages.MessageItem{TS: fmt.Sprintf("%d.0", i), ThreadTS: "1.0", Text: "reply"})
	}
	m.SetThread(parent, replies, "C1", "1.0")
	_ = m.View(10, 60) // small viewport: the thread overflows

	visible := func() bool {
		return m.selectedStartLine >= m.vp.YOffset() && m.selectedEndLine <= m.vp.YOffset()+m.vp.Height()
	}

	// From the bottom, jump to the first reply: the viewport moves up.
	before := m.vp.YOffset()
	if !m.SelectByTS("2.0") {
		t.Fatal("SelectByTS on the first reply returned false")
	}
	_ = m.View(10, 60)
	if m.vp.YOffset() >= before {
		t.Fatalf("viewport did not move up for the first reply: before=%d after=%d", before, m.vp.YOffset())
	}
	if !visible() {
		t.Fatal("first reply not visible after snap")
	}

	// From the top, jump to the last reply: the viewport moves down.
	before = m.vp.YOffset()
	if !m.SelectByTS("30.0") {
		t.Fatal("SelectByTS on the last reply returned false")
	}
	_ = m.View(10, 60)
	if m.vp.YOffset() <= before {
		t.Fatalf("viewport did not move down for the last reply: before=%d after=%d", before, m.vp.YOffset())
	}
	if !visible() {
		t.Fatal("last reply not visible after snap")
	}
}
