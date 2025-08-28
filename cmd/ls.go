package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"multi-oc/internal/discovery"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available clusters (from the hub)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		clusters, err := discovery.ListManagedClusters(ctx)
		if err != nil {
			return err
		}
		if len(clusters) == 0 {
			fmt.Println("No clusters found.")
			return nil
		}
		for _, c := range clusters {
			var extras []string
			if c.APIURL != "" {
				extras = append(extras, c.APIURL)
			}
			fmt.Printf("%s\t%s\n", c.Name, strings.Join(extras, " "))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
