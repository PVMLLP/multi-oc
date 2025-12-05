package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"multi-oc/internal/configstate"
	"multi-oc/internal/identity"

	"github.com/spf13/cobra"
)

var (
	hubURL   string
	insecure bool
	caFile   string
	headless bool
	token    string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the hub (headless token flow)",
	Long:  "You will be prompted for the hub API URL (if not provided). A copyable OAuth URL is printed for token retrieval; paste the token to complete the login.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Prompt hub URL if not provided
		if hubURL == "" {
			fmt.Fprint(os.Stderr, "Hub API URL (e.g., https://api.hub.example:6443): ")
			line, _ := reader.ReadString('\n')
			hubURL = strings.TrimSpace(line)
		}
		// remove accidental leading '@'
		hubURL = strings.TrimPrefix(hubURL, "@")
		if hubURL == "" {
			return fmt.Errorf("hub API URL is required")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Persist hub URL first
		if err := configstate.SaveHub(hubURL); err != nil {
			return err
		}
		// Bridge flags to env used by EnsureHubLogin
		if insecure {
			_ = os.Setenv("MOC_HUB_INSECURE", "true")
		}
		if caFile != "" {
			_ = os.Setenv("MOC_HUB_CA_FILE", caFile)
		}
		// Always use headless EnsureHubLogin (prints OAuth URL and prompts for token)
		return identity.EnsureHubLogin(ctx)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&hubURL, "hub", "", "API URL of the hub cluster (https://api.hub:6443)")
	loginCmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification for the hub")
	loginCmd.Flags().StringVar(&caFile, "ca-file", "", "Path to a CA file for the hub")
	// Deprecated flags (kept for compatibility, no effect in headless flow):
	loginCmd.Flags().BoolVar(&headless, "headless", false, "Deprecated (no-op): login is headless by default")
	loginCmd.Flags().StringVar(&token, "token", "", "Deprecated (no-op): token will be prompted interactively")
}
