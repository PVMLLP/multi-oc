package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"multi-oc/cmd"
	"multi-oc/internal/discovery"
	"multi-oc/internal/kubeexec"
)

func main() {
	// Direkte Ausführung: moc <cluster> [oc args...]
	if len(os.Args) > 1 {
		first := os.Args[1]
		switch first {
		case "login", "ls", "logout", "help", "completion", "version":
			// cobra ausführen
		default:
			clusterName := first
			ocArgs := os.Args[2:]
			if len(ocArgs) == 0 {
				log.Fatalf("Bitte oc-Argumente angeben, z. B.: get nodes")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			cluster, err := discovery.GetCluster(ctx, clusterName)
			if err != nil {
				log.Fatal(err)
			}
			if cluster.APIURL == "" {
				log.Fatalf("API-URL für Cluster %s nicht gefunden", clusterName)
			}

			authArgs, cleanup, err := kubeexec.BuildOcAuthArgs(ctx, cluster)
			if err != nil {
				log.Fatal(err)
			}
			defer cleanup()

			args := append([]string{"--request-timeout=30s"}, authArgs...)
			args = append(args, ocArgs...)
			command := exec.CommandContext(ctx, "oc", args...)
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			command.Stdin = os.Stdin
			if err := command.Run(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
