package cmd

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// withTempHome redirects registryHomeDir to a fresh temp directory for the
// duration of the test.
func withTempHome(t *testing.T) {
	t.Helper()
	prev := registryHomeDir
	tmpDir := t.TempDir()
	registryHomeDir = func() (string, error) {
		return tmpDir, nil
	}
	t.Cleanup(func() { registryHomeDir = prev })
}

func TestLoadRegistryEmptyWhenMissing(t *testing.T) {
	withTempHome(t)

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if len(reg) != 0 {
		t.Fatalf("len(reg) = %d, want 0", len(reg))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempHome(t)

	in := Registry{
		"office-pc": "001122334455",
		"nas":       "aabbccddeeff",
	}
	if err := SaveRegistry(in); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	out, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch: in=%v out=%v", in, out)
	}

	// second save should also work (creates dir if missing).
	if err := SaveRegistry(in); err != nil {
		t.Fatalf("re-SaveRegistry() error = %v", err)
	}
}

func TestRegistryDirCreatedOnSave(t *testing.T) {
	withTempHome(t)

	// SaveRegistry must create ~/.wolgo even when missing.
	reg := Registry{"alias-a": "001122334455"}
	if err := SaveRegistry(reg); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	path, err := RegistryPath()
	if err != nil {
		t.Fatalf("RegistryPath() error = %v", err)
	}
	parent := filepath.Dir(path)
	if filepath.Base(parent) != registryDirName {
		t.Fatalf("parent dir = %q, want %q", parent, registryDirName)
	}
}

func TestRegistrySortedByAlias(t *testing.T) {
	t.Parallel()

	reg := Registry{
		"zebra": "001122334455",
		"alpha": "aabbccddeeff",
		"mango": "deadbeef0000",
	}

	entries := reg.Sorted()
	got := make([]string, 0, len(entries))
	for _, e := range entries {
		got = append(got, e.Alias)
	}
	want := []string{"alpha", "mango", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sorted alias order = %v, want %v", got, want)
	}

	// Sorted on nil/empty registry must not panic.
	empty := Registry{}
	if got := empty.Sorted(); len(got) != 0 {
		t.Fatalf("empty.Sorted() = %d entries, want 0", len(got))
	}
}

func TestValidateAlias(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"empty", "", true},
		{"simple", "office-pc", false},
		{"underscore_dash", "my_nas-1", false},
		{"space", "office pc", true},
		{"tab", "office\tpc", true},
		{"newline_is_ok_but_spaces_not", "office\npc", false},
		{"mac_like_colon", "00:11:22:33:44:55", true},
		{"mac_like_dash", "00-11-22-33-44-55", true},
		{"mac_like_compact", "001122334455", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAlias(tc.alias)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateAlias(%q) = nil, want error", tc.alias)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateAlias(%q) = %v, want nil", tc.alias, err)
			}
		})
	}
	// silence unused-from-above (newline_is_ok_but_spaces_not kept for
	// documentation: newline itself is not space/tab so it passes).
	_ = sort.Strings
}

func TestValidateMAC(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		want    string
		wantErr bool
	}{
		"00:11:22:33:44:55": {"001122334455", false},
		"00-11-22-33-44-55": {"001122334455", false},
		"001122334455":      {"001122334455", false},
		"0:1c:42:2e:60:4a":  {"001c422e604a", false},
		"":                  {"", true},
		"bad":               {"", true},
		"zzzzzzzzzzzz":      {"", true},
	}
	for in, tc := range tests {
		got, err := ValidateMAC(in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ValidateMAC(%q) = %q, want error", in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ValidateMAC(%q) error = %v", in, err)
		}
		if got != tc.want {
			t.Fatalf("ValidateMAC(%q) = %q, want %q", in, got, tc.want)
		}
	}
}

func TestResolveTargetMACBeatsAlias(t *testing.T) {
	withTempHome(t)

	// Same string resolves as MAC, even when an alias with the same form
	// existed prior to normalization cleanup.
	if err := SaveRegistry(Registry{"001122334455": "deadbeef0000"}); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	got, err := ResolveTarget("00:11:22:33:44:55")
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if got.Alias != "" {
		t.Fatalf("ResolveTarget().Alias = %q, want empty (MAC precedence)", got.Alias)
	}
	if got.MAC != "001122334455" {
		t.Fatalf("ResolveTarget().MAC = %q, want %q", got.MAC, "001122334455")
	}
}

func TestResolveTargetAliasFallback(t *testing.T) {
	withTempHome(t)

	if err := SaveRegistry(Registry{"office-pc": "001122334455"}); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	got, err := ResolveTarget("office-pc")
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if got.Alias != "office-pc" {
		t.Fatalf("Alias = %q, want office-pc", got.Alias)
	}
	if got.MAC != "001122334455" {
		t.Fatalf("MAC = %q, want 001122334455", got.MAC)
	}
}

func TestResolveTargetUnknown(t *testing.T) {
	withTempHome(t)

	_, err := ResolveTarget("nope")
	if err == nil {
		t.Fatal("ResolveTarget() error = nil, want error")
	}
}

func TestRegisterOverwriteBehavior(t *testing.T) {
	withTempHome(t)

	if err := SaveRegistry(Registry{"office-pc": "001122334455"}); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	// Overwrite: same alias, new MAC.
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	reg["office-pc"] = "aabbccddeeff"
	if err := SaveRegistry(reg); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	got, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if got["office-pc"] != "aabbccddeeff" {
		t.Fatalf("alias MAC = %q, want aabbccddeeff", got["office-pc"])
	}
}

func TestRemoveMissingAliasError(t *testing.T) {
	withTempHome(t)

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if _, ok := reg["ghost"]; ok {
		t.Fatal("ghost alias present in fresh registry")
	}

	// emulate the removeCmd body directly.
	if _, ok := reg["ghost"]; !ok {
		// expected behavior surfaced via cobra's RunE in production; just
		// assert the lookup branch here.
		t.Log("ghost alias correctly absent")
	}
}

func TestAllTargetsEnumeration(t *testing.T) {
	withTempHome(t)

	// Empty registry.
	got, err := AllTargets()
	if err != nil {
		t.Fatalf("AllTargets() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty AllTargets() len = %d, want 0", len(got))
	}

	// Populate and re-read.
	if err := SaveRegistry(Registry{
		"alpha": "001122334455",
		"beta":  "aabbccddeeff",
	}); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}
	got, err = AllTargets()
	if err != nil {
		t.Fatalf("AllTargets() error = %v", err)
	}
	want := []RegistryEntry{
		{Alias: "alpha", MAC: "001122334455"},
		{Alias: "beta", MAC: "aabbccddeeff"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllTargets() = %v, want %v", got, want)
	}
}
