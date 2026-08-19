## Context

See `proposal.md` -- Why. The constraints that shape the approach:

- **The data already arrives.** `internal/slack/boot/boot.go` parses
  `status_text`, `status_emoji`, and `status_expiration` into
  `boot.SelfProfile`, and `bootstrap.Result.Self` carries it into
  `connectWorkspace`. Nothing reads it today except a test. Startup
  display is therefore plumbing, not a new fetch.
- **Presence and DND already model exactly this problem** -- a
  per-workspace piece of self-state, set optimistically, echoed over the
  WebSocket, with a time-based local expiry. `internal/ui/presence.go`,
  `internal/ui/presencemenu/`, `internal/ui/mode_presence_snooze.go`,
  and the `DNDTickMsg` chain are all directly reusable shapes. This change
  should look like a sibling of DND, not like a new subsystem.
- **`internal/slack.SlackAPI` is deliberately narrow.** Its doc comment
  names methods that are omitted on purpose (workspace enumeration), so
  additions are a decision to be justified, and the single mock in
  `internal/slack/client_test.go` must be extended in step.
- **Auth is xoxc + `d` cookie**, not an app token. Every write goes through
  the cookie-bearing `http.Client` built in `newCookieHTTPClient`
  (`internal/slack/client.go`).
- **The cache is one SQLite DB** with `workspace_id` columns, migrated by
  idempotent `CREATE TABLE IF NOT EXISTS` statements in a single schema
  string in `internal/cache/db.go`.

## Goals / Non-Goals

**Goals:**

- One action path shared by the menu and the `:status` command, so the two
  surfaces cannot drift.
- Expiry enforced by Slack, with slk's local timer responsible only for the
  display -- never for correctness.
- Reuse the DND state/tick/optimistic-write machinery rather than
  introducing a parallel one.
- Keep `SlackAPI` growth to the single method the feature needs.

**Non-Goals:**

- No new overlay framework. The composer is a mode plus a small widget
  package in the existing shape.
- No general profile editing. `users.profile.set` is used for the status
  fields only.
- No reading or rendering of *other* users' statuses. That would touch the
  sidebar, message headers, and the user cache, and is a separate change.

## Decisions

### Expiry is Slack's job; the local tick only owns the display

`users.profile.set` accepts `status_expiration` as a Unix timestamp and
Slack enforces it for every client. slk sends the absolute timestamp and
does **not** schedule an API call to clear the status later.

A local tick still exists, but only to stop *showing* a status whose expiry
has passed -- modeled on the `DNDTickMsg` chain guarded by
`presenceController.ClaimTicker` (`internal/ui/presence.go`). It compares
against the absolute expiry timestamp rather than counting down, so a
suspended machine wakes with a correct display.

*Alternative rejected:* a client-side timer that clears the status via the
API. It fails whenever slk is not running at the expiry moment, which is
precisely the case the feature exists to cover, and it would fight the
server-side expiry when both fire.

### `user_change` gets a handler

`internal/slack/events.go` handles no `user_change` event, so today nothing
would tell slk that a status was set on another device or dropped by the
server. Adding the case is what makes the "reflected from other clients"
and "expired at the server" requirements true rather than approximated.

It is scoped tightly: only events for the authenticated user's own ID are
acted on, and only the status fields are read. Everything else in the
payload is ignored, consistent with how the `boot` package models only what
has a consumer.

*Alternative rejected:* polling `users.profile.get`. More requests, worse
latency, and a Tier 3 budget spent on something the socket already knows.

### Status lives next to presence in the per-workspace state

`workspaceContext` (`cmd/slk/main.go`) already holds `Presence`,
`DNDEnabled`, and `DNDEndTS`; `workspaceStatus` in
`internal/ui/presence.go` mirrors them for the UI. Custom status fields
join both, and `ui.StatusChangeMsg` grows the same three fields.

This keeps one message type carrying all self-state, so a workspace switch
restores presence, DND, and custom status through one existing path
(`presenceController.Handle`) rather than two that must be kept in sync.

*Alternative rejected:* a separate `CustomStatusMsg` and controller. It
duplicates the per-team cache, the workspace-switch restore, and the
optimistic-write reconciliation for no gain.

