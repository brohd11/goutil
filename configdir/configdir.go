// Package configdir is the small layer of per-app config plumbing the monorepo apps
// kept duplicating: the hidden ~/.<app> directory convention, YAML load where a missing
// file is not an error, an atomic save that can't truncate a working config on failure,
// and a surgical single-key edit that leaves the rest of the user's file alone.
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

// SaveKey sets key=value on the top-level mapping of the YAML file at path, editing the
// parsed node tree rather than re-marshalling a struct. Every other key, and every
// comment the user wrote, survives untouched -- which a load-modify-save round trip
// through a typed struct cannot promise, since it can only write the fields the struct
// knows about and drops comments entirely.
//
// A missing file (and its parent directory) is created. seed supplies the initial
// document for that case: nil starts from an empty mapping holding just this key, while
// a value is marshalled first so the app's other defaults land alongside it. Both
// behaviors were in use -- bubblestack's theme file wanted the former, gdaddon's config
// the latter -- and the difference was the only thing separating two otherwise identical
// copies of this function.
//
// The write is not atomic (unlike SaveAtomic): it is a read-modify-write of a file the
// user may be editing, and the read half is what would go stale.
func SaveKey(path, key, value string, seed any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		// nil leaves data empty; setMappingScalar seeds the mapping from nothing.
		if seed != nil {
			if data, err = yaml.Marshal(seed); err != nil {
				return err
			}
		}
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	setMappingScalar(&doc, key, value)
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// setMappingScalar sets key=value on the top-level mapping of a parsed YAML document,
// overwriting an existing key's value or appending the pair when absent. An empty
// document is initialized to a mapping first.
func setMappingScalar(doc *yaml.Node, key, value string) {
	if len(doc.Content) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Kind = yaml.ScalarNode
			m.Content[i+1].Tag = "!!str"
			m.Content[i+1].Value = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
