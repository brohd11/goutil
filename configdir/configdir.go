// Package configdir is the per-app config plumbing: the ~/.<app> directory, YAML load,
// atomic save, and single-key edits that preserve the rest of the file.
package configdir

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Dir returns ~/.<app>, erroring rather than guessing when home is unknown.
func Dir(app string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "."+app), nil
}

// Load reads the YAML file at path into v. A missing file leaves v untouched; malformed
// YAML is an error.
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

// Ensure writes v to <dir>/<name> with SaveAtomic when that file does not exist yet, and
// reports whether it did. An existing file is never touched.
func Ensure(dir, name string, v any) (created bool, err error) {
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, SaveAtomic(dir, name, v)
}

// SaveAtomic marshals v to YAML and writes <dir>/<name> via a temp file and rename, so a
// failed write leaves the previous file intact.
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

// SaveKey sets key=value on the top-level mapping of the YAML file at path by editing the
// node tree, so the user's other keys and comments survive. A missing file is created
// from seed (nil starts from an empty mapping). Not atomic: it is a read-modify-write of
// a file the user may be editing.
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

// setMappingScalar sets key=value on the document's top-level mapping, creating the
// mapping if the document is empty.
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
