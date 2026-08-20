## Purpose

Lets the user set, see, and clear their own Slack custom status without
leaving slk, with an expiry so a temporary status cleans itself up instead
of lingering for days.

## ADDED Requirements

### Requirement: Set a custom status

The user SHALL be able to set a custom status consisting of an emoji and a
short text on the active workspace. Either part MAY be omitted, but not
both. When text is given with no emoji, the resulting status SHALL still be
valid and SHALL carry whatever default emoji Slack assigns.

#### Scenario: Emoji and text together

- **WHEN** the user sets a status with emoji `:taco:` and text `Lunch`
- **THEN** the workspace's Slack profile carries that emoji and text
- **AND** the status bar shows the status within one render

#### Scenario: Text with no emoji

- **WHEN** the user sets a status with text `Heads down` and no emoji
- **THEN** the status is accepted and applied

#### Scenario: Both parts empty

- **WHEN** the user submits a status with neither emoji nor text
- **THEN** the app SHALL reject the submission and keep the composer open
- **AND** the existing status SHALL be left untouched

#### Scenario: Text longer than Slack permits

- **WHEN** the user enters status text longer than 100 characters
- **THEN** the app SHALL prevent further input rather than sending a
  request that Slack will reject

### Requirement: Status applies to the active workspace only

Setting, clearing, and displaying a custom status SHALL be scoped to the
workspace that is active at the moment the action is committed. Other
connected workspaces SHALL be unaffected.

#### Scenario: Other workspaces unaffected

- **WHEN** the user sets a status while workspace A is active
- **THEN** workspace B's status is unchanged

#### Scenario: Status bar follows the active workspace

- **WHEN** the user switches from workspace A, which has a status, to
  workspace B, which has none
- **THEN** the status bar stops showing a custom status
- **AND** switching back to A shows A's status again without a network
  round trip

### Requirement: Choose an expiry when setting a status

The user SHALL be able to choose when a status expires, from a set of
durations that includes at least 30 minutes, 1 hour, 4 hours, end of day,
and "don't clear". The chosen expiry SHALL be sent to Slack so that it is
enforced for every client, not only slk.

#### Scenario: Expiry sent to Slack

- **WHEN** the user sets a status with a 1-hour expiry
- **THEN** the request carries an expiry timestamp one hour in the future
- **AND** the status clears on other Slack clients at that time without
  slk running

#### Scenario: No expiry

- **WHEN** the user selects "don't clear"
- **THEN** the status is set with no expiry and persists until changed

### Requirement: Default expiry

When the user sets a status without choosing an expiry, the app SHALL apply
a default expiry. The default SHALL be configurable, and SHALL be 1 hour
when unconfigured.

#### Scenario: Unconfigured default

- **WHEN** a user with no expiry setting in their config sets a status
  without choosing a duration
- **THEN** the status is set to expire in 1 hour

#### Scenario: Configured default

- **WHEN** a user has configured a default expiry of 4 hours and sets a
  status without choosing a duration
- **THEN** the status is set to expire in 4 hours

#### Scenario: Default disabled

- **WHEN** a user has configured the default expiry as "don't clear"
- **THEN** a status set without choosing a duration has no expiry

### Requirement: Expired statuses stop being displayed

The app SHALL stop displaying a custom status once its expiry has passed,
without requiring a restart, a reconnect, or any user action.

#### Scenario: Expiry passes while the app is running

- **WHEN** a status with a 1-hour expiry has been showing for one hour
- **THEN** the status bar stops showing it within one minute of the expiry

#### Scenario: Expiry passes while the app is suspended

- **WHEN** the machine sleeps past a status expiry and then wakes
- **THEN** the status bar does not show the expired status

#### Scenario: Status already expired at startup

- **WHEN** the app starts and the workspace's stored status has an expiry
  in the past
- **THEN** no custom status is displayed

### Requirement: Clear a custom status

The user SHALL be able to clear the current custom status in one action
from any surface that can set one. Clearing SHALL be offered only when a
status is currently set.

#### Scenario: Clearing a set status

- **WHEN** the user chooses to clear their status
- **THEN** the Slack profile's status text and emoji are emptied
- **AND** the status bar's custom-status segment disappears

