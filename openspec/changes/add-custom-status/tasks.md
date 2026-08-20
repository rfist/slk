## 1. De-risk the one blocking assumption

- [ ] 1.1 Probe `users.profile.set` against a live workspace using slk's
      existing xoxc token and `d` cookie, setting and then clearing a
      throwaway status. Stop the change here if it is rejected.
- [ ] 1.2 Record the observed response shape and any scope error in
      `devdocs/fyi.md`, including whether `status_expiration` is honoured
      as sent.

## 2. Slack API surface

- [ ] 2.1 Add `SetUserCustomStatusContext` to the `SlackAPI` interface in
      `internal/slack/client.go`, with a comment saying why the write
      surface widened.
- [ ] 2.2 Add a `Client.SetUserCustomStatus(ctx, emoji, text string,
      expiration time.Time)` wrapper beside `SetUserPresence`, translating
      the zero time to "no expiry".
- [ ] 2.3 Extend `mockSlackAPI` in `internal/slack/client_test.go` and
      cover the wrapper: emoji-only, text-only, both empty (clear), and
      expiry translation.

## 3. Self-status state

- [ ] 3.1 Add custom-status fields (emoji, text, expiry) to
      `workspaceContext` in `cmd/slk/main.go`, next to `Presence` /
      `DNDEnabled` / `DNDEndTS`.
- [ ] 3.2 Add the same fields to `ui.StatusChangeMsg`
      (`internal/ui/msgs.go`) and to `workspaceStatus` in
      `internal/ui/presence.go`; extend `presenceController.Set` and the
      workspace-switch restore path.
- [ ] 3.3 Seed the fields at connect time from `bootstrap.Result.Self`'s
      profile, treating an already-past expiry as "no status".
- [ ] 3.4 Tests: per-team cache retains status across a simulated
      workspace switch; a past expiry seeds empty.

## 4. Live status changes from elsewhere

- [ ] 4.1 Add a `user_change` case to `internal/slack/events.go`, acting
      only on events for the authenticated user and reading only the
      status fields.
- [ ] 4.2 Emit `StatusChangeMsg` from the handler in `cmd/slk/main.go`
      alongside the existing `manual_presence_change` / `dnd_updated`
      emitters.
- [ ] 4.3 Tests: a `user_change` for another user is ignored; one for self
      updates status and leaves presence and DND untouched.

## 5. Display

- [ ] 5.1 Add status text/emoji/expiry fields and a setter to
      `internal/ui/statusbar/model.go`; mark the model dirty on change.
- [ ] 5.2 Render the segment next to the presence/DND segment, resolving
      the emoji shortcode through `internal/emoji`, truncating on measured
      width, and falling back to text-only if the emoji cannot be measured.
- [ ] 5.3 Add a status-expiry tick modeled on the `DNDTickMsg` chain,
      guarded by a claim flag like `presenceController.ClaimTicker`, that
      compares against the absolute expiry so a suspended machine wakes
      correct.
- [ ] 5.4 Tests: segment hidden when unset; shown with a glyph when set;
      cleared by the tick once the expiry passes; narrow-terminal
      truncation keeps the connection segment visible.

## 6. Shared action path

- [ ] 6.1 Define one status payload type (emoji, text, absolute expiry) and
      the setter callback that applies it, replacing nothing in the
      existing presence actions.
- [ ] 6.2 Extend `App.SetStatusSetter`'s callback in `cmd/slk/main.go` to
      handle set and clear, applying optimistically and reverting plus
      toasting on failure.
- [ ] 6.3 Tests: a failing setter restores the previously displayed status
      and raises a toast; a disconnected workspace does not display an
      unsent status.

## 7. Menu surface

- [ ] 7.1 Add actions and rows to `internal/ui/presencemenu/model.go`:
      "Set custom status...", duration presets, and "Clear status" shown
      only when a status is set.
- [ ] 7.2 Keep the existing presence and DND rows and their behaviour
      unchanged; extend `OpenWith` to accept the current status.
- [ ] 7.3 Verify the box-size/render agreement test still holds with the
      new rows and that labels stay plain ASCII per the package's existing
      width note.
- [ ] 7.4 Tests: clear row absent when no status; selecting a preset
      produces the expected payload; presence rows still return their
      original actions.

## 8. Composer

- [ ] 8.1 Add a status-composer mode following
      `internal/ui/mode_presence_snooze.go`, collecting emoji, text, and
      expiry, pre-filled with the current status when opened empty.
- [ ] 8.2 Wire the emoji field to `internal/ui/emojipicker`.
- [ ] 8.3 Enforce the 100-character text limit at input time; reject a
      submission with neither emoji nor text and keep the composer open.
- [ ] 8.4 Tests: cancel sends nothing; over-limit input is refused;
      empty-empty submission is refused; commit produces the shared
      payload.

## 9. Command surface

- [ ] 9.1 Register `status` in the `commands` map in
      `internal/ui/command.go`.
- [ ] 9.2 Implement the parser: optional leading `:emoji:`, optional
      trailing duration (`30m`, `1h`, `4h`, `today`, `never`), remainder as
      text; bare `:status` opens the composer; `:status clear` clears.
- [ ] 9.3 Echo the parsed text and expiry back as a toast so a swallowed
      trailing duration is visible.
- [ ] 9.4 Tests: table over the grammar, including `Sprint review 1h`,
      emoji-only, text-only, `clear`, and no arguments.

## 10. History

- [ ] 10.1 Add the `status_history` table to the schema string in
      `internal/cache/db.go`, keyed `(workspace_id, emoji, text)` with
      `use_count`, `last_used`, `duration_seconds`.
- [ ] 10.2 Add `RecordStatusUse`, `GetStatusHistory(workspaceID, limit)`,
      and `DeleteStatusHistoryEntry`, reusing the frecency expression from
      `internal/cache/frecent.go`; enforce the per-workspace cap on write.
- [ ] 10.3 Record on successful set only -- not on failure, not on clear;
      update the remembered duration on repeat.
- [ ] 10.4 Feed history rows into the menu above the presets, filterable by
      the menu's existing query, applying the remembered duration on
      select without opening the composer.
- [ ] 10.5 Add the delete action for a selected history row.
- [ ] 10.6 Tests: dedup on repeat; duration updated on repeat; ordering
      favours recent-and-frequent over frequent-and-old; cap enforced;
      per-workspace isolation; history readable before connect.

## 11. Configuration

- [ ] 11.1 Add the default-expiry setting to `internal/config`, defaulting
      to 1 hour, with "don't clear" expressible. Resolve the design's open
      question about which block it belongs in by matching neighbouring
      keys.
- [ ] 11.2 Apply the default when a surface commits a status without an
      explicit duration.
- [ ] 11.3 Tests: unconfigured yields 1 hour; configured value honoured;
      "don't clear" yields no expiry.

## 12. Finish

- [ ] 12.1 Run `go build ./...`, `go test ./... -race`, `go vet`, and
      `gofmt -l` on changed files only.
- [ ] 12.2 Update the Ctrl+S help text and the keybinding description so
      "set status" describes what the menu now does.
- [ ] 12.3 Document the `:status` grammar and the default-expiry setting in
      the wiki, matching where presence and DND are documented.
- [ ] 12.4 Log the decisions and anything surprising in `devdocs/fyi.md`.
