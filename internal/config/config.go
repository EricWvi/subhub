package config

import "time"

const DefaultRefreshInterval = 2 * time.Hour

type Config struct {
	ListenAddr             string
	DatabasePath           string
	UpstreamRequestTimeout time.Duration
	DefaultRefreshInterval time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:             ":8080",
		DatabasePath:           "data/subhub.db",
		UpstreamRequestTimeout: 15 * time.Second,
		DefaultRefreshInterval: DefaultRefreshInterval,
	}
}
