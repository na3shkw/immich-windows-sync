package config

type ImmichConfig struct {
	ServerURL string `json:"serverURL"`
	APIKey    string `json:"apiKey"`
}

type Config struct {
	Immich        ImmichConfig `json:"immich"`
	TargetFolders []string     `json:"targetFolders"`
}
