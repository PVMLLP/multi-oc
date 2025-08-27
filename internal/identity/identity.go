package identity

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	keyring "github.com/zalando/go-keyring"
	"multi-oc/internal/configstate"
)

const (
	serviceHubToken = "multi-oc-hub-refresh-token"
)

// LoginHub führt ein oc web-Login gegen den Hub durch und überlässt die Token-Verwaltung der oc-Kubeconfig.
func LoginHub(ctx context.Context, hubURL string) error {
	if hubURL == "" {
		return fmt.Errorf("hubURL leer")
	}
	cmd := exec.CommandContext(ctx, "oc", "login", "--server", hubURL, "--web")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func GetHubRefreshToken() (string, string, error) {
	hubURL, err := configstate.LoadHub()
	if err != nil {
		return "", "", err
	}
	if hubURL == "" {
		return "", "", errors.New("kein Hub konfiguriert; bitte 'moc login --hub <url>' ausführen")
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
