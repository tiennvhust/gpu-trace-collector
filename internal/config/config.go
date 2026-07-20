// Package config loads and validates the collector configuration.
//
// » Design choice: one YAML file, environment variables expanded inside it
// » (so secrets like API keys and SASL passwords come from the environment,
// » never from the file checked into git). This is the same pattern the
// » OpenTelemetry Collector uses: https://opentelemetry.io/docs/collector/configuration/
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SASL configures SASL/PLAIN authentication toward the Kafka endpoint.
//
// » Azure Event Hubs speaks the Kafka protocol on port 9093 with TLS +
// » SASL/PLAIN, username "$ConnectionString" and the connection string as the
// » password. Flip tls: true, sasl.enabled: true and this collector talks to
// » Event Hubs with no code change:
// » https://learn.microsoft.com/en-us/azure/event-hubs/azure-event-hubs-apache-kafka-overview
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
//
// » Multi-tenancy is THE core requirement of a shared ingestion platform:
// » identity (api_key), isolation (per-tenant rate limits), and attribution
// » (the tenant name stamped on every Kafka record). See how Cortex/Mimir
// » model tenants: https://grafana.com/docs/mimir/latest/references/architecture/about-tenant-ids/
type Tenant struct {
	Name         string  `yaml:"name"`
	APIKey       string  `yaml:"api_key"`
	EventsPerSec float64 `yaml:"events_per_sec"`
	Burst        int     `yaml:"burst"`
	APIKey2      string  `yaml:"api_key2"`
}

// Config is the root configuration.
type Config struct {
	GRPCListen         string   `yaml:"grpc_listen"`
	HTTPListen         string   `yaml:"http_listen"`
	QueueCapacity      int      `yaml:"queue_capacity"`
	Workers            int      `yaml:"workers"`
	MaxRecvMsgBytes    int      `yaml:"max_recv_msg_bytes"`
	Kafka              Kafka    `yaml:"kafka"`
	Tenants            []Tenant `yaml:"tenants"`
	GlobalEventsPerSec float64  `yaml:"global_events_per_sec"`
	GlobalBurst        int      `yaml:"global_burst"`
}

// Load reads path, expands ${ENV_VARS}, unmarshals, applies defaults and
// validates.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// » os.ExpandEnv turns "${COLLECTOR_KEY_DEV}" into the env value. An
	// » unset variable expands to "" — validation below catches empty keys.
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
		c.GRPCListen = ":4317" // » 4317 is the registered OTLP/gRPC port.
	}
	if c.HTTPListen == "" {
		c.HTTPListen = ":9464" // » 9464 is the conventional Prometheus exporter port.
	}
	if c.QueueCapacity <= 0 {
		c.QueueCapacity = 8192
	}
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.MaxRecvMsgBytes <= 0 {
		// » gRPC's default is already 4 MiB; we set it explicitly because an
		// » ingestion service must own this number, not inherit it. Oversized
		// » requests are rejected by grpc-go before our handler runs — the
		// » cheapest possible admission control.
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

	if c.GlobalEventsPerSec > 0 && c.GlobalBurst <= 0 {
		return errors.New("config: global_burst must be > 0 when global_events_per_sec is set")
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
		if t.APIKey2 != "" {
			if t.APIKey == t.APIKey2 {
				return fmt.Errorf("config: tenant %q: api_key2 must differ from api_key", t.Name)
			}
			if seen[t.APIKey2] {
				return fmt.Errorf("config: duplicate api_key2 for tenant %q", t.Name)
			}
			seen[t.APIKey2] = true
		}
		if t.EventsPerSec <= 0 {
			return fmt.Errorf("config: tenant %q: events_per_sec must be > 0", t.Name)
		}
		// » Subtle trap: rate.Limiter.AllowN(n) is permanently false when
		// » n > burst, so burst must be >= the largest single request a
		// » tenant may send (datapoints per export). The agent exports every
		// » 5s, so burst ≈ 2× events_per_sec × 5 is a sane floor.
		if t.Burst <= 0 {
			return fmt.Errorf("config: tenant %q: burst must be > 0", t.Name)
		}
	}
	return nil
}
