## Purpose

Remembers the statuses the user has already set so that recurring ones --
lunch, commuting, focus time -- can be reapplied in a couple of keystrokes
instead of retyped.

## ADDED Requirements

### Requirement: Statuses are recorded when set

Every custom status the user successfully sets SHALL be recorded in the
history for the workspace it was set on, together with the expiry duration
chosen at the time.

#### Scenario: A new status is recorded

- **WHEN** the user sets the status `:taco: Lunch` with a 1-hour expiry
- **THEN** `:taco: Lunch` appears in that workspace's history with a
  remembered duration of 1 hour

#### Scenario: Failed set is not recorded

- **WHEN** a status change fails
- **THEN** nothing is added to the history

#### Scenario: Clearing is not recorded

- **WHEN** the user clears their status
- **THEN** no history entry is created for the empty status

### Requirement: Repeated statuses are not duplicated

Setting a status whose emoji and text match an existing history entry SHALL
update that entry rather than create a second one. Re-setting an existing
status with a different expiry SHALL update the entry's remembered
duration.

#### Scenario: Same status set twice

- **WHEN** the user sets `:taco: Lunch` on two separate days
- **THEN** the history contains exactly one `:taco: Lunch` entry

#### Scenario: Same status, new duration

- **WHEN** the user sets `:taco: Lunch` with a 30-minute expiry, having
  previously used 1 hour
- **THEN** the entry's remembered duration becomes 30 minutes

### Requirement: History is offered for reuse

The menu that sets a custom status SHALL present history entries as
directly selectable rows, ordered so that statuses used recently and often
appear before those used rarely or long ago. Selecting one SHALL apply it
immediately, using its remembered duration, without opening the composer.

#### Scenario: Reapplying a remembered status

- **WHEN** the user opens the menu and selects the history row
  `:taco: Lunch`
- **THEN** the status is set to `:taco: Lunch` with its remembered duration
- **AND** the menu closes without prompting for text or expiry

#### Scenario: Ordering

- **WHEN** one status was used ten times a month ago and another was used
  twice yesterday
- **THEN** both appear in the list, ordered by combined recency and
  frequency rather than by frequency alone

#### Scenario: Empty history

- **WHEN** the user has never set a custom status on this workspace
- **THEN** the menu shows no history rows and no empty-list placeholder
  occupies a selectable row

### Requirement: History is filterable

The user SHALL be able to narrow the history rows by typing, consistent
with how the menu already filters its other rows.

#### Scenario: Typing filters the list

- **WHEN** the history contains `:taco: Lunch` and `:house: Working from
  home`, and the user types `lun`
- **THEN** only `:taco: Lunch` remains selectable

### Requirement: History is scoped per workspace

Each workspace SHALL have its own history. Statuses set on one workspace
SHALL NOT appear in another workspace's list.

#### Scenario: Workspaces keep separate lists

- **WHEN** the user sets `:taco: Lunch` on workspace A only
- **THEN** workspace A's menu offers it and workspace B's menu does not

### Requirement: History survives restarts

History SHALL persist across app restarts and SHALL be available before or
without a network connection.

#### Scenario: Restart

- **WHEN** the user sets a status, quits, and relaunches
- **THEN** the status is still offered in the menu

#### Scenario: Offline

- **WHEN** the app starts while the workspace has not yet connected
- **THEN** opening the menu still lists the stored history

### Requirement: History is bounded

The history SHALL be bounded in size per workspace so that it cannot grow
without limit. When the bound is reached, the least useful entries by the
same recency-and-frequency ordering SHALL be dropped.

#### Scenario: Bound reached

- **WHEN** the user has set more distinct statuses than the bound allows
- **THEN** the stored history holds at most the bound
- **AND** the entries that remain are the most recently and frequently used

### Requirement: History is removable

The user SHALL be able to delete an individual history entry from the menu
without setting it.

#### Scenario: Deleting an entry

- **WHEN** the user selects a history row and invokes the delete action
- **THEN** the entry is removed from the list and from storage
- **AND** the current status, if any, is unchanged
