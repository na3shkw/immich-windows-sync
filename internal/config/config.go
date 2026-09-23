package config

import (
	"encoding/json"
	"immich-windows-sync/internal/appenv"
	"os"
	"path/filepath"
)

type ImmichConfig struct {
	ServerURL string `json:"serverURL"`
	APIKey    string `json:"apiKey"`
}

type Config struct {
	Immich          ImmichConfig `json:"immich"`
	TargetFolders   []string     `json:"targetFolders"`
	ExcludedFolders []string     `json:"excludedFolders"`
}

func getConfigPath() (string, error) {
	appDir, err := appenv.AppDir()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(appDir, "config.json")
	return configPath, nil
}

func Load() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func Save(config Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(path), 0644)
	if err != nil {
		return err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
