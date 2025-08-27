package configstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	appDirName = "multi-oc"
	stateFile   = "state.json"
)

type state struct {
	HubURL string `json:"hubURL"`
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
	return filepath.Join(base, appDirName), nil
}

func SaveHub(hubURL string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	st := state{HubURL: hubURL}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stateFile), b, 0o600)
}

func LoadHub() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var st state
	if err := json.Unmarshal(b, &st); err != nil {
		return "", err
	}
	return st.HubURL, nil
}
