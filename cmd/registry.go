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

	"github.com/gofrs/flock"
)

// registryDirName is the directory under the user's home where the alias
// registry file lives.
const registryDirName = ".wolgo"

// registryFileName is the filename of the alias registry JSON.
const registryFileName = "targets.json"

// registryLockSuffix is appended to the registry filename to derive the
// advisory lock file guarding concurrent read-modify-write cycles.
const registryLockSuffix = ".lock"

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
//
// Callers that read and then write the registry (e.g. register, remove) must
// wrap the whole cycle in WithRegistryLock so concurrent processes cannot
// lose updates.
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
	// A file containing the JSON literal "null" unmarshals to a nil map;
	// writing to it would panic, so normalize to an empty registry.
	if reg == nil {
		reg = Registry{}
	}
	return reg, nil
}

// SaveRegistry persists the registry to disk, creating the parent directory
// if needed. The write is atomic: data is staged in a temp file in the same
// directory, fsynced, then renamed over the target so a crash can never
// leave a truncated registry behind.
//
// SaveRegistry itself takes no lock; read-modify-write cycles belong inside
// WithRegistryLock.
func SaveRegistry(reg Registry) error {
	path, err := RegistryPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+registryFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("staging registry write: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing registry %s: %w", path, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("setting registry permissions: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("syncing registry %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("closing registry temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("replacing registry %s: %w", path, err)
	}
	return nil
}

// WithRegistryLock runs fn while holding an exclusive advisory lock on the
// registry. It serializes read-modify-write cycles across concurrent wolgo
// processes on all supported platforms.
//
// The lock is kernel-managed and released automatically if the process dies,
// so a stale lock can never block future invocations.
func WithRegistryLock(fn func() error) error {
	path, err := RegistryPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	fl := flock.New(path + registryLockSuffix)
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("locking registry %s: %w", path, err)
	}
	defer func() { _ = fl.Unlock() }()

	return fn()
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
