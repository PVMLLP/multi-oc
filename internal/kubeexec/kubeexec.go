package kubeexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"multi-oc/internal/discovery"
)

func BuildTempKubeconfigForCluster(ctx context.Context, c discovery.Cluster) (string, func(), error) {
	if c.APIURL == "" {
		return "", nil, fmt.Errorf("APIURL leer")
	}
	dir, err := os.MkdirTemp("", "moc-kubeconfig-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	kc := filepath.Join(dir, "config")

	args := []string{"login", "--server", c.APIURL, "--kubeconfig", kc}
	// CA-Handling
	if len(c.CAData) > 0 {
		caPath := filepath.Join(dir, "ca.crt")
		if err := os.WriteFile(caPath, c.CAData, 0o600); err != nil {
			cleanup(); return "", nil, err
		}
		args = append(args, "--certificate-authority", caPath)
	} else {
		args = append(args, "--insecure-skip-tls-verify=true")
	}
	// Web-Login (SSO) für Ziel-Cluster
	args = append(args, "--web")

	cmd := exec.CommandContext(ctx, "oc", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		cleanup(); return "", nil, fmt.Errorf("oc login fehlgeschlagen: %w", err)
	}
	return kc, cleanup, nil
}
