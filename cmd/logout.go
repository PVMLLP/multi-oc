package cmd

import (
	"fmt"

	"multi-oc/internal/identity"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := identity.LogoutHub(); err != nil {
			return err
		}
		fmt.Println("Credentials removed (hub)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
