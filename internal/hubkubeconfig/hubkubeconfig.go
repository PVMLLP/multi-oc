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

type Options struct {
	Force   bool
	Verbose bool
	MSAName string
}

// WriteClusterKubeconfig fetches the admin-kubeconfig Secret from the hub for the given cluster
// and writes it to ~/.config/multi-oc/kubeconfigs/<cluster>.kubeconfig.
// Returns true if a kubeconfig was written.
func WriteClusterKubeconfig(ctx context.Context, c discovery.Cluster, opts Options) (bool, error) {
	if c.Name == "" {
		return false, fmt.Errorf("cluster name is empty")
	}
	if err := identity.EnsureHubLogin(ctx); err != nil {
		return false, err
	}
	target, err := defaultPath(c.Name)
	if err != nil {
		return false, err
	}
	// Skip if exists and not forcing
	if !opts.Force {
		if st, err := os.Stat(target); err == nil && !st.IsDir() && st.Size() > 0 {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[%s] kubeconfig already exists → skip: %s\n", c.Name, target)
			}
			return false, nil
		}
	}
	// Try multiple sources on the hub
	var raw []byte
	if b, _ := getSecretKubeconfig(ctx, c.Name, "admin-kubeconfig", opts); len(b) > 0 {
		raw = b
	} else if b, _ := getSecretKubeconfig(ctx, c.Name, c.Name+"-admin-kubeconfig", opts); len(b) > 0 {
		raw = b
	} else if b, _ := getSecretKubeconfig(ctx, c.Name, "managedcluster-info", opts); len(b) > 0 {
		raw = b
	} else if b, _ := findHiveKubeconfigAnyNamespace(ctx, c.Name, opts); len(b) > 0 {
		raw = b
	} else if kc, _ := getKubeconfigFromManagedServiceAccount(ctx, c, opts); len(kc) > 0 {
		raw = kc
	} else {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[%s] no kubeconfig secret found on hub (checked admin-kubeconfig, managedcluster-info, hive)\n", c.Name)
		}
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return false, err
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		return false, err
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[%s] wrote kubeconfig → %s\n", c.Name, target)
	}
	return true, nil
}

// WriteAllKubeconfigs attempts to fetch and write kubeconfigs for all managed clusters.
func WriteAllKubeconfigs(ctx context.Context, opts Options) (int, error) {
	clusters, err := discovery.ListManagedClusters(ctx)
	if err != nil {
		return 0, err
	}
	written := 0
	for _, c := range clusters {
		ok, err := WriteClusterKubeconfig(ctx, c, opts)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[%s] error: %v\n", c.Name, err)
			}
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

// getKubeconfigFromManagedServiceAccount tries to assemble a kubeconfig from a ManagedServiceAccount-issued token secret.
// It looks for common secret names <msaName>-token and managedserviceaccount-<msaName>-token in the cluster namespace,
// expecting keys "token" and "ca.crt". It then constructs a kubeconfig using the cluster's API URL.
func getKubeconfigFromManagedServiceAccount(ctx context.Context, c discovery.Cluster, opts Options) ([]byte, error) {
	name := opts.MSAName
	if name == "" {
		name = "moc"
	}
	// candidate secrets
	candidates := []string{
		name + "-token",
		"managedserviceaccount-" + name + "-token",
	}
	for _, sec := range candidates {
		tok, ca, err := readTokenSecret(ctx, c.Name, sec, opts)
		if err != nil || len(tok) == 0 || len(ca) == 0 {
			continue
		}
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[%s] using MSA token from secret %s\n", c.Name, sec)
		}
		kc := buildKubeconfigYAML(c.APIURL, ca, string(tok))
		return kc, nil
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[%s] no ManagedServiceAccount token secret found (tried %s)\n", c.Name, candidates)
	}
	return nil, fmt.Errorf("msa token not found")
}

func readTokenSecret(ctx context.Context, namespace, secretName string, opts Options) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, "oc", "get", "secret", secretName, "-n", namespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[%s] secret %s not found\n", namespace, secretName)
		}
		return nil, nil, err
	}
	var s secret
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, nil, err
	}
	tokB64, okTok := s.Data["token"]
	caB64, okCA := s.Data["ca.crt"]
	if !okTok || !okCA {
		return nil, nil, fmt.Errorf("secret %s/%s missing token or ca.crt", namespace, secretName)
	}
	tok, err := base64.StdEncoding.DecodeString(tokB64)
	if err != nil {
		return nil, nil, err
	}
	ca, err := base64.StdEncoding.DecodeString(caB64)
	if err != nil {
		return nil, nil, err
	}
	return tok, ca, nil
}

func buildKubeconfigYAML(server string, ca []byte, token string) []byte {
	caB64 := base64.StdEncoding.EncodeToString(ca)
	type (
		Cluster struct {
			Name    string                 `json:"name"`
			Cluster map[string]string      `json:"cluster"`
		}
		User struct {
			Name string                 `json:"name"`
			User map[string]string      `json:"user"`
		}
		Context struct {
			Name    string                 `json:"name"`
			Context map[string]string      `json:"context"`
		}
		Kubeconfig struct {
			APIVersion     string     `json:"apiVersion"`
			Kind           string     `json:"kind"`
			Clusters       []Cluster  `json:"clusters"`
			AuthInfos      []User     `json:"users"`
			Contexts       []Context  `json:"contexts"`
			CurrentContext string     `json:"current-context"`
		}
	)
	kc := Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{{
			Name: "cluster",
			Cluster: map[string]string{
				"server":                   server,
				"certificate-authority-data": caB64,
			},
		}},
		AuthInfos: []User{{
			Name: "user",
			User: map[string]string{
				"token": token,
			},
		}},
		Contexts: []Context{{
			Name: "ctx",
			Context: map[string]string{
				"cluster": "cluster",
				"user":    "user",
			},
		}},
		CurrentContext: "ctx",
	}
	// marshal to JSON; oc supports both, but we can emit JSON for simplicity
	b, _ := json.MarshalIndent(kc, "", "  ")
	return b
}
func getSecretKubeconfig(ctx context.Context, namespace, secretName string, opts Options) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "oc", "get", "secret", secretName, "-n", namespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[%s] secret %s not found\n", namespace, secretName)
		}
		return nil, err
	}
	var s secret
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	enc, ok := s.Data["kubeconfig"]
	if !ok || enc == "" {
		return nil, fmt.Errorf("secret %s/%s missing 'kubeconfig' key", namespace, secretName)
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func findHiveKubeconfigAnyNamespace(ctx context.Context, clusterName string, opts Options) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "oc", "get", "secret", "-A", "-l", "hive.openshift.io/cluster-deployment-name="+clusterName, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[%s] hive label scan failed: %v\n", clusterName, err)
		}
		return nil, err
	}
	var list struct {
		Items []secret `json:"items"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, err
	}
	for _, it := range list.Items {
		if enc, ok := it.Data["kubeconfig"]; ok && enc != "" {
			raw, decErr := base64.StdEncoding.DecodeString(enc)
			if decErr == nil {
				return raw, nil
			}
		}
	}
	return nil, fmt.Errorf("no hive kubeconfig secret found")
}


