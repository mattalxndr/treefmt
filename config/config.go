package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

// Config is used to represent the list of configured Formatters.
type Config struct {
	Global struct {
		// BatchSize controls the maximum number of paths to batch before applying them to a sequence of formatters.
		BatchSize int `toml:"batch_size"`
		// Excludes is an optional list of glob patterns used to exclude certain files from all formatters.
		Excludes []string `toml:"excludes"`
	} `toml:"global"`
	Formatters map[string]*Formatter `toml:"formatter"`
}

// ReadFile reads from path and unmarshals toml into a Config instance.
func ReadFile(path string, names []string) (cfg *Config, err error) {
	if _, err = toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// filter formatters based on provided names
	if len(names) > 0 {
		filtered := make(map[string]*Formatter)

		// check if the provided names exist in the config
		for _, name := range names {
			formatterCfg, ok := cfg.Formatters[name]
			if !ok {
				return nil, fmt.Errorf("formatter %v not found in config", name)
			}
			filtered[name] = formatterCfg
		}

		// updated formatters
		cfg.Formatters = filtered
	}

	return
}
