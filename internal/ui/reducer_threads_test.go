package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
)

// TestApp_WorkspaceReadyAndActivationBothEnsureSubscriptions pins where
// the subscriptions.thread.getView fetch is triggered.
//
// Boot is a trigger because the socket replays nothing: an app that
// was closed (or asleep) for days missed every thread_subscription_
// changed event, and rendering the threads view from the days-old
// SQLite snapshot is the staleness this fetch fixes. Activation stays
// a trigger as a safety net. Collapsing both to one actual network
// sweep per workspace is the main-package gate's job
// (threadSubsGate) — the reducer deliberately fires unconditionally
// and stays dumb about throttling.
func TestApp_WorkspaceReadyAndActivationBothEnsureSubscriptions(t *testing.T) {
	app := NewApp()
	ensured := make(chan string, 4)
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		ListFetch: func(teamID ids.TeamID) tea.Msg {
			return ThreadsListLoadedMsg{TeamID: string(teamID)}
		},
		EnsureSubscriptions: func(teamID ids.TeamID) {
			ensured <- string(teamID)
		},
	}))

	_, cmd := app.Update(WorkspaceReadyMsg{TeamID: "T1", TeamName: "Test", InitialActive: true})
	for _, m := range drainBatch(cmd) {
		_ = m
	}
	select {
	case team := <-ensured:
		if team != "T1" {
			t.Errorf("workspace-ready ensured subscriptions for %q; want T1", team)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("workspace-ready did not ensure subscriptions; a workspace slk never opens the Threads view on stays stale for the whole session")
	}

	app.activeTeamID = "T1"
	_, cmd = app.Update(ThreadsViewActivatedMsg{})
	for _, m := range drainBatch(cmd) {
		_ = m
	}
	select {
	case team := <-ensured:
		if team != "T1" {
			t.Errorf("activation ensured subscriptions for %q; want T1", team)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("opening the Threads view did not ensure subscriptions")
	}
}

// A pending reply target must not survive the thread open that owed it.
//
// The ThreadRepliesLoadedMsg arm returns early when the fetch fails
// (m.Replies == nil) WITHOUT consuming pendingThreadReplyTS. Without a
// reset in openThreadPanel, a later plain open of the same thread would
// inherit that stale target and jump to a reply the user never asked
// for. openThreadPanel clears it; callers wanting a selection set it
// after the call.
func TestThreadOpenClearsStalePendingReply(t *testing.T) {
	app := NewApp()
	app.activeTeamID = "T1"
	app.activeChannelID = "C1"
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		CacheRead: func(ids.ChannelID, ids.ThreadTS) []messages.MessageItem { return nil },
		Fetch:     func(ids.ChannelID, ids.ThreadTS) tea.Msg { return nil },
	}))

	// A thread location naming reply R1 opens the thread, then its
	// replies fetch fails: the arm bails before consuming the target.
	_ = app.openThreadForPermalink("C1", "P1", "R1")
	if app.pendingThreadReplyTS != "R1" {
		t.Fatalf("pending reply = %q, want R1 owed by the permalink open", app.pendingThreadReplyTS)
	}
	app.Update(ThreadRepliesLoadedMsg{ThreadTS: "P1", Replies: nil})
	if app.pendingThreadReplyTS != "R1" {
		t.Fatalf("a failed fetch should leave the target owed; got %q", app.pendingThreadReplyTS)
	}

	// The user now opens the same thread plainly. This open owes
	// nothing, so the stale target must be gone before replies land.
	_ = app.openThreadPanel(messages.MessageItem{TS: "P1", ThreadTS: "P1"}, "C1", "P1")
	if app.pendingThreadReplyTS != "" {
		t.Fatalf("plain thread open inherited a stale pending reply %q", app.pendingThreadReplyTS)
	}

	app.Update(ThreadRepliesLoadedMsg{ThreadTS: "P1", Replies: []messages.MessageItem{
		{TS: "R1", ThreadTS: "P1", Text: "reply 1"},
		{TS: "R2", ThreadTS: "P1", Text: "reply 2"},
	}})
	if got := app.threadPanel.SelectedReply(); got != nil && got.TS == "R1" {
		t.Fatal("plain thread open jumped to the stale target R1")
	}
}

// A jump into a thread must land on the named reply after BOTH
// replies-loads. openThreadPanel batches two: the cache first, then the
// network fetch. Both call SetThread, which resets the selection — so a
// pending target consumed on the cached pass left the fetched pass
// highlighting the wrong reply, which is what the user saw.
func TestThreadReplySelectionSurvivesTheFetchAfterTheCache(t *testing.T) {
	replies := []messages.MessageItem{
		{TS: "R1", ThreadTS: "P1", Text: "first"},
		{TS: "R2", ThreadTS: "P1", Text: "second"},
		{TS: "R3", ThreadTS: "P1", Text: "third"},
	}
	app := NewApp()
	app.activeTeamID = "T1"
	app.activeChannelID = "C1"
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		CacheRead: func(ids.ChannelID, ids.ThreadTS) []messages.MessageItem {
			return append([]messages.MessageItem{{TS: "P1", ThreadTS: "P1", Text: "parent"}}, replies...)
		},
		Fetch: func(ids.ChannelID, ids.ThreadTS) tea.Msg { return nil },
	}))

	// A location naming the middle reply opens the thread.
	_ = app.openThreadForPermalink("C1", "P1", "R2")

	// The cached replies land first.
	app.Update(ThreadRepliesLoadedMsg{ThreadTS: "P1", Replies: replies})
	if got := app.threadPanel.SelectedReply(); got == nil || got.TS != "R2" {
		t.Fatalf("after the cached load, selected = %+v, want R2", got)
	}

	// Then the authoritative fetch lands and replaces the replies.
	app.Update(ThreadRepliesLoadedMsg{ThreadTS: "P1", Replies: replies})
	if got := app.threadPanel.SelectedReply(); got == nil || got.TS != "R2" {
		t.Fatalf("after the fetched load, selected = %+v, want R2 — the selection was lost on the second load", got)
	}
}
