## Why

In a busy workspace the message you need is the one you cannot find
again. A thread you are half-following scrolls out of reach; a decision
posted in a popular channel is buried under an hour of chatter. slk can
already *reach* any message -- permalink navigation, search, and
`FetchAround` all land on an exact timestamp -- but it offers no way to
*name* a location and come back to it.

The same gap shows up in the navigation history slk already has.
`Ctrl+H` / `Ctrl+K` walk back and forward through visited channels, but
they store channel IDs only, so going back returns you to the right
channel at the wrong place. Every jump in the app -- search hit,
permalink, unread walk -- loses your position the moment you leave.

## What Changes

- **Position-aware navigation history.** The per-workspace back/forward
  stack stores a full location (channel, message, optional thread)
  instead of a bare channel ID, and captures it at the moment you
  *leave* a channel rather than when you arrived. `Ctrl+H` starts
  behaving like browser back: same channel, same place. This benefits
  every existing jump, not just marks.
- **Vim-style marks.** `m{a-z}` and `m{A-Z}` record the current
  location under a letter; `'{letter}` jumps to it. Lowercase marks
  live for the session, uppercase marks persist across restarts,
  matching vim's buffer-local / global split. A config option makes all
  marks persistent.
- **Marks address messages and thread replies, not just channels.**
  A mark taken with the thread panel focused records the reply inside
  the thread, so returning to "that message in the 200-reply thread" is
  two keystrokes. This is the case the feature exists for; a
  channel-only mark would add nothing over `Ctrl+T`.
- **A marks overlay.** Pressing `'` opens a list of the marks that
  exist and where they point, so the letters do not have to be
  memorised. `:marks` opens the same overlay explicitly. The
  press-`'` overlay can be disabled in config for users who want plain
  vim behaviour; `:marks` always shows it.
- **Deleting marks.** `:delmarks {letters}` and a delete action on a
  selected overlay row.
- **One address type shared by both features.** Marks and the
  navigation stack converge on a single location struct and a single
  "go to this location" applier, reusing the permalink navigation
  pipeline (`routeLink` / `pendingLinkNav` / `completePendingLinkNav`)
  that already handles channel switching, history fetching, thread
  opening, and the not-found case.

**Non-Goals.** Marks are **scoped to one workspace**. Every stored mark
records its workspace ID so a future change can add cross-workspace
jumps without a migration or re-marking, but the jump path in this
change asserts the active workspace and refuses otherwise. slk has no
in-app cross-workspace navigation today -- `routeLink` sends
other-workspace permalinks to the browser -- so building that path here
would mean shipping behaviour that cannot be exercised.

Also out of scope: marks in the sidebar (channel-only marks, which
`Ctrl+T` already covers), vim's numbered and automatic marks beyond the
single back-jump position, marking a window/split layout, and sharing
marks between machines.

## Capabilities

### New Capabilities

- `navigation-history`: the per-workspace back/forward stack --
  what it records, when it records it, how it degrades when a recorded
  location no longer exists, and how `Ctrl+H` / `Ctrl+K` walk it.
  Covers both the existing channel-level behaviour and the position
  upgrade, since the current behaviour has never been specified.
- `marks`: setting, jumping to, listing, and deleting named locations;
  the session/persistent split; the overlay; the configuration
  surface.

### Modified Capabilities

None. This repository has no existing specs under `openspec/specs/`;
both capabilities above are introduced by this change.

## Impact

**Code**

- `internal/ui/navhistory.go` -- `navStack.entries` becomes a slice of
  location structs; `Push` gains the departing position; `Walk`'s
  stale-entry FSM learns to degrade a dead message to its channel
  rather than dropping the entry.
- `internal/ui/reducer_channels.go` -- the `ChannelSelectedMsg` arm
  must capture the outgoing position *before* `CloseThread()` and
  `clearSelections()` tear it down (currently `navHistory.Push` runs
  after both, at line 357).
- `internal/ui/reducer_links.go` -- `pendingLinkNav` and the
  `routeLink` / `completePendingLinkNav` pipeline are generalised so a
  mark jump and a permalink jump share one applier.
- `internal/ui/thread/model.go` -- gains `SelectByTS`, which the
  messages pane already has (`internal/ui/messages/model.go:842`) and
  the thread panel does not. Without it a mark inside a thread opens
  the thread but lands at the top.
- `internal/ui/` -- a pending-key sub-state for `m` and `'` following
  the `pendingWinCmd` / `handleWindowChord` precedent
  (`mode_normal.go:42`, `windows.go:21`); a new mode and overlay
  package for the marks list; `:marks` and `:delmarks` entries in the
  command registry (`command.go`).
- `internal/ui/keys.go` -- `m`, `'`, and backtick are currently
  unbound in normal mode.
- `internal/cache/` -- one new table in the schema string plus its
  accessors, keyed by workspace like `channel_visits` and
  `thread_subscriptions`.
- `internal/config/` -- a `[marks]` block.

**APIs and dependencies**

None. No Slack API method is added, no new Go dependency. Every jump
target is reached through machinery that already exists; persistence
uses the SQLite cache already open.

**Risk**

The navigation-history change modifies behaviour that search and
permalink navigation already depend on and that existing tests cover.
It is sequenced first and verified on its own so a regression surfaces
before marks are built on top of it.
