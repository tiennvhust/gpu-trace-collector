// Package config loads and validates the collector configuration.
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SASL configures SASL/PLAIN authentication toward the Kafka endpoint.
type SASL struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Kafka configures the produce side.
type Kafka struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
	TLS     bool     `yaml:"tls"`
	SASL    SASL     `yaml:"sasl"`
}

// Tenant is one authenticated producer of telemetry.
type Tenant struct {
	Name         string  `yaml:"name"`
	APIKey       string  `yaml:"api_key"`
	EventsPerSec float64 `yaml:"events_per_sec"`
	Burst        int     `yaml:"burst"`
}

// Config is the root configuration.
type Config struct {
	GRPCListen      string   `yaml:"grpc_listen"`
	HTTPListen      string   `yaml:"http_listen"`
	QueueCapacity   int      `yaml:"queue_capacity"`
	Workers         int      `yaml:"workers"`
	MaxRecvMsgBytes int      `yaml:"max_recv_msg_bytes"`
	Kafka           Kafka    `yaml:"kafka"`
	Tenants         []Tenant `yaml:"tenants"`
}

// Load reads path, expands ${ENV_VARS}, unmarshals, applies defaults and
// validates.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.GRPCListen == "" {
		c.GRPCListen = ":4317"
	}
	if c.HTTPListen == "" {
		c.HTTPListen = ":9464"
	}
	if c.QueueCapacity <= 0 {
		c.QueueCapacity = 8192
	}
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.MaxRecvMsgBytes <= 0 {
		c.MaxRecvMsgBytes = 4 * 1024 * 1024
	}
	if c.Kafka.Topic == "" {
		c.Kafka.Topic = "telemetry.otlp"
	}
}

func (c *Config) validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return errors.New("config: kafka.brokers must not be empty")
	}
	if len(c.Tenants) == 0 {
		return errors.New("config: at least one tenant is required")
	}
	seen := map[string]bool{}
	for _, t := range c.Tenants {
		if t.Name == "" || t.APIKey == "" {
			return fmt.Errorf("config: tenant %q: name and api_key are required", t.Name)
		}
		if seen[t.APIKey] {
			return fmt.Errorf("config: duplicate api_key for tenant %q", t.Name)
		}
		seen[t.APIKey] = true
		if t.EventsPerSec <= 0 {
			return fmt.Errorf("config: tenant %q: events_per_sec must be > 0", t.Name)
		}
		if t.Burst <= 0 {
			return fmt.Errorf("config: tenant %q: burst must be > 0", t.Name)
		}
	}
	return nil
}