#### Scenario: Nothing to clear

- **WHEN** no custom status is set
- **THEN** the clear action is not offered

### Requirement: Display the current status

The status bar SHALL show the active workspace's custom status when one is
set, rendering the emoji as a glyph rather than as a shortcode, and SHALL
show nothing when no status is set. The segment SHALL be truncated rather
than allowed to push other status-bar segments off screen.

#### Scenario: Shortcode rendered as a glyph

- **WHEN** the current status emoji is `:taco:`
- **THEN** the status bar shows the taco glyph, not the literal text
  `:taco:`

#### Scenario: Narrow terminal

- **WHEN** the terminal is too narrow to show the full status text
- **THEN** the status text is truncated
- **AND** the connection and presence segments remain visible

### Requirement: Status is known at startup

On connecting to a workspace, the app SHALL display any custom status
already set on that workspace, without an extra request beyond those the
startup sequence already makes.

#### Scenario: Status set from another client before launch

- **WHEN** the user set a status from the Slack desktop app and then starts
  slk
- **THEN** slk shows that status once the workspace finishes connecting

### Requirement: Status changes from other clients are reflected

While connected, the app SHALL update its displayed status when the user's
status is changed outside slk.

#### Scenario: Status set on another device

- **WHEN** the user sets a status from the Slack mobile app while slk is
  running and connected
- **THEN** slk's status bar shows the new status without a restart

#### Scenario: Status cleared on another device

- **WHEN** the user clears their status from another client
- **THEN** slk's status-bar segment disappears

### Requirement: Set a status from the presence menu

The menu bound to the "set status" key SHALL offer setting a custom status
alongside its existing presence and do-not-disturb actions, including an
entry point for composing a new status and a row for clearing the current
one.

#### Scenario: Composing from the menu

- **WHEN** the user opens the menu and chooses to set a custom status
- **THEN** a composer opens for entering emoji, text, and expiry
- **AND** committing it applies the status and closes the composer

#### Scenario: Abandoning the composer

- **WHEN** the user cancels out of the composer
- **THEN** no status change is sent and the previous status remains

#### Scenario: Existing presence actions still reachable

- **WHEN** the menu is opened
- **THEN** the presence and do-not-disturb actions behave exactly as before
  this change

### Requirement: Set a status from the command line

The app SHALL accept a `status` command that sets, clears, or opens a
composer for the custom status.

The argument grammar SHALL be: an optional leading `:emoji:` token, an
optional trailing duration token, and the remaining tokens as the status
text. Recognised duration tokens SHALL include at least `30m`, `1h`, `4h`,
`today`, and `never`.

#### Scenario: Full form

- **WHEN** the user runs `:status :taco: Lunch 1h`
- **THEN** the status is set to the taco emoji with text `Lunch`, expiring
  in one hour

#### Scenario: Text only

- **WHEN** the user runs `:status Heads down`
- **THEN** the status is set with text `Heads down` and the default expiry

#### Scenario: Clearing

- **WHEN** the user runs `:status clear`
- **THEN** the current status is cleared

#### Scenario: No arguments

- **WHEN** the user runs `:status` with no arguments
- **THEN** the composer opens, pre-filled with the current status

#### Scenario: Text that ends in a duration-shaped word

- **WHEN** the user runs `:status Sprint review 1h`
- **THEN** the trailing token is consumed as the expiry and the text is
  `Sprint review`
- **AND** the app SHALL surface the parsed text and expiry back to the user
  so the interpretation is visible rather than silent

### Requirement: Failures are visible and do not leave a false display

A status change SHALL be reflected in the UI immediately, without waiting
for the network. If the request subsequently fails, the app SHALL restore
the previously displayed status and tell the user the change did not take
effect.

#### Scenario: Request fails

- **WHEN** the user sets a status and the request returns an error
- **THEN** the status bar returns to what it showed before the attempt
- **AND** a message tells the user the status change failed

#### Scenario: Workspace disconnected

- **WHEN** the user attempts to set a status while the active workspace has
  no usable connection
- **THEN** the app reports that the change could not be sent
- **AND** does not display the unsent status as if it were applied
