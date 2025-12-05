package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"multi-oc/cmd"
	"multi-oc/internal/discovery"
	"multi-oc/internal/identity"
	"multi-oc/internal/keystore"
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
				// Ensure hub login before returning an error
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				_ = identity.EnsureHubLogin(ctx)
				log.Fatalf("Please pass oc arguments, e.g.,: get nodes")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			cluster, err := discovery.GetCluster(ctx, clusterName)
			if err != nil {
				log.Fatal(err)
			}
			if cluster.APIURL == "" {
				log.Fatalf("API URL for cluster %s not found", clusterName)
			}

			// Attempt oc call, on first auth failure delete stored token and retry once
			for attempt := 0; attempt < 2; attempt++ {
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
					// On first failure, drop cached token and retry
					if attempt == 0 {
						_ = keystore.DeleteTargetToken(cluster.Name)
						_, _ = os.Stderr.WriteString("Authentication failed. Please provide a fresh token when prompted.\n")
						continue
					}
					log.Fatal(err)
				}
				break
			}
			return
		}
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
