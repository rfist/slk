## Purpose

Lets the user name a message -- in a channel or deep inside a thread --
with a single letter and return to it later in two keystrokes, so an
important message in a busy channel stops being something that has to be
found again from scratch.

## ADDED Requirements

### Requirement: Setting a mark

The user SHALL be able to record the current location under a single
letter. Letters `a` through `z` and `A` through `Z` SHALL each be usable
as a mark name, giving 52 distinct marks per workspace. Setting a mark
whose letter is already in use SHALL replace the previous location
without confirmation, matching vim.

#### Scenario: Marking a channel message

- **WHEN** the user sets mark `a` while a message in the channel
  timeline is selected
- **THEN** mark `a` records that channel and that message

#### Scenario: Marking a reply inside a thread

- **WHEN** the user sets mark `a` while a reply in an open thread is
  selected
- **THEN** mark `a` records that thread, that reply, and **the channel
  the thread belongs to** -- which is not necessarily the channel most
  recently opened, since a thread may be read from a list spanning
  several channels

#### Scenario: Overwriting an existing mark

- **WHEN** the user sets mark `a` while mark `a` already points
  somewhere else
- **THEN** mark `a` points at the new location
- **AND** the previous location is discarded without a prompt

#### Scenario: Marking with no message selected

- **WHEN** the user sets a mark while no message is selected
- **THEN** the app SHALL decline to record a mark
- **AND** SHALL tell the user why

#### Scenario: Abandoning the letter

- **WHEN** the user begins setting a mark and then presses a key that is
  not a mark letter
- **THEN** no mark is recorded and the app returns to its previous state
  without an error

### Requirement: Jumping to a mark

The user SHALL be able to jump to the location recorded under a letter.
The jump SHALL open the channel if it is not already open, load
surrounding history if the message is not currently loaded, open the
thread panel if the mark names a thread, and select the marked message.

#### Scenario: Jump to a mark in another channel

- **WHEN** the user jumps to a mark naming a message in a channel that
  is not currently open
- **THEN** that channel opens and the marked message is selected and
  visible

#### Scenario: Jump to a mark in the current channel

- **WHEN** the user jumps to a mark naming a message in the channel
  already open
- **THEN** the marked message is selected without reloading the channel

#### Scenario: Jump to a marked thread reply

- **WHEN** the user jumps to a mark naming a reply inside a thread
- **THEN** the channel opens, the thread panel opens for that thread,
  and the marked reply is selected

#### Scenario: Marked message is outside the loaded history

- **WHEN** the user jumps to a mark whose message is older than the
  loaded history
- **THEN** the surrounding history is loaded and the marked message is
  selected

#### Scenario: Jump taken from a different view

- **WHEN** the user jumps to a mark while looking at something other
  than the channel timeline -- a list of threads, or a thread panel --
- **THEN** the app SHALL show the marked message, changing whatever
  view and focus state is needed for it to be visible
- **AND** a thread panel left open from before the jump SHALL NOT
  remain on screen unless the mark itself names a thread

#### Scenario: Jump completes while the channel is still loading

- **WHEN** the user jumps to a mark in a channel whose history is still
  being fetched
- **THEN** the marked message SHALL still be the selected one once
  loading settles, rather than being displaced by the arriving data

#### Scenario: Jump to an unset mark

- **WHEN** the user jumps to a letter that has no mark
- **THEN** nothing changes and the user is told the mark is not set

#### Scenario: Abandoning the jump

- **WHEN** the user begins a jump and then presses a key that is not a
  mark letter
- **THEN** no navigation happens and the app returns to its previous
  state without an error

### Requirement: A mark jump is reversible

A jump to a mark SHALL record the location the user is leaving, and the
user SHALL be able to return to it with a dedicated back-jump. The
back-jump SHALL hold only the most recently departed location, replaced
on every jump, and SHALL work whether or not the jump changed channel.
Taking the back-jump SHALL itself record the location it departs, so
that repeating it returns the user to where they just were.

#### Scenario: Back-jump after a jump to another channel

- **WHEN** the user jumps to a mark in a different channel and then
  takes the back-jump
- **THEN** they return to the channel and message they were on before
  the jump

#### Scenario: Back-jump after a jump within the same channel

- **WHEN** the user jumps to a mark in the channel they are already
  viewing and then takes the back-jump
- **THEN** they return to the message they were on before the jump

#### Scenario: The back-jump remembers only the latest departure

- **WHEN** the user jumps to one mark, then jumps to another, and then
  takes the back-jump
- **THEN** they return to the location they left on the *second* jump,
  not the first

#### Scenario: Repeating the back-jump returns

- **WHEN** the user jumps to a mark, takes the back-jump, and takes the
  back-jump again
- **THEN** they are back at the mark, because the first back-jump
  recorded the location it departed

#### Scenario: Back-jump with nothing recorded

- **WHEN** the user takes the back-jump without having jumped to a mark
- **THEN** nothing changes and no error is raised

#### Scenario: Navigation history still records a cross-channel jump

- **WHEN** the user jumps to a mark in a different channel
- **THEN** the channel they left is recorded in the navigation history
  with the position they were at, as any other channel change is

### Requirement: A mark whose target is gone survives

When a jump cannot find the marked message, the channel SHALL still
open and the mark SHALL be retained. Marks SHALL NOT be deleted
automatically.

