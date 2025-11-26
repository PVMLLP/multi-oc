package discovery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"multi-oc/internal/identity"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Cluster struct {
	Name   string `json:"name"`
	APIURL string `json:"apiURL"`
	CAData []byte `json:"caData"`
}

type managedClusterList struct {
	Items []managedCluster `json:"items"`
}

type managedCluster struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		ManagedClusterClientConfigs []struct {
			URL      string `json:"url"`
			CABundle string `json:"caBundle"`
		} `json:"managedClusterClientConfigs"`
	} `json:"spec"`
}

type cacheFile struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Items       []Cluster `json:"items"`
}

func configDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "multi-oc"), nil
}

func cachePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	cdir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cdir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(cdir, "managedclusters.json"), nil
}

func ttl() time.Duration {
	v := os.Getenv("MOC_DISCOVERY_TTL_SECONDS")
	if v == "" {
		return 60 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 60 * time.Second
	}
	return time.Duration(n) * time.Second
}

func ListManagedClusters(ctx context.Context) ([]Cluster, error) {
	// 1) Cache versuchen
	cp, err := cachePath()
	if err == nil {
		if b, err := os.ReadFile(cp); err == nil && len(b) > 0 {
			var cf cacheFile
			if json.Unmarshal(b, &cf) == nil {
				if time.Since(cf.GeneratedAt) <= ttl() {
					return cf.Items, nil
				}
			}
		}
	}

	// 2) Live vom Hub via oc
	cmd := exec.CommandContext(ctx, "oc", "get", "managedclusters.cluster.open-cluster-management.io", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		// Erster Versuch fehlgeschlagen → Login sicherstellen und einmalig erneut versuchen
		// Nur bei typischen Auth-/Kontextfehlern sinnvoll; wir versuchen generell einmal.
		if loginErr := identity.EnsureHubLogin(ctx); loginErr != nil {
			// Rückgabe einer freundlicheren Fehlermeldung
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return nil, fmt.Errorf("Hub-Verbindung erforderlich (Login fehlgeschlagen): %w", loginErr)
			}
			return nil, fmt.Errorf("Hub-Verbindung erforderlich (oc nicht ausführbar?): %w", loginErr)
		}
		// Retry
		cmd2 := exec.CommandContext(ctx, "oc", "get", "managedclusters.cluster.open-cluster-management.io", "-o", "json")
		out2, err2 := cmd2.Output()
		if err2 != nil {
			return nil, fmt.Errorf("oc get managedclusters nach Login fehlgeschlagen: %w", err2)
		}
		out = out2
	}
	var mcl managedClusterList
	if err := json.Unmarshal(out, &mcl); err != nil {
		return nil, err
	}
	result := make([]Cluster, 0, len(mcl.Items))
	for _, it := range mcl.Items {
		var api string
		var caBytes []byte
		if len(it.Spec.ManagedClusterClientConfigs) > 0 {
			api = it.Spec.ManagedClusterClientConfigs[0].URL
			ca := it.Spec.ManagedClusterClientConfigs[0].CABundle
			if ca != "" {
				if decoded, decErr := base64.StdEncoding.DecodeString(ca); decErr == nil {
					caBytes = decoded
				} else {
					caBytes = []byte(ca)
				}
			}
		}
		result = append(result, Cluster{
			Name:   it.Metadata.Name,
			APIURL: api,
			CAData: caBytes,
		})
	}

	// 3) Cache schreiben (best effort)
	if cp, err := cachePath(); err == nil {
		_ = os.WriteFile(cp, mustJSON(cacheFile{GeneratedAt: time.Now(), Items: result}), 0o600)
	}
	return result, nil
}

func GetCluster(ctx context.Context, name string) (Cluster, error) {
	clusters, err := ListManagedClusters(ctx)
	if err != nil {
		return Cluster{}, err
	}
	for _, c := range clusters {
		if c.Name == name {
			return c, nil
		}
	}
	return Cluster{}, fmt.Errorf("Cluster %s nicht gefunden", name)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
