# goutil

Shared utilities for the brohd11 Go apps. Grab-bag module: small, focused
packages that more than one app needs, published as
`github.com/brohd11/goutil` and overlaid locally via the `go.work` in the
parent directory.

## Packages

- `selfupdate` — check GitHub for a newer release (via the `/releases/latest`
  redirect, no API) and install it by running the repo's own `install.sh`
  against the running binary's directory. Includes `NewUpdateCommand`, the
  shared cobra `update` command used by the apps.
- `stream` — run a subprocess and relay its stdout+stderr to a `Reporter` one
  line at a time as it arrives (splitting on `\r` as well as `\n`, so progress
  bars stream), folding the last line into the error on a non-zero exit. The
  command-streaming machinery behind gitstack's `GitStream` and golaunch's
  script runner.

## Development

```sh
go test ./...
```

Tag `v*` releases like the other libraries (bubblestack, gitstack); consumers
pin the tag in their `go.mod`.
