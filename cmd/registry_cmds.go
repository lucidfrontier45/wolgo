package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register <alias> <mac>",
	Short: "Register an alias for a MAC address",
	Long: `Register a human-readable alias for a MAC address.

The alias is saved to ~/.wolgo/targets.json and can then be used wherever a
MAC address is accepted. Registering an existing alias overwrites it.

Examples:
  wolgo register office-pc 00:11:22:33:44:55`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		alias := args[0]
		if err := ValidateAlias(alias); err != nil {
			return err
		}

		normalized, err := ValidateMAC(args[1])
		if err != nil {
			return err
		}

		reg, err := LoadRegistry()
		if err != nil {
			return err
		}

		overwriting := false
		for existingAlias, existingMAC := range reg {
			if existingMAC == normalized && existingAlias != alias {
				fmt.Printf("note: %s already registered as %q\n", normalized, existingAlias)
			}
			if existingAlias == alias {
				overwriting = true
			}
		}

		reg[alias] = normalized
		if err := SaveRegistry(reg); err != nil {
			return err
		}

		action := "registered"
		if overwriting {
			action = "updated"
		}
		fmt.Printf("%s %s -> %s\n", action, alias, normalized)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered aliases",
	Long: `Print every registered alias and its associated MAC address, sorted
by alias.

Example: wolgo list`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		reg, err := LoadRegistry()
		if err != nil {
			return err
		}
		entries := reg.Sorted()
		if len(entries) == 0 {
			fmt.Println("no registered targets")
			return nil
		}
		for _, entry := range entries {
			fmt.Printf("%s -> %s\n", entry.Alias, entry.MAC)
		}
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Remove a registered alias",
	Long: `Remove the entry for the given alias from the registry.

Example: wolgo remove office-pc`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		alias := args[0]
		reg, err := LoadRegistry()
		if err != nil {
			return err
		}
		if _, ok := reg[alias]; !ok {
			return fmt.Errorf("alias %q is not registered", alias)
		}
		delete(reg, alias)
		if err := SaveRegistry(reg); err != nil {
			return err
		}
		fmt.Printf("removed %s\n", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerCmd, listCmd, removeCmd)
}
