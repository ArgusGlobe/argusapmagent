package config

import (
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultHermesHost = "127.0.0.1"
	defaultHermesPort = "19090"
	defaultHealthPort = "8080"
	defaultEnv        = "local"
	defaultService    = "unknown-service"
	defaultInterval   = 30 * time.Second

	envHermesHost        = "HERMES_HOST"
	envHermesGRPCPort    = "HERMES_GRPC_PORT"
	envProbeEnvironment  = "PROBE_ENVIRONMENT"
	envProbeServiceName  = "PROBE_SERVICE_NAME"
	envProbeIntervalSecs = "PROBE_COLLECTION_INTERVAL_SECONDS"
	envProbeHealthPort   = "PROBE_HEALTH_PORT"
	envECSMetadataURIV4  = "ECS_CONTAINER_METADATA_URI_V4"
)

type Config struct {
	HermesHost         string
	HermesGRPCPort     string
	Environment        string
	ServiceName        string
	CollectionInterval time.Duration
	HealthPort         string
	MetadataURI        string
}

func Load() Config {
	return Config{
		HermesHost:         env(envHermesHost, defaultHermesHost),
		HermesGRPCPort:     env(envHermesGRPCPort, defaultHermesPort),
		Environment:        env(envProbeEnvironment, defaultEnv),
		ServiceName:        env(envProbeServiceName, defaultService),
		CollectionInterval: interval(env(envProbeIntervalSecs, "")),
		HealthPort:         port(env(envProbeHealthPort, defaultHealthPort)),
		MetadataURI:        os.Getenv(envECSMetadataURIV4),
	}
}

func (c Config) HermesAddress() string {
	return net.JoinHostPort(c.HermesHost, c.HermesGRPCPort)
}

func (c Config) HealthAddress() string {
	return net.JoinHostPort("", c.HealthPort)
}

func (c Config) HealthURL() string {
	return "http://" + net.JoinHostPort("127.0.0.1", c.HealthPort) + "/healthz"
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func interval(raw string) time.Duration {
	if raw == "" {
		return defaultInterval
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs < 5 {
		return defaultInterval
	}
	return time.Duration(secs) * time.Second
}

func port(raw string) string {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 65535 {
		return defaultHealthPort
	}
	return raw
}
