## Context

See `proposal.md` -- Why. The constraints that shape the approach:

- **The hard part is already built.** `internal/ui/reducer_links.go`
  contains a complete cold-jump pipeline for permalinks: `routeLink`
  parses the target, sets `pendingLinkNav`, and
  `completePendingLinkNav` finishes once the channel's messages land --
  selecting the timestamp, falling back to
  `ChannelService.FetchAround` (`internal/ui/services.go:382`) when the
  message is outside the loaded window, opening the thread panel via
  `openThreadForPermalink` when a thread is named, and dropping the
  navigation as stale if the user moves elsewhere mid-flight. A mark
  jump needs exactly this.
- **The address type already exists too.**
  `slackurl.Permalink` (`internal/slackurl/slackurl.go:19`) is
  `{Subdomain, ChannelID, MessageTS, ThreadTS}` -- workspace, channel,
  message, thread. That is the mark referent and, after this change,
  the navigation-history entry.
- **The navigation history is channel-only and clears state too
  early.** `internal/ui/navhistory.go` stores `[]string` of channel
  IDs. Its `Push` is called at `internal/ui/reducer_channels.go:357`,
  *after* `a.CloseThread()` and `a.clearSelections()` at lines 340-341
  have already discarded the outgoing selection. Recording position is
  therefore a call-site ordering change, not only a struct change.
- **`m`, `'`, and backtick are unbound** in `handleNormalMode`
  (`internal/ui/mode_normal.go`). The doc comment at line 18 claims
  `M (mark unread)` but the actual binding is `U`
  (`internal/ui/keys.go:41`); the comment is stale, not a conflict.
- **A pending-key chord pattern already exists.** `Ctrl+W` sets
  `a.pendingWinCmd` (`mode_normal.go:93`), shows a status-bar hint via
  `statusbar.SetHelpHint`, and dispatches the next key through
  `handleWindowChord` (`internal/ui/windows.go:21`), where unmapped
  keys -- Esc included -- cancel silently.
- **The thread panel cannot select by timestamp.**
  `internal/ui/messages/model.go:842` has `SelectByTS`;
  `internal/ui/thread/model.go` has `SelectByIndex` and `HasReply` but
  no timestamp equivalent.
- **The cache is one SQLite DB** with `workspace_id` columns, migrated
  by idempotent `CREATE TABLE IF NOT EXISTS` statements in a single
  schema string in `internal/cache/db.go`. `channel_visits` and
  `thread_subscriptions` are the per-workspace precedents.
- **Config is TOML with top-level blocks** in
  `internal/config/config.go`; `General`, `Appearance`, `Sidebar` and
  friends are sibling structs on `Config`.

## Goals / Non-Goals

**Goals:**

- One location type and one applier shared by mark jumps, permalink
  jumps, and history walks, so the three cannot drift.
- The navigation-history upgrade lands first and stands on its own: it
  is valuable and verifiable without any mark existing.
- Reuse `pendingLinkNav` rather than growing a second in-flight
  navigation mechanism beside it.
- Leave cross-workspace jumps buildable later without a schema
  migration or re-marking.

**Non-Goals:**

- No new navigation pipeline. If a jump cannot be expressed as "set a
  pending navigation, complete it when messages land", the design is
  wrong.
- No change to how channels are fetched, cached, or rendered. This
  change only decides *where to land*.
- No general-purpose bookmark manager: marks are 52 letters per
  workspace, addressed by keystroke, with no naming, tagging, or
  ordering.

## Decisions

### One location type, extracted from the permalink struct

A single `Location{TeamID, ChannelID, MessageTS, ThreadTS}` becomes the
currency of both capabilities. It is deliberately the same shape as
`slackurl.Permalink` (`internal/slackurl/slackurl.go:19`) with
`Subdomain` replaced by the team ID, because the subdomain is a
rendering detail of Slack URLs while the team ID is what
`workspaceContext` and every cache table key on.

`pendingLinkNav` (`internal/ui/reducer_links.go:27`) collapses into
this type, and `routeLink` becomes "parse a permalink into a
`Location`, then hand it to the applier". Mark jumps and history walks
enter at the same point.

*Alternative rejected:* keeping `pendingLinkNav` private to links and
giving marks their own in-flight struct. Two structures tracking "a
navigation that finishes when messages land" would need the same
staleness rule, the same `FetchAround` fallback, and the same thread
handoff, maintained twice.

### Navigation history updates the departure position, captured before teardown

`navStack.entries` becomes `[]Location`. Two things must both be true,
and conflating them is the trap:

1. The entry that gets **pushed** is still the channel being *opened*,
   exactly as today. The current location must always be present in the
   stack at the cursor, or `Ctrl+H` walks from a cursor that sits one
   behind and skips a channel.
2. The position recorded for the channel being *left* **updates the
   existing entry** at the cursor rather than becoming a new one.

So the arm does: capture the departing location, `UpdateCurrent` the
entry at the cursor with it, then `Push` the arriving channel.

