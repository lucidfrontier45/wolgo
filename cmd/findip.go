package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// findIPAll, when true, lists every registered alias in `alias -> mac`
// format instead of resolving IPs.
var findIPAll bool

var findIPCmd = &cobra.Command{
	Use:   "find-ip [MAC_ADDRESS | alias] [--all]",
	Short: "Find IP addresses from a MAC address or alias",
	Long: `Scans the system ARP cache and prints all IP addresses associated
with the given MAC or registered alias. Supports Linux, macOS, and Windows.

Pass --all to list every registered alias instead of performing a lookup.

Examples:
  wolgo find-ip 00:11:22:33:44:55  Find IPs for a MAC
  wolgo find-ip office-pc          Find IPs by alias
  wolgo find-ip --all              List registered aliases`,
	Args: func(_ *cobra.Command, args []string) error {
		switch {
		case findIPAll && len(args) > 0:
			return fmt.Errorf("--all cannot be combined with a target argument")
		case !findIPAll && len(args) > 1:
			return fmt.Errorf("accepts at most one argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if findIPAll {
			return runFindIPAll()
		}
		if len(args) == 0 {
			return cmd.Help()
		}

		target, err := ResolveTarget(args[0])
		if err != nil {
			return err
		}

		ips, err := FindIPsFromMAC(target.MAC)
		if err != nil {
			return fmt.Errorf("ARP scan failed: %w", err)
		}

		if len(ips) == 0 {
			fmt.Printf("no IPs found for %s\n", displayTarget(target))
			return nil
		}
		for _, ip := range ips {
			fmt.Println(ip)
		}
		return nil
	},
}

// runFindIPAll prints one sorted "alias -> mac" line per registered alias.
// Per the CLI spec this intentionally omits IP data; callers can re-run
// find-ip per alias when they actually want addresses.
func runFindIPAll() error {
	entries, err := AllTargets()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no registered targets; use 'wolgo register' first")
	}
	for _, entry := range entries {
		fmt.Printf("%s -> %s\n", entry.Alias, entry.MAC)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(findIPCmd)
	findIPCmd.Flags().
		BoolVar(&findIPAll, "all", false, "list registered aliases instead of resolving IPs")
}
