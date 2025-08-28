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

// BuildOcAuthArgs erzeugt die Auth-Argumente für "oc" (ohne Kubeconfig):
// --server, --token, optional --certificate-authority oder --insecure-skip-tls-verify.
// Sie nutzt: Env (MOC_TARGET_TOKEN/CA_FILE/INSECURE) -> Keyring -> interaktiven Prompt.
// Gibt zusätzlich eine Cleanup-Funktion zurück (löscht temporäre CA-Datei falls angelegt).
func BuildOcAuthArgs(ctx context.Context, c discovery.Cluster) ([]string, func(), error) {
	_ = ctx
	if c.APIURL == "" {
		return nil, nil, fmt.Errorf("APIURL leer")
	}
	cleanup := func() {}

	// 1) Token aus Env -> Keyring -> Prompt
	token := sanitizeToken(os.Getenv("MOC_TARGET_TOKEN"))
	if token == "" {
		if t, err := keystore.GetTargetToken(c.Name); err == nil && t != "" {
			token = sanitizeToken(t)
		}
	}
	if token == "" {
		// Hinweis-URL für Token-Beschaffung
		hint := deriveOAuthTokenURL(c.APIURL)
		if hint != "" {
			fmt.Fprintf(os.Stderr, "Kein Token gefunden. Öffne im Browser (auf einem beliebigen Rechner mit Zugriff):\n  %s\nMelde dich dort an, kopiere das Token (beginnend mit 'sha256~') und füge es hier ein.\n", hint)
		} else {
			fmt.Fprintln(os.Stderr, "Kein Token gefunden. Bitte besorge dir aus der OpenShift Web Console dein 'oc login --token' und füge es hier ein (sha256~...).")
		}
		fmt.Fprint(os.Stderr, "Token: ")
		stdin := bufio.NewReader(os.Stdin)
		line, _ := stdin.ReadString('\n')
		token = sanitizeToken(line)
		if token == "" {
			return nil, nil, fmt.Errorf("kein gültiges Token erkannt")
		}
		_ = keystore.SetTargetToken(c.Name, token)
	}

	// 2) TLS-Flags bestimmen
	insecure := os.Getenv("MOC_TARGET_INSECURE") == "true"
	caFile := os.Getenv("MOC_TARGET_CA_FILE")

	args := []string{"--server", c.APIURL, "--token", token}

	if caFile != "" {
		args = append(args, "--certificate-authority", caFile)
	} else if len(c.CAData) > 0 {
		// Schreibe CA in temporäre Datei
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

// deriveOAuthTokenURL versucht aus der API-URL die OAuth-Token-Request-URL abzuleiten.
func deriveOAuthTokenURL(api string) string {
	u, err := url.Parse(api)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	withoutAPI := strings.TrimPrefix(host, "api.")
	if withoutAPI == host {
		return ""
	}
	return "https://oauth-openshift.apps." + withoutAPI + "/oauth/token/request"
}
