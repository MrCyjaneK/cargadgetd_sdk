// Package cfg reads and writes one plugin's settings in
// ~/.config/cargadgetd/plugins/<id>.yaml.
package cfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var ownRoot string

// SetOwnRootForTest points plugin files at dir for the test.
func SetOwnRootForTest(t testing.TB, dir string) {
	t.Helper()
	prev := ownRoot
	ownRoot = dir
	t.Cleanup(func() { ownRoot = prev })
}

func OwnPath(id string) string {
	if !ownID(id) {
		return ""
	}
	root := ownRoot
	if root == "" {
		if testing.Testing() {
			return ""
		}
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil || home == "" {
				return ""
			}
			dir = filepath.Join(home, ".config")
		}
		root = filepath.Join(dir, "cargadgetd", "plugins")
	}
	return filepath.Join(root, id+".yaml")
}

func ownID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// OwnGet reports the plugin file's value. ok is false when the file or the
// key is absent.
func OwnGet(id, key string) (string, bool) {
	path := OwnPath(id)
	if path == "" || key == "" {
		return "", false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var m map[string]string
	if yaml.Unmarshal(b, &m) != nil || m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

// OwnSet writes one key in the plugin file and leaves the other keys.
func OwnSet(id, key, value string) error {
	path := OwnPath(id)
	if path == "" || key == "" {
		if testing.Testing() && ownRoot == "" {
			return os.ErrNotExist
		}
		return os.ErrInvalid
	}
	m := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(b, &m)
	}
	if m == nil {
		m = map[string]string{}
	}
	m[key] = value
	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".plugin-*.yaml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, werr := tmp.Write(out)
	cerr := tmp.Close()
	if werr != nil {
		os.Remove(name)
		return werr
	}
	if cerr != nil {
		os.Remove(name)
		return cerr
	}
	return os.Rename(name, path)
}

// OwnBool is a convenience for a "true"/"false" plugin value.
func OwnBool(id, key string) (bool, bool) {
	v, ok := OwnGet(id, key)
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
