package kubeexec

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"multi-oc/internal/discovery"
	"multi-oc/internal/keystore"
)

// BuildOcAuthArgs builds authentication args for "oc" (without kubeconfig):
// --server, --token, and optionally --certificate-authority or --insecure-skip-tls-verify.
// Sources: Env (MOC_TARGET_TOKEN/CA_FILE/INSECURE) -> Keyring -> interactive prompt.
// Returns a cleanup function (removes temporary CA file if created).
func BuildOcAuthArgs(ctx context.Context, c discovery.Cluster) ([]string, func(), error) {
	_ = ctx
	if c.APIURL == "" {
		return nil, nil, fmt.Errorf("APIURL empty")
	}
	cleanup := func() {}

	// 1) Token from env -> Keyring -> prompt
	token := sanitizeToken(os.Getenv("MOC_TARGET_TOKEN"))
	if token == "" {
		if t, err := keystore.GetTargetToken(c.Name); err == nil && t != "" {
			token = sanitizeToken(t)
		}
	}
	if token == "" {
		// Hint URL for token retrieval
		hint := deriveOAuthTokenURL(c.APIURL)
		if hint != "" {
			fmt.Fprintf(os.Stderr, "No token found. Open in a browser (from any machine with access):\n  %s\nSign in there, copy the token (starting with 'sha256~') and paste it here.\n", hint)
		} else {
			fmt.Fprintln(os.Stderr, "No token found. Please get your 'oc login --token' from the OpenShift Web Console and paste it here (sha256~...).")
		}
		fmt.Fprint(os.Stderr, "Token: ")
		stdin := bufio.NewReader(os.Stdin)
		line, _ := stdin.ReadString('\n')
		token = sanitizeToken(line)
		if token == "" {
			return nil, nil, fmt.Errorf("no valid token detected")
		}
		_ = keystore.SetTargetToken(c.Name, token)
	}

	// 2) Determine TLS flags
	insecure := os.Getenv("MOC_TARGET_INSECURE") == "true"
	caFile := os.Getenv("MOC_TARGET_CA_FILE")

	args := []string{"--server", c.APIURL, "--token", token}

	if caFile != "" {
		args = append(args, "--certificate-authority", caFile)
	} else if len(c.CAData) > 0 {
		// Write CA to a temporary file
		tmpDir, err := os.MkdirTemp("", "moc-ca-*")
		if err != nil {
			return nil, nil, err
		}
		caPath := filepath.Join(tmpDir, "ca.crt")
		if err := os.WriteFile(caPath, c.CAData, 0o600); err != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, nil, err
		}
		cleanup = func() { _ = os.RemoveAll(tmpDir) }
		args = append(args, "--certificate-authority", caPath)
	} else if insecure {
		args = append(args, "--insecure-skip-tls-verify=true")
	}

	return args, cleanup, nil
}

func sanitizeToken(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`sha256~[A-Za-z0-9\-_\.]+`)
	if m := re.FindString(s); m != "" {
		return m
	}
	if i := strings.Index(s, "="); i >= 0 {
		cand := s[i+1:]
		cand = strings.TrimSpace(cand)
		cand = strings.Trim(cand, "'\"`")
		if m := re.FindString(cand); m != "" {
			return m
		}
		s = cand
	}
	s = strings.Trim(s, "'\"`")
	if re.MatchString(s) {
		return s
	}
	return s
}

// deriveOAuthTokenURL derives the OAuth token request URL from the API URL.
func deriveOAuthTokenURL(api string) string {
	u, err := url.Parse(api)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	var withoutAPI string
	switch {
	case strings.HasPrefix(host, "api."):
		withoutAPI = strings.TrimPrefix(host, "api.")
	case strings.HasPrefix(host, "api-int."):
		withoutAPI = strings.TrimPrefix(host, "api-int.")
	default:
		return ""
	}
	return "https://oauth-openshift.apps." + withoutAPI + "/oauth/token/request"
}
