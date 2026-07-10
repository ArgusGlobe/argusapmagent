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
	defaultEnv        = "local"
	defaultService    = "unknown-service"
	defaultInterval   = 30 * time.Second
)

type Config struct {
	HermesHost         string
	HermesGRPCPort     string
	Environment        string
	ServiceName        string
	CollectionInterval time.Duration
	MetadataURI        string
}

func Load() Config {
	return Config{
		HermesHost:         env("ARGUS_HERMES_HOST", defaultHermesHost),
		HermesGRPCPort:     env("ARGUS_HERMES_GRPC_PORT", defaultHermesPort),
		Environment:        env("ARGUS_ENVIRONMENT", defaultEnv),
		ServiceName:        env("ARGUS_SERVICE_NAME", defaultService),
		CollectionInterval: interval(env("ARGUS_COLLECTION_INTERVAL_SECONDS", "")),
		MetadataURI:        os.Getenv("ECS_CONTAINER_METADATA_URI_V4"),
	}
}

func (c Config) HermesAddress() string {
	return net.JoinHostPort(c.HermesHost, c.HermesGRPCPort)
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
