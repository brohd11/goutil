# goutil

## Packages

- `configcmd` — the shared opt-in Cobra `config` command: materialize an app's
  defaults, edit its config through `$EDITOR`/`$VISUAL` (falling back to Notepad on
  Windows), or open the config directory with `config --dir`.
- `configdir` — resolve the monorepo's `~/.<app>` config directories, load optional
  YAML, and save it atomically.
- `executil` — build argv-based subprocesses portably, including Windows `.cmd` and
  `.bat` launchers that must cross the `cmd.exe` boundary.
- `selfupdate` — check GitHub for a newer release (via the `/releases/latest`
  redirect, no API) and install it by running the repo's own installer —
  `install.sh` under `sh`, or an in-memory `Invoke-RestMethod`/PowerShell flow on
  Windows — against the running binary's directory. Includes `NewUpdateCommand`,
  the shared cobra `update` command used by the apps.
- `stream` — run a subprocess and relay its stdout+stderr to a `Reporter` one
  line at a time as it arrives (splitting on `\r` as well as `\n`, so progress
  bars stream), folding the last line into the error on a non-zero exit. The
  command-streaming machinery behind gitstack's `GitStream` and golaunch's
  script runner.
- `sysopen` — open a path, reveal a file, or launch a URL with native platform
  behavior (including `ShellExecuteW` on Windows), without taking a dependency on
  Cobra, Bubble Tea, or an app domain.

## Docs

- [`docs/install.md`](docs/install.md) — the shared install/update reference for the
  apps: every option `install.sh` and `install.ps1` take, how each decides about
  `PATH`, and how `<binary> update` works. The app READMEs link here rather than
  restating it.
