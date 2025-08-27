package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"multi-oc/internal/configstate"
	"multi-oc/internal/identity"
)

var (
	hubURL string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login am Hub via Browser-SSO",
	RunE: func(cmd *cobra.Command, args []string) error {
		if hubURL == "" {
			return fmt.Errorf("--hub ist erforderlich")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := identity.LoginHub(ctx, hubURL); err != nil {
			return err
		}
		return configstate.SaveHub(hubURL)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&hubURL, "hub", "", "API-URL des Hub-Clusters (https://api.hub:6443)")
}
