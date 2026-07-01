package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// wakeAll, when true, sends WOL to every registered target.
var wakeAll bool

// rootCmd is the base command for wolgo.
// When invoked with a MAC address or alias (e.g. "wolgo 00:11:22:33:44:55"
// or "wolgo office-pc"), it sends a Wake-on-LAN magic packet. Use --all to
// fan out over every registered alias. Subcommands handle all other
// operations.
var rootCmd = &cobra.Command{
	Use:   "wolgo [MAC_ADDRESS | alias] [--all]",
	Short: "Wake-on-LAN and network utilities",
	Long: `wolgo is a CLI tool for Wake-on-LAN and network operations.

When given a MAC address or a registered alias, it sends a WOL magic packet.
Pass --all to fan out over every registered alias.

Examples:
  wolgo 00:11:22:33:44:55        Send WOL packet
  wolgo office-pc                Send WOL by alias
  wolgo --all                    Send WOL to every registered target
  wolgo find-ip 00:11:22:33:44:55  Find IP from MAC address
  wolgo find-ip office-pc          Find IP by alias
  wolgo find-ip --all              List registered aliases
  wolgo register 00:11:22:33:44:55 office-pc
  wolgo list                      List aliases
  wolgo remove office-pc          Remove alias`,
	Args: func(_ *cobra.Command, args []string) error {
		switch {
		case wakeAll && len(args) > 0:
			return fmt.Errorf("--all cannot be combined with a target argument")
		case !wakeAll && len(args) > 1:
			return fmt.Errorf("accepts at most one argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if !wakeAll && len(args) == 0 {
			return cmd.Help()
		}

		if wakeAll {
			return runWakeAll()
		}

		target, err := ResolveTarget(args[0])
		if err != nil {
			return err
		}

		macBytes, err := parseMAC(target.MAC)
		if err != nil {
			return fmt.Errorf("invalid MAC address: %w", err)
		}

		if err := sendWOL(macBytes); err != nil {
			return fmt.Errorf("failed to send WOL packet: %w", err)
		}

		fmt.Printf("Wake-on-LAN packet sent to %s\n", displayTarget(target))
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// displayTarget formats a resolved target for CLI output.
func displayTarget(t ResolvedTarget) string {
	if t.Alias != "" {
		return fmt.Sprintf("%s (%s)", t.Alias, t.MAC)
	}
	return t.MAC
}

// runWakeAll sends a WOL packet to every registered alias.
func runWakeAll() error {
	entries, err := AllTargets()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no registered targets; use 'wolgo register' first")
	}

	var errs int
	for _, entry := range entries {
		macBytes, err := parseMAC(entry.MAC)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: invalid MAC %s\n", entry.Alias, entry.MAC)
			errs++
			continue
		}
		if err := sendWOL(macBytes); err != nil {
			fmt.Fprintf(os.Stderr, "failed to send WOL to %s: %v\n", entry.Alias, err)
			errs++
			continue
		}
		fmt.Printf("Wake-on-LAN packet sent to %s -> %s\n", entry.Alias, entry.MAC)
	}
	if errs > 0 {
		return fmt.Errorf(
			"sent WOL to %d/%d targets (%d errors)",
			len(entries)-errs,
			len(entries),
			errs,
		)
	}
	return nil
}

func init() {
	rootCmd.Flags().BoolVar(&wakeAll, "all", false, "send WOL to every registered alias")
}
