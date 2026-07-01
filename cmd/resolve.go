// Package cmd provides the wolgo CLI commands and helpers.
package cmd

import "fmt"

// ResolvedTarget describes a target after CLI input resolution: the
// normalized MAC plus, when resolved via alias, the alias it came from.
type ResolvedTarget struct {
	Alias string // empty when resolved directly as a MAC
	MAC   string // normalized (lowercase, no separators)
}

// ResolveTarget maps a CLI argument to a MAC address.
//
// Precedence:
//  1. Valid MAC → use MAC (alias field stays empty).
//  2. Existing alias in the registry → use the registered MAC.
//  3. Otherwise → error.
//
// The MAC-beats-alias rule keeps users from accidentally hiding a MAC behind
// a same-named alias.
func ResolveTarget(input string) (ResolvedTarget, error) {
	if input == "" {
		return ResolvedTarget{}, fmt.Errorf("target must not be empty")
	}

	if mac := normalizeMAC(input); mac != "" {
		return ResolvedTarget{MAC: mac}, nil
	}

	reg, err := LoadRegistry()
	if err != nil {
		return ResolvedTarget{}, err
	}
	if mac, ok := reg[input]; ok {
		return ResolvedTarget{Alias: input, MAC: mac}, nil
	}

	return ResolvedTarget{}, fmt.Errorf("unknown target: %s (not a valid MAC or alias)", input)
}

// AllTargets returns every registered alias/MAC pair, sorted by alias.
func AllTargets() ([]RegistryEntry, error) {
	reg, err := LoadRegistry()
	if err != nil {
		return nil, err
	}
	return reg.Sorted(), nil
}
