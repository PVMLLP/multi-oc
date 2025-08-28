package identity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

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
