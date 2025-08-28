package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "0.1.1"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and credits",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("moc %s\n", Version)
		fmt.Println("Credits: Thorsten Stremetzne, People Visions & Magic LLP")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
