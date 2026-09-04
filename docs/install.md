# Install options

Every brohd11 Go app ships the same two installers — `install.sh` for macOS and Linux,
`install.ps1` for Windows. Each is stamped from one template, so apart from the repo
name and the binary name they behave identically. This is the reference for all of them.

macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/<repo>/main/install.sh | sh
```

Windows, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/<repo>/main/install.ps1 | iex
```

`<repo>` is from this table, and `<binary>` is the command name — note that gossh
installs from `oh-my-gossh`, the one place the two differ:

| App | Repo |
| --- | --- |
| `gdaddon` | `brohd11/gdaddon` |
| `golaunch` | `brohd11/golaunch` |
| `gote` | `brohd11/gote` |
| `gossh` | `brohd11/oh-my-gossh` |
| `repoview` | `brohd11/repoview` |

Everything from here to [Windows](#windows) describes `install.sh`; the Windows section
covers where `install.ps1` differs, and the [Updating](#updating-in-place) and
[Troubleshooting](#troubleshooting) sections at the end cover both.

## Read it before you pipe it

Same install, in two steps:

```sh
curl -fsSL -o install.sh https://raw.githubusercontent.com/<repo>/main/install.sh
less install.sh && sh install.sh
```

Options work the same either way — `sh install.sh --no-modify-path`,
`VERSION=v0.1.1 sh install.sh`, and so on.

## Options

| Option | Default | Effect |
| --- | --- | --- |
| `BIN_DIR=<dir>` | `~/.local/bin` | Where the binary is installed. Created if missing; must be writable by you. |
| `VERSION=<tag>` | `latest` | Pin a release, e.g. `VERSION=v0.1.1`. |
| `--modify-path` | — | Append the `export PATH` line to your shell rc file without prompting. |
| `--no-modify-path` | — | Never touch any rc file; print the line instead. |
| `-h`, `--help` | — | Print the options and exit. |

Env vars go in front of the command, flags after it:

```sh
curl -fsSL https://raw.githubusercontent.com/<repo>/main/install.sh | BIN_DIR=/usr/local/bin sh
curl -fsSL https://raw.githubusercontent.com/<repo>/main/install.sh | sh -s -- --no-modify-path
```

## PATH handling

If `BIN_DIR` is already on your `PATH`, the installer says nothing about it. If it
isn't, one of three things happens:

- **default** — you're asked `Add it to <rc file>? [y/N]`, but only when there is a
  terminal to answer with. The prompt reads from `/dev/tty`, not stdin, so it still
  works under `curl | sh`. Answer anything but yes and it just prints the line.
- **`--no-modify-path`** — no rc file is ever read or written; the line is printed.
- **`--modify-path`** — the line is appended with no prompt, even with no terminal.
  This is the one for unattended setup scripts.

With no terminal and no `--modify-path` (CI, a piped run with no tty), nothing is
written: an installer should not edit dotfiles unasked.

**Which file gets appended**, based on `$SHELL`:

| Shell | File |
| --- | --- |
| zsh | `~/.zshrc` |
| bash on macOS | the first of `~/.bash_profile`, `~/.bash_login`, `~/.profile` that already exists; `~/.bash_profile` if none do |
| bash on Linux | `~/.bashrc` |
| anything else | none — the installer prints instead of writing |

macOS terminals start login shells, and bash sources only the *first* of those three
files that exists, so creating `~/.bash_profile` for someone whose setup lives in
`~/.profile` would silently stop `.profile` from ever being read again. Hence
preferring whatever is already there.

For fish, csh and tcsh — and any shell not listed — a POSIX `export` line would be
wrong syntax, so the installer names the directory and leaves adding it to you.

Two more details worth knowing:

- The appended block is marked `# added by install.sh -- <BIN_DIR> on PATH`, keyed on
  the directory rather than the binary. These apps share one `BIN_DIR` by default, so
  installing a second one won't stack a duplicate `export PATH` line. If the rc file
  mentions `BIN_DIR` anywhere already — your line, another tool's — the installer
  leaves the file alone and prints instead.
- If the rc file is a symlink (a managed dotfiles repo, say), the append follows it to
  the real file. The installer prints the link target before asking, so a dirty
  dotfiles repo isn't a surprise.

## Platforms

`install.sh` covers `darwin-arm64`, `darwin-amd64`, `linux-amd64` and `linux-arm64`; see
[Windows](#windows) for `windows-amd64`. An unsupported OS or architecture fails
immediately with a message rather than 404-ing partway through.

**macOS**: binaries downloaded *in a browser* carry a quarantine attribute that has to
be cleared before they'll run:

```sh
xattr -dr com.apple.quarantine path/to/<binary>
```

`install.sh` downloads with curl, so this doesn't apply to it — only to manual
downloads. Building from source avoids it too.

## Windows

```powershell
irm https://raw.githubusercontent.com/<repo>/main/install.ps1 | iex
```

`install.ps1` is the same installer in PowerShell: same version-less release assets,
same `BIN_DIR` and `VERSION` overrides, same three PATH modes. It needs Windows
PowerShell 5.1, which ships with Windows — nothing to install first.

Prefer to read before you run it? `irm <url> | Out-File install.ps1` and inspect it, then
`.\install.ps1`.

### Options

Env vars are set before the pipeline:

```powershell
$env:BIN_DIR = 'C:\tools'
$env:VERSION = 'v0.1.1'
irm https://raw.githubusercontent.com/<repo>/main/install.ps1 | iex
```

Flags are the awkward part: `iex` runs a string and cannot pass arguments to it. Build a
scriptblock instead, which can:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/<repo>/main/install.ps1))) -NoModifyPath
```

`-NoModifyPath`, `-ModifyPath` and `-h` mean exactly what they do on Unix. The `--kebab`
spellings work too, which is how `<binary> update` drives both scripts with one flag.

### Where it installs

`%LOCALAPPDATA%\bin` — that is `C:\Users\<you>\AppData\Local\bin` — created if it
isn't there. `Local`, not the `Roaming` half that `%APPDATA%` points at: a roaming
profile syncs between machines, and a native per-architecture `.exe` should not follow
you onto a different one.

All of these apps install there, so a single PATH entry covers all of them.

### PATH

If `%LOCALAPPDATA%\bin` isn't on your `PATH`, the installer offers to add it, and adds
it to the **user** PATH — no admin, nothing machine-wide. What it writes is the literal
string `%LOCALAPPDATA%\bin`, not the expanded path, so the entry stays portable and
reads the same as the rest of a normal user PATH.

The write goes to `HKCU:\Environment` directly rather than through
`[Environment]::SetEnvironmentVariable`, because that helper *expands* `%VAR%` references
when it reads the value back — a read-append-write through it would silently flatten
every `%USERPROFILE%`-style entry you already had into a hardcoded path. The registry
write is followed by a `WM_SETTINGCHANGE` broadcast, which is what the .NET helper would
have done for you, so newly opened terminals pick the change up without a sign-out.

Already-open terminals do not, including the one you installed from. The installer
prints the line for that:

```powershell
$env:Path += ';C:\Users\<you>\AppData\Local\bin'
```

To see exactly what is in your user PATH, unexpanded:

```powershell
(Get-Item HKCU:\Environment).GetValue('Path', '', 'DoNotExpandEnvironmentNames')
```

`-NoModifyPath` skips all of it, and `-ModifyPath` writes without asking. With neither
flag and no interactive console — CI, a service, or a self-update — nothing is written.

### Platforms

Only `windows-amd64` is published. On ARM64 Windows the installer says so and installs
the amd64 build, which runs under the OS's x64 emulation. 32-bit Windows is not
supported.

### SmartScreen

A `.zip` fetched over HTTP carries the Mark of the Web, and the `.exe` inside inherits
it — SmartScreen then blocks it on first run. The installer calls `Unblock-File` on the
binary before putting it in place, so this only bites you on a manual download from the
Releases page. Clear it the same way:

```powershell
Unblock-File -Path $env:LOCALAPPDATA\bin\<binary>.exe
```

It's the Windows counterpart of the macOS quarantine attribute above.

### A leftover `.old` file

Windows locks a running `.exe` against being overwritten, but not against being renamed.
So when the installer finds a binary already in place it renames it to
`<binary>.exe.old` and moves the new one in — which is what lets `<binary> update`
replace itself while running. The next install or update deletes the `.old`, once
nothing has it open. Deleting it yourself is fine too.

## Updating in place

Every one of these apps registers the same `update` command:

```sh
<binary> update           # install the newer release, if there is one
<binary> update --check   # report only; download nothing
```

It works on all three platforms and resolves the latest tag from the `/releases/latest`
redirect rather than the GitHub API, so neither checking nor installing is subject to API
rate limits. When an update exists, it runs the same installer the docs tell you to fetch:
`install.sh` under `sh`, or an in-memory `Invoke-RestMethod`/PowerShell scriptblock on
Windows. It sets `BIN_DIR` to the directory the running binary lives in and pins `VERSION`
to the tag it just checked. So the update lands exactly where the binary already is and
installs exactly what the check saw. `--no-modify-path` is passed too: an already-installed
binary should never ask about `PATH` again.

Overwriting the running binary is safe. Both installers stage the download in a temp
directory and only move it into place once the archive extracts cleanly, so a failed
download can't leave a half-written binary behind. On Windows the running `.exe` is
displaced by renaming rather than overwritten, since Windows won't allow the latter —
see [the `.old` file](#a-leftover-old-file).

Version comparison is plain semver on `vX.Y.Z`. Anything unparseable is treated as *not*
newer, so a `dev` build is never offered an update — it prints `dev build, skipping
update` and exits.

gdaddon and gossh run the same check from inside the TUI as well, so you can update
without dropping to a shell.

## Troubleshooting

**`no <os>-<arch> build is published`** — your platform isn't in the release matrix.
The message lists what is; building from source is the other option.

**`curl is required`** / **`unzip is required`** — install the named tool and re-run.
(macOS and Linux only; `install.ps1` uses what PowerShell already has.)

**`<dir> is not writable (set BIN_DIR to somewhere you own)`** — you pointed `BIN_DIR`
at something like `/usr/local/bin` without permission. Either use a directory you own
or run the installer under `sudo` with `BIN_DIR` set explicitly.

**`download failed: <url>`** — usually a `VERSION` tag that doesn't exist, or a release
that never published an asset for your platform. Check the repo's releases page; the
error prints the link.

**`<binary>: command not found` after a successful install** — `BIN_DIR` isn't on your
`PATH`. The installer prints the line to add; open a new shell after adding it, or run
the line in the current one. On Windows the same thing reads
`The term '<binary>' is not recognized`, and an already-open terminal will keep saying it
even after the PATH entry is written — that's expected; open a new one.

**`running install.ps1: ...` from `<binary> update`** — fetching or running the installer
failed; its own output is streamed above the error. `no PowerShell found on PATH` instead
means neither `pwsh` nor `powershell` was found, which shouldn't happen on stock Windows.

**`... cannot be loaded because running scripts is disabled on this system`** — the
execution policy blocked a saved `install.ps1`. The `irm | iex` one-liner and
`<binary> update` are unaffected because neither saves the PowerShell installer to disk.
To run a copy you downloaded yourself:
`powershell -ExecutionPolicy Bypass -File .\install.ps1`.
