> Read `design.md` before starting. Every task below names the file to
> change and, where one exists, the existing code to copy the shape from.
> Prefer extending the named precedent over inventing a new pattern.
>
> Do not edit files under `openspec/` while implementing. If a task turns
> out to be wrong, stop and say so rather than adjusting the spec to match
> the code.

## 1. The shared location type

- [ ] 1.1 Add a `Location` struct (team ID, channel ID, message TS,
      thread TS) in `internal/ui/`. Mirror the field set of
      `slackurl.Permalink` (`internal/slackurl/slackurl.go:19`), using
      the team ID in place of `Subdomain`. Use the `internal/ids` typed
      IDs where the surrounding code already does.
- [ ] 1.2 Add a helper that builds a `Location` from the app's current
      state: active team, active channel, `a.messagepane.SelectedMessage()`
      (`internal/ui/messages/model.go:818`) when the messages pane is
      focused, and the thread panel's `ThreadTS()`
      (`internal/ui/thread/model.go:451`) plus selected reply when the
      thread panel is focused. Returns ok=false when no message is
      selected.
- [ ] 1.3 Tests: the helper produces a thread-bearing location when the
      thread panel is focused, a channel-level one otherwise, and
      ok=false with nothing selected.

## 2. One applier for every jump

- [ ] 2.1 Replace `pendingLinkNav` (`internal/ui/reducer_links.go:27`)
      with the `Location` type. Keep the field semantics identical:
      a non-empty thread TS still means "open the thread panel instead
      of selecting".
- [ ] 2.2 Extract the body of `routeLink`
      (`internal/ui/reducer_links.go:42`) after permalink parsing into
      an applier that takes a `Location` and returns the `tea.Cmd`.
      `routeLink` becomes: parse the URL, convert to a `Location`,
      call the applier. Behaviour must not change.
- [ ] 2.3 Leave `completePendingLinkNav` and `openThreadForPermalink`
      working exactly as they do now, including the `FetchAround`
      fallback and the stale-navigation drop.
- [ ] 2.4 Run `go test ./... -race`. The existing permalink navigation
      tests must pass unchanged -- this task group is a refactor with
      no behaviour change. Do not proceed until they are green.

## 3. Navigation history carries position

> This group changes behaviour that search, permalink, and unread
> navigation already depend on. Finish it, get it green, and commit
> before starting group 4.

- [ ] 3.1 Change `navStack.entries` in `internal/ui/navhistory.go` from
      `[]string` to a slice of `Location`. Keep `navStackMax = 50`, the
      forward-path truncation, and the cursor semantics unchanged.
- [ ] 3.2 Change `navHistoryStore.Push` to take a `Location`. Push
      continues to receive the channel being *opened*, exactly as it
      does today. Keep the consecutive-dedupe, but compare on channel
      identity so returning to the same channel at a different position
      is still not a new entry.
- [ ] 3.3 Add `navHistoryStore.UpdateCurrent(teamID string, loc
      Location)` to `internal/ui/navhistory.go`: it overwrites the
      entry at the cursor with `loc`, but only when that entry already
      names the same channel. A no-op when the team has no stack, when
      the cursor is at -1, or when the channels differ.
- [ ] 3.4 In `internal/ui/reducer_channels.go`, capture the departing
      location at the **top** of the `ChannelSelectedMsg` arm --
      before `a.CloseThread()` (line 340) and `a.clearSelections()`
      (line 341) discard it. Call `UpdateCurrent` with it immediately
      before the existing push. **Keep pushing the ARRIVING channel**
      (`m.ID`) as the code does today: the departing position updates
      the entry being left, it does not become a new entry. Pushing the
      departure instead would leave the current location absent from
      the stack, so the cursor would sit one behind and `Ctrl+H` would
      skip a channel.
- [ ] 3.5 Acceptance check for 3.1-3.4: **every existing navhistory
      test in `internal/ui/app_test.go` must still pass, unchanged.**
      They assert channel sequence and cursor, neither of which this
      change alters -- only the position data carried inside each entry
      is new. Failures there mean the approach is wrong, not that the
      tests are stale.
- [ ] 3.6 Change `navHistoryStore.Walk` to return a `Location`. Keep
      the stale-entry skip-and-drop behaviour for entries whose
      *channel* does not resolve via `ChannelLookupFunc`.
- [ ] 3.7 Change `a.navigateBack` / `a.navigateForward`
      (`internal/ui/app.go:744`) to feed the walked `Location` into the
      group-2 applier instead of synthesizing a bare
      `ChannelSelectedMsg`. Keep `FromHistory: true` so the walk does
      not grow the stack.
