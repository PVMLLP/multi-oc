package identity

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"multi-oc/internal/configstate"

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

// EnsureHubLogin stellt sicher, dass eine gültige oc-Session zum Hub besteht.
// Falls keine Hub-URL konfiguriert ist, wird sie interaktiv abgefragt und gespeichert.
// Es wird standardmäßig der Browser-Flow genutzt (--web). Optional kann das Verhalten
// über Umgebungsvariablen beeinflusst werden:
//   MOC_HUB_INSECURE=true    → --insecure-skip-tls-verify
//   MOC_HUB_CA_FILE=/path    → --certificate-authority
func EnsureHubLogin(ctx context.Context) error {
	hubURL, err := configstate.LoadHub()
	if err != nil {
		return err
	}
	if strings.TrimSpace(hubURL) == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprint(os.Stderr, "Hub API URL (z. B. https://api.hub.example:6443): ")
		line, _ := reader.ReadString('\n')
		hubURL = strings.TrimSpace(line)
		hubURL = strings.TrimPrefix(hubURL, "@")
		if hubURL == "" {
			return fmt.Errorf("Hub API URL erforderlich")
		}
		if err := configstate.SaveHub(hubURL); err != nil {
			return err
		}
	}
	insecure := os.Getenv("MOC_HUB_INSECURE") == "true"
	caFile := os.Getenv("MOC_HUB_CA_FILE")
	// Browser-Flow (token="")
	if err := LoginHub(ctx, hubURL, insecure, caFile, ""); err != nil {
		return err
	}
	return nil
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