#### Scenario: Marked message was deleted

- **WHEN** the user jumps to a mark whose message no longer exists
- **THEN** the channel opens
- **AND** the user is told the message was not found
- **AND** the mark still exists and still points at the same location

#### Scenario: Marked channel is no longer accessible

- **WHEN** the user jumps to a mark naming a channel that can no longer
  be resolved
- **THEN** the user's current position is unchanged
- **AND** the user is told the channel is unavailable
- **AND** the mark still exists

### Requirement: Lowercase marks are temporary, uppercase marks persist

Marks named with a lowercase letter SHALL NOT survive restarting the
application. Marks named with an uppercase letter SHALL survive
restarting. A configuration option SHALL make lowercase marks persist as
well; it SHALL default to off.

#### Scenario: Lowercase mark after restart

- **WHEN** the user sets mark `a` and restarts the application
- **THEN** mark `a` is no longer set

#### Scenario: Uppercase mark after restart

- **WHEN** the user sets mark `A` and restarts the application
- **THEN** mark `A` still points at the same location

#### Scenario: All marks persistent

- **WHEN** the persist-all option is enabled, the user sets mark `a`,
  and the application is restarted
- **THEN** mark `a` still points at the same location

#### Scenario: Disabling persist-all keeps existing entries out of the way

- **WHEN** the persist-all option is turned off after lowercase marks
  were persisted
- **THEN** lowercase marks are not restored on the next start
- **AND** uppercase marks are unaffected

### Requirement: Marks belong to a workspace

Every mark SHALL be recorded against the workspace it was set in, and
SHALL be visible and jumpable only while that workspace is active. The
same letter in two workspaces SHALL refer to two independent marks.

#### Scenario: The same letter in two workspaces

- **WHEN** the user sets mark `a` in one workspace and mark `a` in
  another
- **THEN** each workspace's mark `a` points at its own location
- **AND** neither overwrites the other

#### Scenario: Marks from another workspace are not offered

- **WHEN** the user lists marks
- **THEN** only marks belonging to the active workspace are shown

#### Scenario: A mark that names another workspace is refused

- **WHEN** a jump is attempted to a mark whose workspace is not the
  active workspace
- **THEN** the jump does not happen
- **AND** the user is told cross-workspace jumps are not supported
- **AND** the mark is retained unchanged

### Requirement: Listing marks

The user SHALL be able to see the marks that exist and where each one
points. Each row SHALL identify the letter, the channel, and enough of
the marked message to recognise it. This information SHALL be available
without a network round trip and SHALL be shown for persisted marks
immediately after a restart.

#### Scenario: Listing on request

- **WHEN** the user invokes the list-marks command
- **THEN** an overlay lists every mark in the active workspace with its
  letter, channel, and a preview of the marked message

#### Scenario: Listing with no marks set

- **WHEN** the user invokes the list-marks command with no marks set
- **THEN** the overlay reports that no marks are set

#### Scenario: Preview available offline and after restart

- **WHEN** the user lists marks after restarting, with no network
  available
- **THEN** each persisted mark still shows its channel and message
  preview

#### Scenario: Selecting a row jumps to it

- **WHEN** the user selects a row in the marks overlay
- **THEN** the overlay closes and the app jumps to that mark

#### Scenario: Preview reflects the message as it was marked

- **WHEN** a marked message is edited after being marked
- **THEN** the overlay may still show the text as it was at mark time
- **AND** jumping to the mark still lands on the current message

### Requirement: The jump-key overlay is optional

Beginning a jump SHALL, by default, show the same list of marks so the
user can choose without memorising letters. A configuration option SHALL
suppress this, leaving the jump key silent as in vim. The option SHALL
NOT affect the list-marks command, which SHALL always show the overlay.

#### Scenario: Overlay shown on jump by default

- **WHEN** the user begins a jump and the option is at its default
- **THEN** the marks list is shown while the app waits for a letter

#### Scenario: Overlay suppressed

- **WHEN** the user begins a jump and the option is disabled
- **THEN** no overlay appears and the app waits silently for a letter

#### Scenario: The command ignores the option

- **WHEN** the option is disabled and the user invokes the list-marks
  command
- **THEN** the overlay is shown

#### Scenario: Choosing a letter while the overlay is open

- **WHEN** the overlay is shown on jump and the user presses a mark
  letter
- **THEN** the app jumps to that mark without requiring a second
  confirmation

### Requirement: Deleting marks

The user SHALL be able to delete marks by letter and from the marks
overlay. Deleting a mark that is not set SHALL be harmless.

#### Scenario: Deleting by letter

- **WHEN** the user deletes mark `a`
- **THEN** mark `a` is no longer set

#### Scenario: Deleting several letters at once

- **WHEN** the user deletes marks naming more than one letter
- **THEN** each named mark is no longer set

#### Scenario: Deleting a persisted mark

- **WHEN** the user deletes an uppercase mark and restarts
- **THEN** the mark is still absent

#### Scenario: Deleting from the overlay

- **WHEN** the user invokes the delete action on a selected overlay row
- **THEN** that mark is removed and the overlay continues to show the
  remaining marks

#### Scenario: Deleting an unset mark

- **WHEN** the user deletes a letter that has no mark
- **THEN** nothing changes and no error is raised
