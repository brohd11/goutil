package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
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
