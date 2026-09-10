package ui

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/cache"
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

func TestThreadMarkedLocalMsg_SuccessAppliesCursor(t *testing.T) {
	app := NewApp()
	app.threadsView.SetSummaries([]cache.ThreadSummary{
		{ChannelID: "C1", ThreadTS: "P1", LastReplyTS: "5.000000", Unread: true},
	})

	_, _ = app.Update(ThreadMarkedLocalMsg{ChannelID: "C1", ThreadTS: "P1", TS: "5.000000"})

	for _, s := range app.threadsView.Summaries() {
		if s.ThreadTS == "P1" && s.Unread {
			t.Error("a successful thread mark must clear the unread flag")
		}
	}
}

func TestThreadMarkedLocalMsg_FailureLeavesFlagAlone(t *testing.T) {
	app := NewApp()
	app.threadsView.SetSummaries([]cache.ThreadSummary{
		{ChannelID: "C1", ThreadTS: "P1", LastReplyTS: "5.000000", Unread: true},
	})

	_, _ = app.Update(ThreadMarkedLocalMsg{
		ChannelID: "C1", ThreadTS: "P1", TS: "5.000000",
		Err: errors.New("subscriptions.thread.mark: thread_not_found"),
	})

	for _, s := range app.threadsView.Summaries() {
		if s.ThreadTS == "P1" && !s.Unread {
			t.Error("a rejected thread mark must leave the thread unread")
		}
	}
}

// markSentinelMsg stands in for the ThreadMarkedLocalMsg the production
// Mark cmd yields, so the test can prove the reducer actually propagated
// the cmd rather than merely calling Mark for its side effect.
type markSentinelMsg struct{}

// The cmd returned by Mark is the entire delivery mechanism for this
// task: it is what persists the read cursor and hands
// ThreadMarkedLocalMsg back to the reducer. If the reducer drops it,
// MarkThread still fires but the cursor is never written locally and
// slk regresses to the stale-cursor bug. Asserting only that Mark was
// called cannot see that, so this asserts the cmd reaches the caller
// and yields the sentinel.
func TestThreadRepliesLoaded_ReturnsMarkCmd(t *testing.T) {
	app := NewApp()
	var got []string
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		Mark: func(channelID ids.ChannelID, threadTS ids.ThreadTS, ts ids.MessageTS) tea.Cmd {
			got = append(got, string(channelID)+"/"+string(threadTS)+"/"+string(ts))
			return func() tea.Msg { return markSentinelMsg{} }
		},
	}))
	app.threadVisible = true
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")

	_, cmd := app.Update(ThreadRepliesLoadedMsg{
		ThreadTS: "P1",
		Replies:  []messages.MessageItem{{TS: "R1"}, {TS: "R5"}},
	})

	if len(got) != 1 || got[0] != "C1/P1/R5" {
		t.Fatalf("Mark calls = %v, want one call marking up to the newest reply", got)
	}
	if cmd == nil {
		t.Fatal("Update returned no cmd; the mark cmd was dropped, so the read cursor would never be persisted")
	}
	var found bool
	for _, m := range drainBatch(cmd) {
		if _, ok := m.(markSentinelMsg); ok {
			found = true
		}
	}
	if !found {
		t.Fatal("the cmd returned by Mark did not reach the caller; ThreadMarkedLocalMsg would never reach the reducer")
	}
}
