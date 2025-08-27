package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"multi-oc/internal/discovery"
	"multi-oc/internal/kubeexec"
)

var execCmd = &cobra.Command{
	Use:   "<cluster> [oc args...]",
	Short: "Führe ein oc-Kommando auf einem Ziel-Cluster aus",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		ocArgs := args[1:]
		if len(ocArgs) == 0 {
			return fmt.Errorf("Bitte oc-Argumente angeben, z. B.: get nodes")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Ziel-Cluster API/CA ermitteln
		cluster, err := discovery.GetCluster(ctx, clusterName)
		if err != nil {
			return err
		}
		if cluster.APIURL == "" {
			return fmt.Errorf("API-URL für Cluster %s nicht gefunden", clusterName)
		}

		// Temporäre Kubeconfig erzeugen (via OAuth SSO Token für Ziel-Cluster)
		kubeconfigPath, cleanup, err := kubeexec.BuildTempKubeconfigForCluster(ctx, cluster)
		if err != nil {
			return err
		}
		defer cleanup()

		command := exec.CommandContext(ctx, "oc", ocArgs...)
		command.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin
		return command.Run()
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
