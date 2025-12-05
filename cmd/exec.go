package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"multi-oc/internal/discovery"
	"multi-oc/internal/identity"
	"multi-oc/internal/keystore"
	"multi-oc/internal/kubeexec"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "<cluster> [oc args...]",
	Short: "Execute an oc command against a target cluster",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		ocArgs := args[1:]
		if len(ocArgs) == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_ = identity.EnsureHubLogin(ctx)
			return fmt.Errorf("Please pass oc arguments, e.g.,: get nodes")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		cluster, err := discovery.GetCluster(ctx, clusterName)
		if err != nil {
			return err
		}
		if cluster.APIURL == "" {
			return fmt.Errorf("API URL for cluster %s not found", clusterName)
		}

		// Attempt oc call, on first auth failure delete stored token and retry once
		for attempt := 0; attempt < 2; attempt++ {
			authArgs, cleanup, err := kubeexec.BuildOcAuthArgs(ctx, cluster)
			if err != nil {
				return err
			}
			defer cleanup()

			argsAll := append([]string{"--request-timeout=30s"}, authArgs...)
			argsAll = append(argsAll, ocArgs...)
			command := exec.CommandContext(ctx, "oc", argsAll...)
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			command.Stdin = os.Stdin
			if err := command.Run(); err != nil {
				if attempt == 0 {
					_ = keystore.DeleteTargetToken(cluster.Name)
					_, _ = os.Stderr.WriteString("Authentication failed. Please provide a fresh token when prompted.\n")
					continue
				}
				return err
			}
			return nil
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
