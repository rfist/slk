## Purpose

Lets the user retrace their steps through a workspace -- back to the
channel they came from, and to the exact message they were reading when
they left it -- so that following a link, a search hit, or an unread
jump is a reversible detour rather than a one-way trip.

## ADDED Requirements

### Requirement: A location addresses a message, not just a channel

A navigable location SHALL identify a workspace, a channel, and
optionally a message within that channel and a thread that the message
belongs to. A location that names only a channel SHALL remain valid and
SHALL mean "the channel, at its default position".

#### Scenario: Location in the channel timeline

- **WHEN** a location is recorded while a channel message is selected
- **THEN** the location identifies the workspace, the channel, and that
  message

#### Scenario: Location inside a thread

- **WHEN** a location is recorded while a reply in an open thread is
  selected
- **THEN** the location identifies the workspace, the channel, the
  thread, and that reply

#### Scenario: Location with no selected message

- **WHEN** a location is recorded while no message is selected
- **THEN** the location identifies the workspace and channel only
- **AND** navigating to it opens the channel at its default position

### Requirement: The history records where the user left, not where they arrived

When the user navigates away from a channel, the history SHALL record
the position the user was at in the channel being left. It SHALL NOT
record the position the user occupied when they first entered it.

#### Scenario: Departure position is captured

- **WHEN** the user opens a channel, scrolls to an older message, and
  then navigates to a different channel
- **THEN** the recorded entry for the first channel names the older
  message the user was on at the moment of departure

#### Scenario: Returning restores the departure position

- **WHEN** the user navigates back to a channel they left while reading
  an older message
- **THEN** the channel opens with that message selected and visible
- **AND** the view is not reset to the newest message

#### Scenario: Leaving via a history walk also records the position

- **WHEN** the user walks back to a channel, moves to a different
  message there, and then walks forward away from it
- **THEN** the entry for that channel records the message they moved to
- **AND** walking back to it again restores that message, not the
  position it held before the walk

#### Scenario: Departure from an open thread

- **WHEN** the user is reading a reply inside an open thread and
  navigates to a different channel
- **THEN** the recorded entry names that thread and that reply
- **AND** navigating back reopens the thread with the reply selected

### Requirement: Back and forward walk the history

The user SHALL be able to move backward and forward through recorded
locations. Moving backward from the oldest entry, or forward from the
newest, SHALL do nothing rather than wrap.

#### Scenario: Walking backward

- **WHEN** the user navigates back
- **THEN** the previous recorded location is restored

#### Scenario: Walking forward after going back

- **WHEN** the user has navigated back and then navigates forward
- **THEN** the location they came from is restored

#### Scenario: At the boundary

- **WHEN** the user navigates back with no older entry available
- **THEN** nothing changes and no error is raised

#### Scenario: Walking does not grow the history

- **WHEN** the user navigates back and then forward several times
- **THEN** the number of recorded entries does not increase

### Requirement: A new visit truncates the forward path

Navigating to a location by any means other than walking the history
SHALL discard any entries ahead of the current position, matching
browser back/forward behaviour.

#### Scenario: New navigation after going back

- **WHEN** the user navigates back two entries and then opens a
  different channel directly
- **THEN** the two entries that were ahead are discarded
- **AND** navigating forward from the new position does nothing

### Requirement: History is per workspace and bounded

Each workspace SHALL keep its own independent history. The history
SHALL be bounded, discarding the oldest entries when the bound is
exceeded. History SHALL NOT survive restarting the application.

#### Scenario: Workspaces do not share history

- **WHEN** the user builds history in one workspace and switches to
  another
- **THEN** navigating back in the second workspace walks only that
  workspace's own history

#### Scenario: Oldest entries are discarded

- **WHEN** the number of recorded entries would exceed the bound
- **THEN** the oldest entries are dropped and the current position
  continues to refer to the same location

#### Scenario: History does not persist

- **WHEN** the application is restarted
- **THEN** the history is empty and navigating back does nothing

### Requirement: A location whose message is gone degrades to its channel

When a recorded location is restored but the message it names can no
longer be found, the channel SHALL still open and the entry SHALL be
retained in its degraded form. The user SHALL be told the message could
not be found. A recorded location whose *channel* can no longer be
resolved SHALL be skipped and removed.

#### Scenario: Message no longer exists

- **WHEN** the user navigates back to a location whose message has been
  deleted
- **THEN** the channel opens
- **AND** the user is shown a notice that the message was not found
- **AND** the user is not left in an unrelated position in history

#### Scenario: Message is outside the loaded history

- **WHEN** the user navigates back to a location whose message is older
  than what is currently loaded
- **THEN** the surrounding history is loaded and the message is selected

#### Scenario: Channel no longer resolves

- **WHEN** the user navigates back and the next entry names a channel
  that can no longer be resolved
- **THEN** that entry is skipped and removed from the history
- **AND** the walk continues to the next valid entry in the same
  direction

#### Scenario: No valid entry in that direction

- **WHEN** every remaining entry in the direction of travel names an
  unresolvable channel
- **THEN** the user's current position is left unchanged
- **AND** the unresolvable entries are removed
