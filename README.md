# slk — edge fork

> Fork of [gammons/slk](https://github.com/gammons/slk), a blazingly fast
> Slack TUI in Go. This `edge` branch is upstream `main` plus the feature
> branches below — each submitted (or planned) as an upstream PR. When
> upstream merges one, the branch is dropped here.
>
> Upstream docs: [README](https://github.com/gammons/slk#readme) ·
> [Wiki](https://github.com/gammons/slk/wiki) · [getslk.sh](https://getslk.sh)

## Prebuilt binaries

CI builds `edge` on every push — grab binaries from the latest
[Actions run](../../actions) artifacts:
`slk_macOS_ARM64`, `slk_Linux_X64`, `slk_Linux_ARM64`.

Verify what you're running: `slk --version`, or `:version` inside the TUI.

## Changes on `edge`

| Branch | What it does | Upstream status |
|---|---|---|
| `feature/thread-parent-select` | Thread parent message is selectable — react/copy/permalink the first message of a thread | to submit (bug fix) |
| `feature/live-thread-replies` | Live thread replies land in the open thread panel even when it shows a non-active channel | to submit (bug fix) |
| `feature/yank-message-text` | `y` copies the selected message's text (mentions/links resolved) to the clipboard via OSC 52 | to submit |
| `feature/version-command` | `:version` command surfaces build identity in the TUI | to submit |
| `feature/editor-compose` | `Ctrl+E` edits the compose draft in `$VISUAL`/`$EDITOR` | issue first (feature) |

Merged upstream: usergroup mentions ([#99](https://github.com/gammons/slk/pull/99)),
usergroup send fix ([#117](https://github.com/gammons/slk/pull/117)),
mention word-match ([#118](https://github.com/gammons/slk/pull/118)),
sixel inline images ([#131](https://github.com/gammons/slk/pull/131)).

Rebased on upstream `v0.15.0`: `yank` now writes through the OSC 52
clipboard path added in #142; `version-command` sits on top of upstream's
`internal/version` package.

## Branch layout

- `main` — clean mirror of upstream `main`; base for all feature branches and PRs
- `feature/*` — one change each, tests included
- `edge` — `main` + all feature branches + this README and the CI workflow

All credit for slk itself goes to [@gammons](https://github.com/gammons).
Licensed under the same terms as upstream (see [LICENSE](LICENSE)).