- [ ] 3.8 In `walkNavCmd` (`internal/ui/app.go`), call `UpdateCurrent`
      with the departing location **before** calling `Walk`. A history
      walk is also a departure, but the `ChannelSelectedMsg` arm cannot
      handle it: `Walk` moves the cursor first, so by the time the
      message is reduced the cursor names the destination and
      `UpdateCurrent`'s channel guard makes it a no-op. Without this,
      walking away from a channel loses the position you were at.
- [ ] 3.9 Tests for 3.8: walk back to a channel, move to a different
      message, walk forward, then walk back again -- the second return
      lands on the message you moved to, not the one stored before the
      first walk.
- [ ] 3.10 A walked location whose message cannot be found must open the
      channel and toast, not drop the entry. The message-level failure
      already exists at `internal/ui/reducer_channels.go:170-178`;
      reuse it rather than adding a second not-found path.
- [ ] 3.11 Tests -- these are the ones that matter most in this change:
      going back after scrolling to an older message restores *that*
      message, not the newest; departing from an open thread records
      the thread and reply and returns to both; an entry whose channel
      no longer resolves is skipped and dropped; walking back and
      forward repeatedly does not grow the stack; per-workspace
      isolation still holds.
- [ ] 3.12 Run `go test ./... -race`, `go vet ./...`, and `gofmt -l` on
      changed files. Commit here: this group is independently valuable
      and independently reviewable.

## 4. Thread panel can select by timestamp

- [ ] 4.1 Add `SelectByTS(ts string) bool` to
      `internal/ui/thread/model.go`, modelled directly on
      `internal/ui/messages/model.go:842` -- same signature, same
      "returns false when the timestamp is absent" contract, same
      scroll-into-view behaviour.
- [ ] 4.2 Call it from the thread-opening path so a location naming a
      reply lands on that reply rather than the top of the thread.
      `openThreadForPermalink` (`internal/ui/reducer_links.go:117`) is
      the entry point.
- [ ] 4.3 Tests: selecting an existing reply returns true and moves the
      selection; an absent timestamp returns false and leaves the
      selection alone; opening a thread at a reply scrolls it into
      view.

## 5. Mark storage

- [ ] 5.1 Add a `marks` table to the schema string in
      `internal/cache/db.go`, keyed `(workspace_id, letter)`, carrying
      the location fields plus the preview snapshot (channel name,
      author display name, message excerpt). Follow the shape of
      `channel_visits` / `thread_subscriptions`.
- [ ] 5.2 Add accessors: upsert a mark, list marks for a workspace,
      delete a mark by letter. Upsert replaces in place -- there is no
      history and no eviction (52 letters per workspace is the bound).
- [ ] 5.3 Add the in-memory session tier: a per-workspace map for
      lowercase marks.
- [ ] 5.4 Add the merged accessor the UI reads through, so no caller
      knows whether a letter came from memory or SQLite. Uppercase
      always persists; lowercase persists only when the config option
      is on.
- [ ] 5.5 Tests: uppercase survives a simulated restart, lowercase does
      not; with persist-all on, lowercase survives; turning persist-all
      off stops loading lowercase rows but does not delete them;
      per-workspace isolation; overwriting a letter replaces it.

## 6. Setting a mark

- [ ] 6.1 Add a pending-key flag for `m` on `App`, following
      `pendingWinCmd` (`internal/ui/app.go:145`). Intercept it at the
      top of `handleNormalMode` (`internal/ui/mode_normal.go:42`),
      before the main switch, exactly as `pendingWinCmd` is.
- [ ] 6.2 Show a status-bar hint while armed via
      `statusbar.SetHelpHint` and restore `a.defaultHelpHint()` when
      consumed, matching `mode_normal.go:42-45` and `:93-95`.
- [ ] 6.3 Any key that is not `a-z` or `A-Z` cancels silently, Esc
      included -- match `handleWindowChord`
      (`internal/ui/windows.go:21`), which documents this as vim
      behaviour.
- [ ] 6.4 On a letter, build the location via the group-1 helper,
      capture the preview snapshot, and store it. With no message
      selected, record nothing and toast.
- [ ] 6.5 Bind `m` in `internal/ui/keys.go` with a help entry. Note the
      stale doc comment at `internal/ui/mode_normal.go:18` claiming
      `M (mark unread)` -- the real binding is `U`; fix the comment
      while you are there.
