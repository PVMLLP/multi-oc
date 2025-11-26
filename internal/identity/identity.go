package identity

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"multi-oc/internal/configstate"

	"regexp"

	keyring "github.com/zalando/go-keyring"
)

const (
	serviceHubToken = "multi-oc-hub-refresh-token"
)

// LoginHub runs an oc login.
// If token != "", it performs a headless login via --token; otherwise it uses the browser flow (--web).
func LoginHub(ctx context.Context, hubURL string, insecure bool, caFile string, token string) error {
	if hubURL == "" {
		return fmt.Errorf("hubURL is empty")
	}
	args := []string{"login", "--server", hubURL}
	if token != "" {
		args = append(args, "--token", token)
	} else {
		args = append(args, "--web")
	}
	if insecure {
		args = append(args, "--insecure-skip-tls-verify=true")
	}
	if caFile != "" {
		args = append(args, "--certificate-authority", caFile)
	}
	cmd := exec.CommandContext(ctx, "oc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EnsureHubLogin ensures there is a valid oc session to the hub.
// If no hub URL is configured, it prompts for it and saves it.
// It uses the browser flow by default (--web). Behavior can be influenced via env:
//
//	MOC_HUB_INSECURE=true    → --insecure-skip-tls-verify
//	MOC_HUB_CA_FILE=/path    → --certificate-authority
func EnsureHubLogin(ctx context.Context) error {
	hubURL, err := configstate.LoadHub()
	if err != nil {
		return err
	}
	if strings.TrimSpace(hubURL) == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprint(os.Stderr, "Hub API URL (e.g., https://api.hub.example:6443): ")
		line, _ := reader.ReadString('\n')
		hubURL = strings.TrimSpace(line)
		hubURL = strings.TrimPrefix(hubURL, "@")
		if hubURL == "" {
			return fmt.Errorf("hub API URL is required")
		}
		if err := configstate.SaveHub(hubURL); err != nil {
			return err
		}
	}
	insecure := os.Getenv("MOC_HUB_INSECURE") == "true"
	caFile := os.Getenv("MOC_HUB_CA_FILE")
	useWeb := os.Getenv("MOC_HUB_USE_WEB") == "true"
	if useWeb {
		// Browser flow
		return LoginHub(ctx, hubURL, insecure, caFile, "")
	}
	// Headless flow: print OAuth token URL and prompt for token, then use --token
	hint := deriveOAuthTokenURL(hubURL)
	if hint != "" {
		fmt.Fprintf(os.Stderr, "Open in a browser (from any machine with access):\n  %s\nSign in there, copy the token (starting with 'sha256~') and paste it here.\n", hint)
	} else {
		fmt.Fprintln(os.Stderr, "OpenShift token URL could not be derived from the hub API URL. Please obtain an OAuth token from the OpenShift Web Console and paste it here (sha256~...).")
	}
	fmt.Fprint(os.Stderr, "Token: ")
	stdin := bufio.NewReader(os.Stdin)
	line, _ := stdin.ReadString('\n')
	token := sanitizeToken(line)
	if token == "" {
		return fmt.Errorf("no valid token detected")
	}
	return LoginHub(ctx, hubURL, insecure, caFile, token)
}

func GetHubRefreshToken() (string, string, error) {
	hubURL, err := configstate.LoadHub()
	if err != nil {
		return "", "", err
	}
	if hubURL == "" {
		return "", "", errors.New("no hub configured; run 'moc login --hub <url>'")
	}
	tok, err := keyring.Get(serviceHubToken, hubURL)
	if err != nil {
		return "", "", err
	}
	return hubURL, tok, nil
}

func LogoutHub() error {
	hubURL, err := configstate.LoadHub()
	if err != nil || hubURL == "" {
		return nil
	}
	_ = keyring.Delete(serviceHubToken, hubURL)
	return nil
}

// sanitizeToken extracts a valid OpenShift token if present, tolerating various wrapper text.
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

// deriveOAuthTokenURL derives the OAuth token display URL from an OpenShift API URL.
// Supports api.<base> and api-int.<base> patterns.
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
	return "https://oauth-openshift.apps." + withoutAPI + "/oauth/token/display"
}