### The menu and the command share one action struct

Both surfaces produce the same value -- emoji, text, absolute expiry -- and
hand it to the same setter callback. `presencemenu.Result` grows a payload
rather than the action switch in `cmd/slk/main.go` growing two
near-identical arms.

The command parser lives in the command layer and produces that same
payload, so `:status :taco: Lunch 1h` and picking the taco row from the
menu are indistinguishable downstream. This is what makes the specs'
"behaves the same from either surface" scenarios cheap to test: one table
of payloads, one applier.

### Composer is a mode, following the custom-snooze precedent

`ModePresenceCustomSnooze` (`internal/ui/mode_presence_snooze.go`, 51
lines) is the smallest working example of "menu selection opens a
sub-mode that collects input". The status composer follows it, with two
differences: it collects text rather than digits, so it uses a real text
input; and it can hand off to the existing `internal/ui/emojipicker`
package for the emoji field instead of requiring the shortcode be typed.

*Alternative considered:* a `huh` form. The dependency is already present,
but no other overlay in `internal/ui` uses it, and adopting it here would
make this change the odd one out.

### History is a table, not a config file

A `status_history` table joins the schema string in
`internal/cache/db.go`, keyed `(workspace_id, emoji, text)`, carrying
`use_count`, `last_used`, and `duration_seconds`. Ranking reuses the
frecency expression already proven in `internal/cache/frecent.go`:
`use_count / (1 + age_in_days)`.

Per-workspace scoping matches `channel_visits` and `thread_subscriptions`.
Storing the duration on the row is what makes a history pick a single
keystroke -- reapplying "Lunch" should not re-ask how long.

The bound is enforced on write (delete the lowest-ranked rows past the cap
after each insert), so no separate maintenance path exists.

*Alternative rejected:* storing history in `config.toml`. It is
user-editable state that changes several times a day; the config file is
for user intent, and mixing the two means rewriting a hand-maintained file
from the hot path.

### Verify `users.profile.set` under xoxc before building on it

The one assumption that could invalidate the plan. It is checked first, as
a throwaway probe against a live workspace, before the API method, the
menu rows, or the table are written. If it fails, the whole change stops
and the spec needs a different mechanism -- so it must not be discovered
after the UI exists.

## Risks / Trade-offs

- **`users.profile.set` may be rejected for xoxc session tokens.** →
  Verified as task 1, before any other work. Nothing else in the change is
  built until it returns success.
- **Widening `SlackAPI` opens the door to general profile writes.** → The
  wrapper exposed on `Client` is `SetUserCustomStatus`, not a generic
  profile setter, so the narrow surface is preserved one level up even
  though slack-go's method is general.
- **`user_change` payloads are large and mostly unmodeled.** → Only the
  self user's status fields are read; the rest is ignored. Consistent with
  the `boot` package's stated policy of modeling only what has a consumer.
- **Tier 3 rate limit on `users.profile.set`.** → Status changes are
  human-paced and the optimistic write means the UI never blocks on the
  call. No batching or backoff is introduced.
- **The `:status` duration grammar can swallow a trailing word** ("Sprint
  review 1h"). → Accepted deliberately, and mitigated by echoing the parsed
  text and expiry back to the user, so a misparse is visible immediately
  rather than discovered an hour later. The composer is the escape hatch.
- **Emoji rendering in the status bar can misreport width.** The presence
  menu's own comment records that mixed emoji widths broke its border
  drawing, which is why its rows are plain ASCII. → The status-bar segment
  measures with the same helper the rest of the bar uses and truncates on
  the measured width; a status whose emoji cannot be measured falls back to
  showing text only.

## Migration Plan

No migration. The new table is created by the existing idempotent schema
statement on first run, and an absent table simply means an empty history.
Older slk versions ignore the table. Nothing about presence or DND changes
shape, so downgrading loses the feature but corrupts nothing.

## Open Questions

- Whether the default expiry belongs under `[general]` or a new
  `[status]` block in `config.toml`. Either satisfies the spec; the
  decision can follow the shape of whatever neighboring keys exist when the
  config task is picked up.
