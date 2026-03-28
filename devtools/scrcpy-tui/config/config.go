package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Preset struct {
	Name         string            `yaml:"name"`
	MaxResolution string           `yaml:"max_resolution,omitempty"`
	MaxFPS       int               `yaml:"max_fps,omitempty"`
	VideoCodec   string            `yaml:"video_codec,omitempty"`
	VideoBitRate string            `yaml:"video_bit_rate,omitempty"`
	AppBindings  map[string]string `yaml:"app_bindings,omitempty"` // Map app name to package name
	Args         []string          `yaml:"args,omitempty"`         // Additional raw arguments
}

type Config struct {
	Presets []Preset `yaml:"presets"`
}

func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exePath, err := os.Executable()
		if err == nil {
			path = filepath.Join(filepath.Dir(exePath), "config.yaml")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
