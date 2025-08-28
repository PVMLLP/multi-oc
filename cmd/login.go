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
	Short: "Login to the hub via browser SSO or headless with token",
	Long:  "If flags are omitted, you will be prompted for the hub API URL and optionally a token (for headless environments).",
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

		// Decide flow: if --headless or no token but headless desired → prompt token
		if headless && token == "" {
			fmt.Fprint(os.Stderr, "Hub OAuth token (sha256~..., leave empty to use browser flow): ")
			line, _ := reader.ReadString('\n')
			token = strings.TrimSpace(line)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := identity.LoginHub(ctx, hubURL, insecure, caFile, token); err != nil {
			return err
		}
		return configstate.SaveHub(hubURL)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&hubURL, "hub", "", "API URL of the hub cluster (https://api.hub:6443)")
	loginCmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification for the hub")
	loginCmd.Flags().StringVar(&caFile, "ca-file", "", "Path to a CA file for the hub")
	loginCmd.Flags().BoolVar(&headless, "headless", false, "Headless login (prompt for token if --token not provided)")
	loginCmd.Flags().StringVar(&token, "token", "", "OAuth token for the hub (used with --headless)")
}
