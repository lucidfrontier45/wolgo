package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for wolgo.
// When invoked with a MAC address (e.g. "wolgo 00:11:22:33:44:55"),
// it sends a Wake-on-LAN magic packet (backward-compatible behavior).
// Use subcommands for other operations like "wolgo find-ip <MAC>".
var rootCmd = &cobra.Command{
	Use:   "wolgo [MAC_ADDRESS | command]",
	Short: "Wake-on-LAN and network utilities",
	Long: `wolgo is a CLI tool for Wake-on-LAN and network operations.

When given a MAC address directly, it sends a WOL magic packet.
Use subcommands for other operations.

Examples:
  wolgo 00:11:22:33:44:55        Send WOL packet
  wolgo find-ip 00:11:22:33:44:55  Find IP from MAC address`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		macStr := args[0]
		mac, err := parseMAC(macStr)
		if err != nil {
			return fmt.Errorf("invalid MAC address: %w", err)
		}

		if err := sendWOL(mac); err != nil {
			return fmt.Errorf("failed to send WOL packet: %w", err)
		}

		fmt.Printf("Wake-on-LAN packet sent to %s\n", macStr)
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
