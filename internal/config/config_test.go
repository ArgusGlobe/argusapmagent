package config

import (
	"testing"
	"time"
)

func TestLoadUsesProbeAndHermesEnvNames(t *testing.T) {
	t.Setenv(envHermesHost, "hermes.internal")
	t.Setenv(envHermesGRPCPort, "19091")
	t.Setenv(envProbeEnvironment, "prod")
	t.Setenv(envProbeServiceName, "checkout")
	t.Setenv(envProbeIntervalSecs, "45")
	t.Setenv(envECSMetadataURIV4, "http://169.254.170.2/v4/task")

	cfg := Load()
	if cfg.HermesHost != "hermes.internal" {
		t.Fatalf("expected hermes host from env, got %q", cfg.HermesHost)
	}
	if cfg.HermesGRPCPort != "19091" {
		t.Fatalf("expected hermes port from env, got %q", cfg.HermesGRPCPort)
	}
	if cfg.Environment != "prod" {
		t.Fatalf("expected probe environment from env, got %q", cfg.Environment)
	}
	if cfg.ServiceName != "checkout" {
		t.Fatalf("expected probe service name from env, got %q", cfg.ServiceName)
	}
	if cfg.CollectionInterval != 45*time.Second {
		t.Fatalf("expected probe collection interval from env, got %s", cfg.CollectionInterval)
	}
	if cfg.MetadataURI != "http://169.254.170.2/v4/task" {
		t.Fatalf("expected metadata uri from env, got %q", cfg.MetadataURI)
	}
}

func TestIntervalFallback(t *testing.T) {
	if got := interval("bad"); got != defaultInterval {
		t.Fatalf("expected fallback interval, got %s", got)
	}
	if got := interval("2"); got != defaultInterval {
		t.Fatalf("expected minimum guard fallback, got %s", got)
	}
}
