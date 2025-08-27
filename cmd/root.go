package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "moc",
	Short: "multi-oc: Zentrales CLI für Multi-Cluster OpenShift",
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
  %%s [cluster|command] [args...]

Commands:
  login           Am Hub anmelden (SSO)
  ls              Verfügbare Cluster auflisten
  logout          Tokens löschen

Beispiele:
  moc login --hub https://api.hub.example:6443
  moc ls
  moc cluster1 get nodes
`, rootCmd.Use))
}
