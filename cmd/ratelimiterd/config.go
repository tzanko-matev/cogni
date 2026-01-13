package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// config describes the ratelimiterd YAML configuration.
type config struct {
	Server struct {
		ListenAddr string `yaml:"listen_addr"`
		Backend    string `yaml:"backend"`
	} `yaml:"server"`
	Registry struct {
		Path string `yaml:"path"`
	} `yaml:"registry"`
	TigerBeetle struct {
		ClusterID           string   `yaml:"cluster_id"`
		Addresses           []string `yaml:"addresses"`
		Sessions            int      `yaml:"sessions"`
		MaxBatchEvents      int      `yaml:"max_batch_events"`
		FlushIntervalMicros int      `yaml:"flush_interval_micros"`
	} `yaml:"tigerbeetle"`
}

// loadConfig reads and validates the configuration file.
func loadConfig(path string) (config, error) {
	var cfg config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = ":8080"
	}
	if cfg.Server.Backend == "" {
		cfg.Server.Backend = "memory"
	}
	if cfg.Registry.Path == "" {
		return cfg, fmt.Errorf("registry.path is required")
	}
	if cfg.Server.Backend == "tigerbeetle" {
		if len(cfg.TigerBeetle.Addresses) == 0 {
			return cfg, fmt.Errorf("tigerbeetle.addresses is required")
		}
		if cfg.TigerBeetle.ClusterID == "" {
			cfg.TigerBeetle.ClusterID = "0"
		}
	}
	return cfg, nil
}

// parseClusterID converts a config cluster_id string to uint32.
func parseClusterID(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid cluster_id: %w", err)
	}
	return uint32(parsed), nil
}

// flushInterval converts microsecond config to a duration.
func flushInterval(cfg config) time.Duration {
	if cfg.TigerBeetle.FlushIntervalMicros <= 0 {
		return 0
	}
	return time.Duration(cfg.TigerBeetle.FlushIntervalMicros) * time.Microsecond
}
