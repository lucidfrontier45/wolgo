package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var findIPCmd = &cobra.Command{
	Use:   "find-ip <MAC_ADDRESS>",
	Short: "Find IP addresses from a MAC address",
	Long: `Scans the system ARP cache and prints all IP addresses associated
with the given MAC address. Supports Linux, macOS, and Windows.

Example: wolgo find-ip 00:11:22:33:44:55`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		macStr := args[0]

		if _, err := parseMAC(macStr); err != nil {
			return fmt.Errorf("invalid MAC address: %w", err)
		}

		ips, err := FindIPsFromMAC(macStr)
		if err != nil {
			return fmt.Errorf("ARP scan failed: %w", err)
		}

		for _, ip := range ips {
			fmt.Println(ip)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(findIPCmd)
}