The capture has to happen at the *top* of the `ChannelSelectedMsg` arm
in `internal/ui/reducer_channels.go`, reading
`a.messagepane.SelectedMessage()` (`internal/ui/messages/model.go:818`)
and the thread panel's `ThreadTS()` (`internal/ui/thread/model.go:451`)
before `a.CloseThread()` (line 340) and `a.clearSelections()` (line
341) discard them. The existing push at line 357 runs after both.

This is what makes "back" mean browser-back rather than
"bookmark-of-first-arrival". It is the most subtle part of the change
and has two distinct ways to go wrong: reading the selection after the
teardown (yielding an empty position that still looks like a valid
channel-level entry), and pushing the departure as the entry (silently
breaking back/forward while every position assertion still passes).

The acceptance check for both: the existing navhistory tests in
`internal/ui/app_test.go` assert channel sequence and cursor, and this
change alters neither. **They must stay green, unmodified.** Only the
position data carried inside each entry is new.

*Alternative rejected:* recording arrival position and updating it on
every selection change. Selection changes on every `j`/`k`; the history
would be written continuously rather than once per navigation.

*Alternative rejected:* pushing the departing location as the entry.
It reads as the natural implementation of "record where you left", and
it is wrong: the location you currently occupy is then never in the
stack, so the cursor trails by one and `Ctrl+H` moves two places back
instead of one.

### The back-jump is its own slot, not a navigation-history entry

Vim has two reversal mechanisms and slk needs both, because they answer
different questions. The jumplist (`Ctrl+O`/`Ctrl+I`, slk's
`Ctrl+H`/`Ctrl+K`) answers "where was I before, over the last N moves".
The `'` pseudo-mark answers "put me back where I just was", and it is a
single slot overwritten on every jump.

slk's navigation history is **channel-granular**: `navHistoryStore.Push`
dedupes on `ChannelID`, so the stack holds at most one entry per
channel at the cursor. That is the right model for back/forward through
visited channels, and it is what the existing tests pin. It cannot
represent two positions within one channel, so it cannot on its own make
a same-channel mark jump reversible.

The back-jump is therefore a separate one-`Location` slot on `App`,
written immediately before every mark jump and before every back-jump.
It is independent of the navigation history, which continues to record
cross-channel mark jumps exactly as it records any other channel change.
Writing the slot on the back-jump itself is what makes repeated
back-jumps toggle between two locations, matching vim.

*Alternative rejected:* dropping the channel-identity dedupe so the
history could hold two entries for one channel. Every re-selection of
the current channel at a different position would then grow the stack,
which is the noise the dedupe exists to prevent.

*Alternative rejected:* making the whole navigation history
position-granular, like vim's real jumplist. That is a larger and
genuinely useful change -- it would also let `Ctrl+H` retrace within a
channel after search hits -- but it redesigns machinery this change has
already verified, for a benefit the single slot delivers.

### Marks reuse the history's degradation rules, but never self-delete

`navHistoryStore.Walk` drops entries whose channel no longer resolves,
which is right for an anonymous positional entry. A mark is *named* --
the user typed the letter and expects it to stay typed -- so a mark
whose target is unreachable is retained and reported, never removed.
The message-level case is already the behaviour
`completePendingLinkNav` produces (channel opens, "Message not found"
toast); marks inherit it unchanged.

*Alternative rejected:* auto-pruning dead marks on startup. It would
require resolving every persisted mark at boot -- a network round trip
per mark, on the startup path, for a problem the user can fix with one
`:delmarks` invocation.

### Marks store a rendered preview alongside the address

Each mark carries a snapshot taken at mark time: channel name, author
display name, and a truncated excerpt of the message text. The overlay
renders from the snapshot and never fetches.

Without it, a persisted uppercase mark is `(channel_id, ts)` after a
restart -- the overlay would show opaque IDs, or would have to fetch 52
messages to draw a list. The snapshot can go stale if the message is
edited; the spec accepts that explicitly, because the *jump* still
resolves live and the preview exists only for recognition.

*Alternative rejected:* resolving previews from the message cache on
open. It works while the message is still cached and fails silently
once retention evicts it, producing a list that degrades with exactly
the marks the user has held longest.

### Two storage tiers behind one accessor

Lowercase marks live in an in-memory per-workspace map; uppercase marks
live in a `marks` table in the schema string in
`internal/cache/db.go`, keyed `(workspace_id, letter)` and carrying the
location plus the preview snapshot. The `[marks] persist_all` option
routes lowercase writes to the table as well.

The UI reads through one accessor that merges both tiers, so no caller
knows which tier a letter came from. Turning `persist_all` off does not
delete previously persisted lowercase rows -- it stops loading them --
so toggling the option is not destructive.

*Alternative rejected:* persisting everything and filtering lowercase
on load. The distinction would then exist only at read time, and a
session mark's whole point is that it does not outlive the session.

