// Package selfupdate checks GitHub for a newer release and installs it by running the
// app's own installer (install.sh, or install.ps1 in PowerShell) with BIN_DIR set to the
// running binary's directory. The latest tag comes from the /releases/latest redirect,
// which avoids the rate-limited API.
package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/brohd11/goutil/stream"
)

// Info is the outcome of Check: the running version, the latest release tag,
// and whether the release is newer.
type Info struct {
	Current   string
	LatestTag string
	Available bool
}

// Check resolves repo's latest release tag and reports whether it is newer than current.
// A "dev" build is never offered an update.
func Check(ctx context.Context, repo, current string) (Info, error) {
	info := Info{Current: current}
	tag, err := latestTag(ctx, releasesLatestURL(repo))
	if err != nil {
		return info, err
	}
	info.LatestTag = tag
	info.Available = newer(normalize(current), tag)
	return info, nil
}

// Apply installs info.LatestTag into binDir by running the platform installer with
// VERSION pinned, streaming its output to report. Replacing the running binary is safe:
// the installers stage in a temp dir, and on Windows the running .exe is renamed aside.
func Apply(ctx context.Context, repo string, info Info, binDir string, report func(string, ...any)) error {
	if report == nil {
		report = func(string, ...any) {}
	}

	goos := runtime.GOOS
	name := scriptName(goos)
	source := installScriptURL(repo, goos)
	if goos != "windows" {
		tmp, err := os.MkdirTemp("", "selfupdate-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)

		script := filepath.Join(tmp, name)
		if err := download(ctx, source, script); err != nil {
			return fmt.Errorf("downloading %s: %w", name, err)
		}
		source = script
	}

	argv, err := interpreter(goos, source)
	if err != nil {
		return err
	}

	env := append(os.Environ(), "BIN_DIR="+binDir, "VERSION="+info.LatestTag)
	if err := stream.Cmd(ctx, "", env, report, argv...); err != nil {
		return fmt.Errorf("running %s: %w", name, err)
	}
	return nil
}

// BinDir is the running executable's directory, symlinks unresolved: a dev install is a
// symlink that the installer then replaces with the release binary.
func BinDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the running binary: %w", err)
	}
	return filepath.Dir(exe), nil
}

// releasesLatestURL / installScriptURL are vars so tests can point them at an
// httptest server.
var (
	releasesLatestURL = func(repo string) string {
		return "https://github.com/" + repo + "/releases/latest"
	}
	installScriptURL = func(repo, goos string) string {
		return "https://raw.githubusercontent.com/" + repo + "/main/" + scriptName(goos)
	}
)

// scriptName is the installer published for goos. Both take the same env vars and flags.
func scriptName(goos string) string {
	if goos == "windows" {
		return "install.ps1"
	}
	return "install.sh"
}

// interpreter is the argv that runs an installer non-interactively. source is a script
// path on Unix and the install.ps1 URL on Windows, where pwsh is preferred over
// powershell.
func interpreter(goos, source string) ([]string, error) {
	if goos != "windows" {
		return []string{"sh", source, "--no-modify-path"}, nil
	}

	shell := ""
	for _, c := range []string{"pwsh", "powershell"} {
		if p, err := exec.LookPath(c); err == nil {
			shell = p
			break
		}
	}
	if shell == "" {
		return nil, fmt.Errorf("no PowerShell found on PATH (looked for pwsh and powershell)")
	}
	return powershellInstallerArgs(shell, source), nil
}

// powershellInstallerArgs is the in-memory form of `irm <url> | iex`, via a scriptblock so
// --no-modify-path can be passed and a failing $code becomes the exit status.
func powershellInstallerArgs(shell, url string) []string {
	quotedURL := "'" + strings.ReplaceAll(url, "'", "''") + "'"
	command := "$ErrorActionPreference = 'Stop'; " +
		". ([scriptblock]::Create((Invoke-RestMethod -UseBasicParsing -TimeoutSec 30 -Uri " + quotedURL + "))) -NoModifyPath; " +
		"if ($code -ne 0) { exit $code }"
	return []string{shell, "-NoProfile", "-NonInteractive", "-Command", command}
}

// httpTimeout bounds each release request.
const httpTimeout = 30 * time.Second

// download GETs the Unix installer into path without requiring curl. Windows uses
// Invoke-RestMethod directly in powershellInstallerArgs instead.
func download(ctx context.Context, url, path string) error {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// latestTag GETs the /releases/latest URL without following the redirect and
// reads the tag off the Location header.
func latestTag(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	const marker = "/releases/tag/"
	i := strings.Index(loc, marker)
	if i < 0 {
		return "", fmt.Errorf("no release tag in redirect from %s (status %s)", url, resp.Status)
	}
	return loc[i+len(marker):], nil
}

// normalize strips a git-describe suffix: "v0.1.1-2-gdfdcacf" becomes "v0.1.1".
func normalize(v string) string {
	if i := strings.Index(v, "-"); i >= 0 {
		v = v[:i]
	}
	return v
}

// newer reports whether latest is a higher semver tag than current. Anything unparseable
// is not newer: skipping an update beats reinstalling on a misread.
func newer(current, latest string) bool {
	c, okC := parseSemver(current)
	l, okL := parseSemver(latest)
	if !okC || !okL {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseSemver parses a vX.Y.Z tag into its numeric components.
func parseSemver(tag string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}
