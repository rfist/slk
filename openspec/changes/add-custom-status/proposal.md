## Why

slk already receives the user's own custom status in the boot payload and
throws it away, and it offers no way to set one. The Ctrl+S binding is
labeled "set status" but only changes presence and DND, so the app makes a
promise it does not keep. Setting a status with an expiry -- "Lunch, back
in an hour" -- is one of the few daily Slack actions that still forces the
user out of the terminal and into the desktop client.

## What Changes

- **Set a custom status** (emoji + text + optional expiry) for the active
  workspace, from two surfaces that share one action path:
  - the existing Ctrl+S menu, which gains history rows, duration presets,
    a "Set custom status..." entry point, and a "Clear status" row;
  - a new `:status` command for users who would rather type than pick.
- **Show the active custom status** in the status bar, next to the
  presence/DND segment. The initial value comes from the boot payload,
  which already carries `status_text`, `status_emoji`, and
  `status_expiration` but currently has no consumer.
- **Expire automatically.** The expiry is sent to Slack as
  `status_expiration` and enforced server-side; slk additionally clears its
  own display when the expiry passes, so the status bar cannot show a
  status that Slack has already dropped.
- **Remember previously used statuses** in a per-workspace history, ranked
  by recency and frequency, and offer them as the top rows of the Ctrl+S
  menu so a repeat status is a two-keystroke action.
- **Widen the Slack write surface**: `internal/slack.SlackAPI` gains
  `users.profile.set`. That interface documents its omissions deliberately,
  so this is a considered widening, not an oversight.

**Non-Goals.** Status is set for the **active workspace only**, mirroring
how presence and DND already behave; fan-out across workspaces is not in
this change. Setting another user's status, reading other users' statuses
(sidebar/message-author decorations), and status-driven automation
(calendar sync, DND coupling) are all out of scope.

## Capabilities

### New Capabilities

- `custom-status`: setting, clearing, displaying, and expiring the
  authenticated user's own Slack custom status for the active workspace,
  across both the menu and the `:status` command surfaces.
- `custom-status-history`: recording statuses the user has set and
  offering them back for reuse, scoped per workspace.

### Modified Capabilities

None. This repository has no existing specs under `openspec/specs/`; both
capabilities above are introduced by this change.

## Impact

**Code**

- `internal/slack/client.go` -- `SlackAPI` gains
  `SetUserCustomStatusContext`; a `Client.SetUserCustomStatus` wrapper
  joins `SetUserPresence`. One mock to extend
  (`internal/slack/client_test.go`).
- `internal/slack/events.go` -- a `user_change` case is needed for status
  changes made from other clients to reach the TUI. Currently absent.
- `internal/ui/presencemenu/` -- new actions and rows; the menu stops being
  presence-only.
- `internal/ui/` -- a new text-entry mode for composing a status (the
  custom-snooze mode is the closest existing shape), a status-expiry tick
  (the DND tick is the closest existing shape), a new `:status` entry in
  the command registry, and new fields on the per-workspace status cache.
- `internal/ui/statusbar/` -- a new segment.
- `internal/cache/` -- one new table in the schema string plus its
  accessors; `frecent_emoji` is the precedent.
- `cmd/slk/main.go` -- the status-setter callback gains the new actions;
  the boot-time self profile is plumbed into the per-workspace context.

**APIs and dependencies**

- New Slack Web API method: `users.profile.set` (Tier 3 rate limit). No new
  Go dependencies -- slack-go v0.23.0 already implements it and routes it
  through the client's configured HTTP transport, so slk's cookie auth and
  overridden API base URL both apply.

**Risk**

- `users.profile.set` has never been exercised under slk's xoxc session
  token. It is what the official web client uses, so it is expected to
  work, but this is the single assumption that could invalidate the plan
  and it is verified before any UI work begins.
