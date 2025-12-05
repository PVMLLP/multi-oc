package hubkubeconfig

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"multi-oc/internal/discovery"
	"multi-oc/internal/identity"
)

type secret struct {
	Data map[string]string `json:"data"`
}

// WriteClusterKubeconfig fetches the admin-kubeconfig Secret from the hub for the given cluster
// and writes it to ~/.config/multi-oc/kubeconfigs/<cluster>.kubeconfig.
// Returns true if a kubeconfig was written.
func WriteClusterKubeconfig(ctx context.Context, c discovery.Cluster) (bool, error) {
	if c.Name == "" {
		return false, fmt.Errorf("cluster name is empty")
	}
	if err := identity.EnsureHubLogin(ctx); err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, "oc", "get", "secret", "admin-kubeconfig", "-n", c.Name, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	var s secret
	if err := json.Unmarshal(out, &s); err != nil {
		return false, err
	}
	enc, ok := s.Data["kubeconfig"]
	if !ok || enc == "" {
		return false, fmt.Errorf("admin-kubeconfig secret found in %s but missing 'kubeconfig' key", c.Name)
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return false, err
	}
	target, err := defaultPath(c.Name)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return false, err
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// WriteAllKubeconfigs attempts to fetch and write kubeconfigs for all managed clusters.
func WriteAllKubeconfigs(ctx context.Context) (int, error) {
	clusters, err := discovery.ListManagedClusters(ctx)
	if err != nil {
		return 0, err
	}
	written := 0
	for _, c := range clusters {
		ok, err := WriteClusterKubeconfig(ctx, c)
		if err != nil {
			// continue on error to try others
			continue
		}
		if ok {
			written++
		}
	}
	return written, nil
}

func defaultPath(clusterName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "multi-oc", "kubeconfigs", clusterName+".kubeconfig"), nil
}


