package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/statusbar"
)

// y in Normal mode (message selected) copies the message's text to the
// clipboard as plain text: mention/link/entity tokens are flattened the
// same way search snippets are, so `<@U1>` becomes @alice and
// `<mailto:a@b.com|a@b.com>` becomes a@b.com — what a user expects to
// paste elsewhere. Clipboard delivery goes through OSC 52 (#142), so the
// write is a tea.Cmd batched with the success toast.

func TestYankTextOfSelectedCopiesFlattenedText(t *testing.T) {
	app := NewApp()
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{{
		TS:   "100.0",
		Text: "contact <@U1> at <mailto:john@example.com|john@example.com>",
	}})
	app.SetUserNames(map[string]string{"U1": "alice"})

	var gotText string
	app.SetClipboardWriter(func(text string) tea.Cmd {
		gotText = text
		return nil
	})

	msgs := drainBatch(app.yankTextOfSelected())

	want := "contact @alice at john@example.com"
	if gotText != want {
		t.Errorf("clipboard = %q, want %q", gotText, want)
	}
	found := false
	for _, m := range msgs {
		if _, ok := m.(statusbar.TextCopiedMsg); ok {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TextCopiedMsg toast, got %+v", msgs)
	}
}

func TestYankTextNoSelectionIsNoop(t *testing.T) {
	app := NewApp()
	app.focusedPanel = PanelMessages

	if cmd := app.yankTextOfSelected(); cmd != nil {
		t.Errorf("expected nil cmd with no messages, got %v", drainBatch(cmd))
	}
}
