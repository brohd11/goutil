//go:build windows

package executil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBatchCommandRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name, rel string
	}{
		{"cmd", "argument helper.cmd"},
		{"bat", "argument helper.bat"},
		{"npm shim", filepath.Join("node_modules", ".bin", "argument helper.cmd")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, tc.rel)
			if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
				t.Fatal(err)
			}
			body := "@echo off\r\n\"%EXECUTIL_HELPER%\" -test.run=TestBatchHelper -- %*\r\n"
			if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			want := []string{"", "two words", "amp&pipe|caret^", "percent%value", `quote\"value`, "雪"}
			out := filepath.Join(dir, "args.json")
			t.Setenv("EXECUTIL_HELPER", os.Args[0])
			t.Setenv("EXECUTIL_HELPER_OUT", out)

			cmd, err := Command(append([]string{script}, want...)...)
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Run(); err != nil {
				t.Fatalf("batch wrapper: %v", err)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("batch argv = %#v, want %#v", got, want)
			}
		})
	}
}

func TestBatchHelper(t *testing.T) {
	if os.Getenv("EXECUTIL_HELPER_OUT") == "" {
		return
	}
	sep := 0
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i + 1
			break
		}
	}
	data, err := json.Marshal(os.Args[sep:])
	if err != nil {
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("EXECUTIL_HELPER_OUT"), data, 0o644); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
