// Package configdir is the small layer of per-app config plumbing the monorepo apps
// kept duplicating: the hidden ~/.<app> directory convention, YAML load where a missing
// file is not an error, and an atomic save that can't truncate a working config on
// failure.
package configdir

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Dir returns ~/.<app>, the hidden config directory every app in the monorepo uses.
// It errors when the home directory can't be determined rather than guessing a
// fallback the user didn't ask for.
func Dir(app string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "."+app), nil
}

// Load reads the YAML file at path into v. A missing file is not an error — v stays
// zero, which is what every app's "first run" default already wants — while malformed
// YAML surfaces the parse error instead of silently keeping a half-read config.
func Load(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return yaml.Unmarshal(data, v)
}

// SaveAtomic marshals v to YAML and writes it to <dir>/<name> atomically: the temp
// file lives beside the target so rename stays on one filesystem, and a failed write
// leaves the previous file untouched. Ported from gote's SaveConfig.
func SaveAtomic(dir, name string, v any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "config-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o644); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		return err
	}
	ok = true
	return nil
}
