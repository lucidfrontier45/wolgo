// Package cmd provides the wolgo CLI commands and helpers.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// registryDirName is the directory under the user's home where the alias
// registry file lives.
const registryDirName = ".wolgo"

// registryFileName is the filename of the alias registry JSON.
const registryFileName = "targets.json"

// registryHomeDir returns the user's home directory. Tests override this to
// point at a temp directory.
var registryHomeDir = os.UserHomeDir

// Registry maps an alias to a normalized MAC address string.
//
// Stored on disk as JSON in $HOME/.wolgo/targets.json.
type Registry map[string]string

// RegistryPath returns the absolute path to the alias registry file.
func RegistryPath() (string, error) {
	home, err := registryHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, registryDirName, registryFileName), nil
}

// LoadRegistry reads the registry from disk. A missing file yields an empty
// registry with no error.
func LoadRegistry() (Registry, error) {
	path, err := RegistryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Registry{}, nil
		}
		return nil, fmt.Errorf("reading registry %s: %w", path, err)
	}

	if len(data) == 0 {
		return Registry{}, nil
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry %s: %w", path, err)
	}
	return reg, nil
}

// SaveRegistry persists the registry to disk, creating the parent directory
// if needed.
func SaveRegistry(reg Registry) error {
	path, err := RegistryPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing registry %s: %w", path, err)
	}
	return nil
}

// Sorted returns alias/MAC pairs sorted alphabetically by alias.
func (r Registry) Sorted() []RegistryEntry {
	entries := make([]RegistryEntry, 0, len(r))
	for alias, mac := range r {
		entries = append(entries, RegistryEntry{Alias: alias, MAC: mac})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Alias < entries[j].Alias
	})
	return entries
}

// RegistryEntry is a single alias/MAC pair, used for sorted output.
type RegistryEntry struct {
	Alias string
	MAC   string
}
