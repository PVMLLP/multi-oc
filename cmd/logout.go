package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"multi-oc/internal/identity"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Löscht gespeicherte Anmeldedaten",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := identity.LogoutHub(); err != nil {
			return err
		}
		fmt.Println("Anmeldedaten entfernt (Hub)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
