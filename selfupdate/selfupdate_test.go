package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

const testRepo = "brohd11/example"

// withServer points the URL builders at a test server for the duration of one
// test and restores them afterwards.
func withServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	orig := releasesLatestURL
	releasesLatestURL = func(string) string { return srv.URL + "/releases/latest" }
	t.Cleanup(func() { releasesLatestURL = orig })
}

func TestCheckUpdateAvailable(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/"+testRepo+"/releases/tag/v0.2.0")
		w.WriteHeader(http.StatusFound)
	})
	info, err := Check(context.Background(), testRepo, "v0.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestTag != "v0.2.0" || !info.Available {
		t.Errorf("got %+v, want tag v0.2.0 available", info)
	}
}

func TestCheckAbsoluteRedirectLocation(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/"+testRepo+"/releases/tag/v1.0.0")
		w.WriteHeader(http.StatusFound)
	})
	info, err := Check(context.Background(), testRepo, "v0.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestTag != "v1.0.0" || !info.Available {
		t.Errorf("got %+v, want tag v1.0.0 available", info)
	}
}

func TestCheckUpToDate(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/"+testRepo+"/releases/tag/v0.1.1")
		w.WriteHeader(http.StatusFound)
	})
	for _, current := range []string{"v0.1.1", "v0.1.1-2-gdfdcacf", "v0.1.1-2-gdfdcacf-dirty", "v0.1.1-dirty"} {
		info, err := Check(context.Background(), testRepo, current)
		if err != nil {
			t.Fatal(err)
		}
		if info.Available {
			t.Errorf("current %q: got available, want up to date against %s", current, info.LatestTag)
		}
	}
}

func TestCheckDevBuildNeverUpdates(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/"+testRepo+"/releases/tag/v9.9.9")
		w.WriteHeader(http.StatusFound)
	})
	info, err := Check(context.Background(), testRepo, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if info.Available {
		t.Errorf("dev build: got available against %s, want never", info.LatestTag)
	}
}

func TestCheckRedirectWithoutTag(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	})
	if _, err := Check(context.Background(), testRepo, "v0.1.1"); err == nil {
		t.Fatal("want an error when the redirect carries no release tag")
	}
}

func TestCheckNoRedirect(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if _, err := Check(context.Background(), testRepo, "v0.1.1"); err == nil {
		t.Fatal("want an error when /releases/latest does not redirect")
	}
}

func TestCheckServerError(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := Check(context.Background(), testRepo, "v0.1.1"); err == nil {
		t.Fatal("want an error on a 500 from /releases/latest")
	}
}

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.1", "v0.1.2", true},
		{"v0.1.1", "v0.2.0", true},
		{"v0.1.1", "v1.0.0", true},
		{"v0.1.1", "v0.1.1", false},
		{"v0.1.2", "v0.1.1", false},
		{"v0.2.0", "v0.10.0", true}, // numeric, not lexical
		{"dev", "v0.1.1", false},
		{"v0.1.1", "dev", false},
		{"v0.1", "v0.1.1", false},   // malformed current
		{"v0.1.1", "latest", false}, // malformed latest
		{"0.1.0", "v0.1.1", true},   // missing v prefix still parses
	}
	for _, c := range cases {
		if got := newer(c.current, c.latest); got != c.want {
			t.Errorf("newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"v0.1.1":                  "v0.1.1",
		"v0.1.1-dirty":            "v0.1.1",
		"v0.1.1-2-gdfdcacf":       "v0.1.1",
		"v0.1.1-2-gdfdcacf-dirty": "v0.1.1",
		"dev":                     "dev",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewUpdateCommand(t *testing.T) {
	cmd := NewUpdateCommand(testRepo, "example", "v0.1.1")
	if cmd.Use != "update" {
		t.Errorf("Use = %q, want %q", cmd.Use, "update")
	}
	if cmd.Flags().Lookup("check") == nil {
		t.Error("want a --check flag")
	}
}

func TestScriptName(t *testing.T) {
	cases := map[string]string{
		"windows": "install.ps1",
		"darwin":  "install.sh",
		"linux":   "install.sh",
	}
	for goos, want := range cases {
		if got := scriptName(goos); got != want {
			t.Errorf("scriptName(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestInstallScriptURLUsesPlatformInstaller(t *testing.T) {
	for goos, want := range map[string]string{
		"windows": "/install.ps1",
		"linux":   "/install.sh",
	} {
		if got := installScriptURL(testRepo, goos); !strings.HasSuffix(got, want) {
			t.Errorf("installScriptURL(%q, %q) = %q, want suffix %q", testRepo, goos, got, want)
		}
	}
}

func TestInterpreterUnix(t *testing.T) {
	argv, err := interpreter("linux", "/tmp/install.sh")
	if err != nil {
		t.Fatalf("interpreter: %v", err)
	}
	want := []string{"sh", "/tmp/install.sh", "--no-modify-path"}
	if len(argv) != len(want) {
		t.Fatalf("argv = %q, want %q", argv, want)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv = %q, want %q", argv, want)
		}
	}
}

// The Windows branch depends on a PowerShell being on PATH, which a Unix test
// runner has no reason to have -- so assert on whichever outcome the machine
// can produce. The command construction itself is covered independently below.
func TestInterpreterWindows(t *testing.T) {
	const url = "https://example.test/install.ps1"
	argv, err := interpreter("windows", url)
	if err != nil {
		if _, lookErr := exec.LookPath("pwsh"); lookErr == nil {
			t.Fatalf("pwsh is on PATH but interpreter failed: %v", err)
		}
		if !strings.Contains(err.Error(), "no PowerShell found") {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	joined := strings.Join(argv, " ")
	for _, want := range []string{"-NoProfile", "-NonInteractive", "-Command", "Invoke-RestMethod", url, "-NoModifyPath"} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv %q is missing %q", argv, want)
		}
	}
	if strings.Contains(joined, "-File") {
		t.Errorf("argv %q should execute the installer in memory, not from a file", argv)
	}
}

func TestPowerShellInstallerArgs(t *testing.T) {
	argv := powershellInstallerArgs("powershell.exe", "https://example.test/it's/install.ps1")
	if len(argv) != 5 {
		t.Fatalf("argv = %q, want executable, three flags, and command", argv)
	}
	if argv[0] != "powershell.exe" || argv[3] != "-Command" {
		t.Fatalf("argv = %q, want powershell.exe ... -Command <command>", argv)
	}
	command := argv[4]
	for _, want := range []string{
		"Invoke-RestMethod -UseBasicParsing -TimeoutSec 30",
		"'https://example.test/it''s/install.ps1'",
		") -NoModifyPath",
		"if ($code -ne 0) { exit $code }",
	} {
		if !strings.Contains(command, want) {
			t.Errorf("command %q is missing %q", command, want)
		}
	}
}
