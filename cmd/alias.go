// Package cmd provides the wolgo CLI commands and helpers.
package cmd

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyAlias is returned when a registration attempt supplies an empty
// alias.
var ErrEmptyAlias = errors.New("alias must not be empty")

// ErrInvalidAlias is returned when an alias contains forbidden characters
// (whitespace).
var ErrInvalidAlias = errors.New("alias must not contain spaces or tabs")

// ErrAliasIsMAC is returned when an alias collides with the MAC address
// format, which would defeat the MAC-beats-alias resolution rule.
var ErrAliasIsMAC = errors.New("alias must not be a valid MAC address")

// ValidateAlias enforces the alias naming rules:
//   - non-empty
//   - no spaces or tabs
//   - not parseable as a MAC address (would conflict with MAC-beats-alias)
func ValidateAlias(alias string) error {
	if alias == "" {
		return ErrEmptyAlias
	}
	if strings.ContainsAny(alias, " \t") {
		return ErrInvalidAlias
	}
	if normalizeMAC(alias) != "" {
		return ErrAliasIsMAC
	}
	return nil
}

// NormalizeMAC returns the canonical lowercase no-separator form of a MAC
// address, or an empty string if the input is not a valid MAC.
func NormalizeMAC(mac string) string {
	return normalizeMAC(mac)
}

// ValidateMAC parses a MAC, requires it to be valid, and returns its
// canonical form.
func ValidateMAC(mac string) (string, error) {
	normalized := normalizeMAC(mac)
	if normalized == "" {
		return "", fmt.Errorf("invalid MAC address: %s", mac)
	}
	return normalized, nil
}
