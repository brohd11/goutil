// Package selfupdate implements the shared self-update mechanism for the
// brohd11 Go apps: check GitHub for a newer release and, when one exists,
// install it by running the same installer the READMEs tell users to fetch —
// install.sh under sh on Unix, install.ps1 under PowerShell on Windows — with
// BIN_DIR pointed at the running binary's directory so the update lands in
// place, and --no-modify-path so an installed binary is never asked about PATH
// again.
//
// Check resolves the latest tag via the /releases/latest redirect, so neither
// the check nor the install touches the rate-limited GitHub API.
package selfupdate

import (
	"bufio"
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
)

// Info is the outcome of Check: the running version, the latest release tag,
// and whether the release is newer.
type Info struct {
	Current   string
	LatestTag string
	Available bool
}

// Check resolves the latest release tag of repo ("owner/name") via the
// /releases/latest redirect — github.com/<repo>/releases/latest answers 302
// to /releases/tag/<tag>, which gives the tag without touching the
// rate-limited API — and reports whether it is newer than current. A "dev"
// build is never comparable, hence never offered an update.
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

// Apply installs info.LatestTag of repo ("owner/name") into binDir by
// downloading the platform's installer and running it with BIN_DIR=binDir and
// VERSION pinned to the checked tag, so it installs exactly what Check saw.
// Overwriting the running binary is safe: both installers stage in a temp dir
// before moving into place, and the Windows one displaces the running .exe by
// renaming it (Windows locks it against overwrite, but not against rename).
// The script's output is streamed line-by-line to report.
func Apply(ctx context.Context, repo string, info Info, binDir string, report func(string, ...any)) error {
	if report == nil {
		report = func(string, ...any) {}
	}

	tmp, err := os.MkdirTemp("", "selfupdate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	name := scriptName(runtime.GOOS)
	script := filepath.Join(tmp, name)

	// Download to a file rather than piping into the interpreter: the env vars
	// and the --no-modify-path flag are needed either way, and this keeps a
	// failed download distinct from a failed install.
	if err := download(ctx, installScriptURL(repo), script); err != nil {
		return fmt.Errorf("downloading %s: %w", name, err)
	}

	argv, err := interpreter(runtime.GOOS, script)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(),
		"BIN_DIR="+binDir,
		"VERSION="+info.LatestTag,
	)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			report("%s", sc.Text())
		}
	}()

	runErr := cmd.Run()
	pw.Close()
	<-scanDone
	if runErr != nil {
		return fmt.Errorf("running %s: %w", name, runErr)
	}
	return nil
}

// BinDir is the directory of the running executable, symlinks deliberately
// unresolved: a dev install (install_unix.sh) is a symlink in ~/.local/bin,
// and install.sh's mv -f then replaces the symlink itself with the real
// release binary.
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
	installScriptURL = func(repo string) string {
		return "https://raw.githubusercontent.com/" + repo + "/main/" + scriptName(runtime.GOOS)
	}
)

// scriptName is the installer published for goos. Both live at the root of
// every app repo and take the same BIN_DIR/VERSION env vars and the same
// --no-modify-path flag, so nothing above this line has to care which is which.
func scriptName(goos string) string {
	if goos == "windows" {
		return "install.ps1"
	}
	return "install.sh"
}

// interpreter is the argv for running script non-interactively, ending in the
// --no-modify-path flag: an already-installed binary must never be asked about
// PATH again.
//
// On Windows that means PowerShell with -ExecutionPolicy Bypass, without which
// a script that was just downloaded refuses to run at all under the default
// policy. pwsh (PowerShell 7) is preferred when installed and powershell
// (Windows PowerShell 5.1, present on every Windows install) is the fallback.
func interpreter(goos, script string) ([]string, error) {
	if goos != "windows" {
		return []string{"sh", script, "--no-modify-path"}, nil
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
	return []string{
		shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", script, "--no-modify-path",
	}, nil
}

// download GETs url into path. This used to shell out to curl, which Windows
// only reliably has as a PowerShell alias for Invoke-WebRequest -- and that
// takes different arguments. net/http is already here for latestTag and
// behaves the same on every platform.
func download(ctx context.Context, url, path string) error {
	client := &http.Client{Timeout: 30 * time.Second}
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
		Timeout: 30 * time.Second, // matches bubblestack's selfUpdateCheckTimeout
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

// normalize reduces a git describe version to its base tag: any dash in the
// output introduces the -N-g<hash>[-dirty] suffix, e.g. "v0.1.1-2-gdfdcacf"
// becomes "v0.1.1". "dev" stays "dev" and is never comparable.
func normalize(v string) string {
	if i := strings.Index(v, "-"); i >= 0 {
		v = v[:i]
	}
	return v
}

// newer reports whether latest is a higher semver tag than current. Anything
// unparseable ("dev", malformed tags) is treated as not newer — better to skip
// an update than to reinstall on a misunderstanding.
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
