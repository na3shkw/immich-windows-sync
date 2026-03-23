package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func getConfigPath() (string, error) {
	appdataDir := os.Getenv("APPDATA")
	if appdataDir == "" {
		return "", fmt.Errorf(`Environment variable "APPDATA" is empty.`)
	}
	configPath := filepath.Join(appdataDir, "immich-sync", "config.json")
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
