# goutil

Shared utilities for the brohd11 Go apps. Grab-bag module: small, focused
packages that more than one app needs, published as
`github.com/brohd11/goutil` and overlaid locally via the `go.work` in the
parent directory.

## Packages

- `selfupdate` — check GitHub for a newer release (via the `/releases/latest`
  redirect, no API) and install it by running the repo's own installer —
  `install.sh` under `sh`, or `install.ps1` under PowerShell on Windows —
  against the running binary's directory. Includes `NewUpdateCommand`, the
  shared cobra `update` command used by the apps.
- `stream` — run a subprocess and relay its stdout+stderr to a `Reporter` one
  line at a time as it arrives (splitting on `\r` as well as `\n`, so progress
  bars stream), folding the last line into the error on a non-zero exit. The
  command-streaming machinery behind gitstack's `GitStream` and golaunch's
  script runner.

## Docs

- [`docs/install.md`](docs/install.md) — the shared install/update reference for the
  apps: every option `install.sh` and `install.ps1` take, how each decides about
  `PATH`, and how `<binary> update` works. The app READMEs link here rather than
  restating it.

## Development

```sh
go test ./...
```

Tag `v*` releases like the other libraries (bubblestack, gitstack); consumers
pin the tag in their `go.mod`.