### Every mark records its workspace, though jumps assert it

`Location.TeamID` is written for every mark from the first
implementation, and the jump path refuses a mark whose `TeamID` is not
the active workspace with a toast. Cross-workspace navigation does not
exist in slk today -- `routeLink`
(`internal/ui/reducer_links.go:47-49`) sends any permalink whose
subdomain differs from the active workspace to the OS browser -- so
building it here would ship an untestable path.

Recording the team ID now means the future change is purely additive to
the jump path: no schema migration, no re-marking. The refusal message
is the seam, and it is where a later implementer will look.

### The chord and the overlay follow existing precedents

`m` and `'` each arm a pending-key flag on `App` in the shape of
`pendingWinCmd` (`internal/ui/mode_normal.go:42`, `:93`), intercepted
at the top of `handleNormalMode` before the main switch, with a
status-bar hint via `statusbar.SetHelpHint` and silent cancel on any
unmapped key including Esc -- matching `handleWindowChord`
(`internal/ui/windows.go:21`).

The marks overlay is a self-contained widget package under
`internal/ui/marks/`, plus a `ModeMarks` entry in `internal/ui/mode.go`
listed in `IsModalOverlay`.

The overlay convention in this repo is `HandleKey(keyStr string)
*XResult` — a pointer to a package-specific result struct, nil meaning
"not handled / stay open" — rendered through `ViewOverlay(termWidth,
termHeight int, background string) string`. `presencemenu.HandleKey`
(`internal/ui/presencemenu/model.go:172`), `channelfinder.HandleKey`
(`internal/ui/channelfinder/model.go:271`), `reactionpicker` and
`themeswitcher` all follow it.

**Follow `internal/ui/channelfinder/` for structure**: it is a
filterable list of destinations that returns a selected target, which
is what the marks overlay is. `internal/ui/linkpicker/` is the closest
*functional* analogue (a list of targets, each of which navigates) but
it is the one package that does **not** follow the convention — its
`HandleKey` returns `(Item, bool)` rather than a result pointer. Do not
copy linkpicker's signature.

The `'` overlay and the `:marks` overlay are the same package in two
entry modes. On the `'` path a letter key jumps immediately rather than
moving a selection, so the overlay never adds a keystroke to the vim
flow -- it only removes the need to remember. That is why disabling it
(`[marks] show_jump_overlay = false`) changes nothing about how `'a`
behaves.

*Alternative rejected:* a status-bar hint line listing marks instead of
an overlay. 52 possible marks with channel and excerpt do not fit one
line, and truncating to fit would defeat the recognition the list
exists for.

### `:marks` and `:delmarks` join the command registry

Both register in the `commands` map in `internal/ui/command.go`,
matching vim's names. `:delmarks` takes a letter list; the overlay's
delete action calls the same underlying removal, so the two surfaces
cannot diverge.

## Risks / Trade-offs

- **The navigation-history change modifies behaviour that existing
  tests cover.** `Ctrl+H` / `Ctrl+K` are used after search hits,
  permalink jumps, and unread walks. → Sequenced first, with the
  existing `navhistory` and `reducer_channels` tests kept green before
  any mark code is written. A regression here is a regression in
  features that already shipped.
- **Capture-before-teardown is easy to get subtly wrong.** Reading the
  selection after `clearSelections()` yields an empty position that
  still produces a plausible-looking channel-level entry. → The spec's
  "departure position is captured" scenario is written to fail in
  exactly that case: it requires the recorded message to be the
  scrolled-to one, not the default.
- **`FetchAround` can fail to find the target.** The message may be
  outside what Slack returns. → Already handled:
  `internal/ui/reducer_channels.go:170-178` verifies the target is in
  the fetched window before replacing the buffer and toasts "Message
  not found in loaded history" otherwise, leaving the user in place.
  Marks inherit this rather than adding a second failure path.
- **Preview snapshots drift from the live message.** → Accepted and
  specified. The preview is for recognition; the jump resolves live.
- **The `marks` table grows unboundedly across workspaces.** → Bounded
  by construction: 52 letters per workspace, replaced in place on
  overwrite. No eviction logic is needed.
- **Adding `SelectByTS` to the thread panel introduces a second
  selection API that can drift from the messages pane's.** → Modelled
  directly on `internal/ui/messages/model.go:842` with the same
  signature and the same "returns false when absent" contract, so the
  two read identically at the call site.

## Migration Plan

No migration. The `marks` table is created by the existing idempotent
schema statement on first run; an absent table means no persisted
marks. The navigation-history change is in-memory only and
session-scoped, so there is nothing on disk to convert. Downgrading
loses marks and reverts `Ctrl+H` to channel-level behaviour without
corrupting anything.

## Open Questions

- Whether the marks overlay should indicate which letters are *free*
  when opened from `m` rather than `'`. The spec does not require it
  and either choice satisfies it; decide when the overlay is built and
  its row rendering is concrete.
