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
		opts := hubkubeconfig.Options{
			Force:   kubeconfigsForce,
			Verbose: kubeconfigsVerbose,
			MSAName: kubeconfigsMSAName,
		}
		n, err := hubkubeconfig.WriteAllKubeconfigs(ctx, opts)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d kubeconfig(s) to ~/.config/multi-oc/kubeconfigs\n", n)
		return nil
	},
}

var (
	kubeconfigsForce   bool
	kubeconfigsVerbose bool
	kubeconfigsMSAName string
)

func init() {
	rootCmd.AddCommand(kubeconfigsCmd)
	kubeconfigsCmd.Flags().BoolVar(&kubeconfigsForce, "force", false, "Overwrite existing kubeconfigs")
	kubeconfigsCmd.Flags().BoolVar(&kubeconfigsVerbose, "verbose", false, "Verbose output")
	kubeconfigsCmd.Flags().StringVar(&kubeconfigsMSAName, "msa-name", "moc", "ManagedServiceAccount name to use (if available)")
}


