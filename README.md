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

## Development

```sh
go test ./...
```

Tag `v*` releases like the other libraries (bubblestack, gitstack); consumers
pin the tag in their `go.mod`.
