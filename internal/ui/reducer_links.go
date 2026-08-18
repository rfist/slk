// internal/ui/reducer_links.go
//
// Link-open routing (issue #62 + in-app permalink navigation).
//
// OpenLinkMsg is the single place every link open flows through:
//   - Slack archive permalinks whose subdomain matches the active
//     workspace AND whose channel resolves via ChannelService.Lookup
//     navigate in-app: dispatch ChannelSelectedMsg, then complete via
//     pendingLinkNav once the channel's messages are loaded (select
//     the target ts, or open the thread panel for thread_ts links).
//   - Everything else opens in the OS browser (a.browserOpener).
//
// Completion hooks live in reducer_channels.go (ChannelSelectedMsg
// and MessagesLoadedMsg arms call completePendingLinkNav).
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/slackurl"
	"github.com/gammons/slk/internal/ui/messages"
)

var reduceLinks reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	m, ok := msg.(OpenLinkMsg)
	if !ok {
		return nil, false
	}
	return a.routeLink(m.URL), true
}

// routeLink decides between in-app navigation and the browser.
func (a *App) routeLink(rawURL string) tea.Cmd {
	pl, ok := slackurl.Parse(rawURL)
	if !ok {
		return a.browserOpener(rawURL)
	}
	domain := a.activeWorkspaceDomain()
	if domain == "" || pl.Subdomain != domain {
		return a.browserOpener(rawURL)
	}
	cmd, ok := a.applyLocation(Location{
		TeamID:    ids.TeamID(a.activeTeamID),
		ChannelID: pl.ChannelID,
		MessageTS: pl.MessageTS,
		ThreadTS:  pl.ThreadTS,
	}, false)
	if ok {
		return cmd
	}
	return a.browserOpener(rawURL)
}

// applyLocation is the single applier every in-app jump flows through
// — a permalink, a mark, or a history walk. It records loc as the
// pending navigation and dispatches the channel switch, so
// completePendingLinkNav can finish once the target channel's messages
// land (or immediately when the channel is already active). ok=false
// means loc's channel could not be resolved and nothing was started;
// callers that need a fallback (routeLink's browser opener, a mark-jump
// toast) branch on ok. A nil cmd with ok=true is a completed navigation
// (e.g. SelectByTS selected the message in the already-active channel),
// not a failure.
//
// fromHistory marks the synthesized ChannelSelectedMsg so the reducer
// does not record the walk as a new visit — history walks must not
// grow the stack. It also forces the ChannelSelectedMsg path even when
// the target channel is already active: a walk is a navigation between
// recorded positions, not an in-place jump, so it always goes through
// the channel-switch pipeline (a same-channel walk is reachable in
// production when stale entries between two same-channel entries are
// skipped and dropped). Direct jumps (permalinks, marks) keep the
// in-place completion so re-selecting the current channel does not
// reload it.
func (a *App) applyLocation(loc Location, fromHistory bool) (tea.Cmd, bool) {
	name, chType, found := a.channels.Lookup(loc.ChannelID)
	if !found {
		return nil, false
	}
	a.pendingLinkNav = &loc
	if !fromHistory && string(loc.ChannelID) == a.activeChannelID {
		// Already viewing the channel; the loaded buffer is as good
		// as it gets, so complete authoritatively right now.
		return a.completePendingLinkNav(a.activeChannelID, true), true
	}
	id, n, t := string(loc.ChannelID), name, chType
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: n, Type: t, FromHistory: fromHistory}
	}, true
}

// completePendingLinkNav finishes (or drops) the pending permalink
// navigation for channelID. authoritative=true means "no more message
// data is coming for this channel" — if the target ts still isn't in
// the buffer, dispatch ChannelService.FetchAround to load a history
// window centered on the target instead of waiting.
//
// Called from: applyLocation (already-active channel, authoritative),
// reduceChannels' ChannelSelectedMsg arm (cache render, best-effort),
// and reduceChannels' MessagesLoadedMsg arm (authoritative).
func (a *App) completePendingLinkNav(channelID string, authoritative bool) tea.Cmd {
	p := a.pendingLinkNav
	if p == nil {
		return nil
	}
	if string(p.ChannelID) != channelID {
		// The user navigated somewhere unrelated before the link
		// target finished loading; the pending nav is stale.
		a.pendingLinkNav = nil
		return nil
	}
	if p.MessageTS == "" && p.ThreadTS == "" {
		// Channel-only location: the channel is already (being)
		// opened and there is nothing to select or open on top of it.
		// Permalinks and search hits always carry a message ts, so
		// only history walks produce this — and FetchAround with an
		// empty ts would be a bogus request.
		a.pendingLinkNav = nil
		return nil
	}
	if p.ThreadTS != "" {
		a.pendingLinkNav = nil
		return a.openThreadForPermalink(string(p.ChannelID), string(p.ThreadTS), string(p.MessageTS))
	}
	if a.messagepane.SelectByTS(string(p.MessageTS)) {
		a.pendingLinkNav = nil
		return nil
	}
	if authoritative {
		a.pendingLinkNav = nil
		channels := a.channels
		chID, ts := p.ChannelID, p.MessageTS
		return func() tea.Msg {
			return channels.FetchAround(chID, ts)
		}
	}
	return nil
}

// openThreadForPermalink opens the thread panel for a permalink that
// carried thread_ts. Unlike openThreadForSelectedMessage it does not
// require the parent message to be in the pane buffer (mirrors
// openSelectedThreadCmd, which builds the parent from a summary):
// the parent row is taken from the loaded buffer or the thread cache
// when available, else a minimal stub that the ThreadRepliesLoadedMsg
// handler backfills from cache once the fetch lands.
//
// replyTS is the message inside the thread the location named (the
// reply, or the parent's own ts when the parent row was the target).
// The replies arrive asynchronously via ThreadRepliesLoadedMsg, so the
// target is remembered in pendingThreadReplyTS and applied there,
// where the replies actually exist — selecting now would miss them.
func (a *App) openThreadForPermalink(channelID, threadTS, replyTS string) tea.Cmd {
	parent := messages.MessageItem{TS: threadTS, ThreadTS: threadTS}
	if channelID == a.activeChannelID {
		for _, m := range a.messagepane.Messages() {
			if m.TS == threadTS {
				parent = m
				break
			}
		}
	}
	if parent.Text == "" {
		if cached := a.threads.CacheRead(ids.ChannelID(channelID), ids.ThreadTS(threadTS)); len(cached) > 0 {
			parent = cached[0]
		}
	}

	// Set AFTER openThreadPanel: it clears any pending reply owed by an
	// earlier open, so setting first would be wiped immediately. The
	// returned cmd has not run yet, so the target is in place well
	// before any ThreadRepliesLoadedMsg is reduced.
	cmd := a.openThreadPanel(parent, channelID, threadTS)
	a.pendingThreadReplyTS = replyTS
	return cmd
}