- [ ] 6.6 Tests: marking a channel message records channel + message;
      marking a thread reply records channel + thread + reply;
      overwriting replaces; a non-letter key cancels and records
      nothing; nothing selected records nothing and toasts.

## 7. Jumping to a mark

- [ ] 7.1 Add the pending-key flag for `'` in the same shape as group
      6, plus backtick as an alias.
- [ ] 7.2 On a letter, look up the mark and hand its `Location` to the
      group-2 applier. Before jumping, push the current location onto
      the navigation history so back returns the user to where they
      were.
- [ ] 7.3 An unset letter changes nothing and toasts.
- [ ] 7.4 A mark whose team ID is not the active workspace does not
      jump, toasts that cross-workspace jumps are unsupported, and is
      retained. This refusal is the documented seam for a future
      cross-workspace change -- comment it as such.
- [ ] 7.5 A mark whose message cannot be found opens the channel,
      toasts, and is retained. A mark whose channel does not resolve
      leaves the user in place, toasts, and is retained. Marks are
      never deleted automatically.
- [ ] 7.6 Tests: jump into another channel selects the message; jump
      within the current channel does not reload; jump to a thread
      reply opens the thread and selects the reply; jump to a message
      outside loaded history triggers the surrounding-history load;
      unset letter is a no-op with a toast; back after a jump returns
      to the pre-jump position; a foreign-workspace mark is refused and
      retained; a dead mark survives a failed jump.

## 8. The marks overlay

- [ ] 8.1 Add `internal/ui/marks/` as a self-contained widget package
      following the repo's overlay convention: `HandleKey(keyStr
      string) *MarksResult` (pointer, nil = not handled / stay open)
      and `ViewOverlay(termWidth, termHeight int, background string)
      string`. Copy the structure from
      `internal/ui/channelfinder/model.go:271` — a filterable list that
      returns a selected target. Do **not** copy
      `internal/ui/linkpicker/`: it is functionally similar but is the
      one overlay that breaks the convention, returning `(Item, bool)`.
- [ ] 8.2 Add `ModeMarks` to `internal/ui/mode.go`, include it in
      `IsModalOverlay`, and give it a `String()` label.
- [ ] 8.3 Render one row per mark: letter, channel, message preview,
      all from the stored snapshot. No fetching -- the overlay must
      draw correctly offline and immediately after a restart.
- [ ] 8.4 Selecting a row closes the overlay and jumps to that mark via
      the group-7 path.
- [ ] 8.5 Add the delete action on the selected row, calling the same
      removal the `:delmarks` command uses.
- [ ] 8.6 Empty state: report that no marks are set.
- [ ] 8.7 Open the overlay when `'` is armed, unless the config option
      disables it. While it is open, pressing a mark letter jumps
      immediately rather than moving the selection -- the overlay must
      not add a keystroke to `'a`.
- [ ] 8.8 Tests: rows render from the snapshot with no service calls;
      a letter press while open jumps directly; selecting a row jumps;
      delete removes the row and leaves the rest; empty state renders;
      the overlay is suppressed on `'` when disabled.

## 9. Commands

- [ ] 9.1 Register `marks` in the `commands` map in
      `internal/ui/command.go`. It always opens the overlay,
      regardless of the jump-overlay config option.
- [ ] 9.2 Register `delmarks`, taking one or more letters. Deleting an
      unset letter is harmless.
- [ ] 9.3 Tests: `:marks` opens the overlay with the option off;
      `:delmarks a` removes one; multiple letters remove several;
      an unset letter is a no-op; a deleted uppercase mark stays gone
      across a simulated restart.

## 10. Configuration

- [ ] 10.1 Add a `Marks` struct to `internal/config/config.go` as a
      `[marks]` block beside `General` / `Appearance` / `Sidebar`,
      with the persist-all option (default off) and the jump-overlay
      option (default on).
- [ ] 10.2 Wire both options through to the storage tier and the `'`
      handler.
- [ ] 10.3 Tests: unset config yields lowercase-temporary and
      overlay-on; each option honoured when set.

## 11. Finish

- [ ] 11.1 Run `go build ./...`, `go test ./... -race`, `go vet ./...`,
      and `gofmt -l` on changed files only.
- [ ] 11.2 Add the new keys to the help overlay so `?` lists `m`, `'`,
      `:marks`, and `:delmarks`.
- [ ] 11.3 Document the marks keys, the two commands, and the `[marks]`
      config block in the wiki, matching where the window commands and
      presence are documented.
- [ ] 11.4 Log the decisions and anything surprising in
      `devdocs/fyi.md`.
