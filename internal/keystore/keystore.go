package keystore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	keyring "github.com/zalando/go-keyring"
)

const (
	serviceTargetToken = "multi-oc-target-token"
)

func GetTargetToken(clusterName string) (string, error) {
	// Zuerst Keyring versuchen
	tok, err := keyring.Get(serviceTargetToken, clusterName)
	if err == nil && tok != "" {
		return tok, nil
	}
	// Datei-Fallback
	return readTokenFromFile(clusterName)
}

func SetTargetToken(clusterName, token string) error {
	// Versuche Keyring
	if err := keyring.Set(serviceTargetToken, clusterName, token); err == nil {
		return nil
	}
	// Datei-Fallback
	return writeTokenToFile(clusterName, token)
}

func DeleteTargetToken(clusterName string) error {
	_ = keyring.Delete(serviceTargetToken, clusterName)
	// Datei-Fallback löschen
	path, err := tokenFilePath(clusterName)
	if err != nil {
		return nil
	}
	_ = os.Remove(path)
	return nil
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

func tokenFilePath(clusterName string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	toks := filepath.Join(dir, "tokens")
	if err := os.MkdirAll(toks, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(toks, fmt.Sprintf("%s.token", clusterName)), nil
}

func writeTokenToFile(clusterName, token string) error {
	path, err := tokenFilePath(clusterName)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

func readTokenFromFile(clusterName string) (string, error) {
	path, err := tokenFilePath(clusterName)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(trimNewline(b)), nil
}

func trimNewline(b []byte) []byte {
	if len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		return b[:len(b)-1]
	}
	return b
}
