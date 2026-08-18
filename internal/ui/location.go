// internal/ui/location.go
//
// Location is the shared address type for navigable positions in a
// workspace: the currency of the navigation history, permalink jumps,
// and vim-style marks. It is deliberately the same shape as
// slackurl.Permalink (internal/slackurl/slackurl.go) with the
// workspace subdomain replaced by the team ID, because the team ID is
// what workspaceContext and every cache table key on.
package ui

import "github.com/gammons/slk/internal/ids"

// Location identifies a workspace, a channel, and optionally a message
// within that channel and a thread the message belongs to. A location
// that names only a channel remains valid and means "the channel, at
// its default position".
type Location struct {
	// TeamID is the workspace the location belongs to. Every recorded
	// location carries it; jump paths assert it matches the active
	// workspace and refuse otherwise.
	TeamID ids.TeamID
	// ChannelID is the channel the location is in.
	ChannelID ids.ChannelID
	// MessageTS is the target message timestamp; empty when the
	// location names only a channel.
	MessageTS ids.MessageTS
	// ThreadTS is the thread parent ts when the location names a
	// reply inside a thread; empty for channel-level locations. A
	// non-empty ThreadTS means "open the thread panel instead of
	// selecting".
	ThreadTS ids.ThreadTS
}

// currentLocation snapshots the position the user is at right now:
// the active team, the active channel, and the selected message — the
// thread panel's selected reply when the thread panel is focused, the
// messages pane's selection otherwise. ok=false when no message is
// selected (there is nothing to name).
func (a *App) currentLocation() (Location, bool) {
	if a.focusedPanel == PanelThread {
		reply := a.threadPanel.SelectedReply()
		if reply == nil {
			return Location{}, false
		}
		return Location{
			TeamID:    ids.TeamID(a.activeTeamID),
			ChannelID: ids.ChannelID(a.activeChannelID),
			MessageTS: ids.MessageTS(reply.TS),
			ThreadTS:  ids.ThreadTS(a.threadPanel.ThreadTS()),
		}, true
	}
	msg, ok := a.messagepane.SelectedMessage()
	if !ok {
		return Location{}, false
	}
	return Location{
		TeamID:    ids.TeamID(a.activeTeamID),
		ChannelID: ids.ChannelID(a.activeChannelID),
		MessageTS: ids.MessageTS(msg.TS),
	}, true
}
