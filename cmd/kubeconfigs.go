package cmd

import (
	"context"
	"fmt"
	"time"

	"multi-oc/internal/hubkubeconfig"

	"github.com/spf13/cobra"
)

var kubeconfigsCmd = &cobra.Command{
	Use:   "kubeconfigs",
	Short: "Fetch kubeconfigs for managed clusters from the hub (admin-kubeconfig secret)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		n, err := hubkubeconfig.WriteAllKubeconfigs(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d kubeconfig(s) to ~/.config/multi-oc/kubeconfigs\n", n)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(kubeconfigsCmd)
}


