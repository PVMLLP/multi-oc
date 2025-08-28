package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "moc",
	Short:         "multi-oc: Central CLI for multi-cluster OpenShift",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(func() {
		_ = os.Setenv("LANG", "C")
	})

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	rootCmd.SetHelpTemplate(fmt.Sprintf(`Usage:
  %s [cluster|command] [args...]

Commands:
  login           Login to the hub (SSO)
  ls              List available clusters
  logout          Remove stored credentials
  version         Show version and credits

Examples:
  moc login --hub https://api.hub.example:6443
  moc ls
  moc cluster1 get nodes

Credits:
  Thorsten Stremetzne, People Visions & Magic LLP
`, rootCmd.Use))
}
