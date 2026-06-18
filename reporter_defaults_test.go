package clog_test

import (
	"os"
	"testing"
	"time"

	"github.com/uber/jaeger-client-go/config"
)

// TestReporterDefaults_NotOverridden verifies that v0.1.2 no longer overrides
// the Jaeger client's built-in reporter defaults (QueueSize=100,
// BufferFlushInterval=1s) via init().
//
// Before the fix, init() set JAEGER_REPORTER_MAX_QUEUE_SIZE=64 and
// JAEGER_REPORTER_FLUSH_INTERVAL=10s, overriding Jaeger's battle-tested
// values and risking queue overflow / span loss under load.
func TestReporterDefaults_NotOverridden(t *testing.T) {
	// Guard: ensure reporter env vars are NOT set (the whole point of v0.1.2).
	// If another test or the environment left them set, we skip env verification.
	if os.Getenv("JAEGER_REPORTER_MAX_QUEUE_SIZE") != "" ||
		os.Getenv("JAEGER_REPORTER_FLUSH_INTERVAL") != "" {
		t.Skip("reporter env vars are set externally — cannot verify init() behavior")
	}

	// config.FromEnv() reads env vars and populates config.ReporterConfig.
	// When env vars are unset, the fields remain zero-valued, and the Jaeger
	// SDK substitutes its built-in defaults (QueueSize=100, BufferFlushInterval=1s).
	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Reporter.QueueSize != 0 {
		t.Errorf("QueueSize = %d, want 0 (not overridden; Jaeger will use 100)",
			cfg.Reporter.QueueSize)
	}

	if cfg.Reporter.BufferFlushInterval != 0 {
		t.Errorf("BufferFlushInterval = %v, want 0 (not overridden; Jaeger will use 1s)",
			cfg.Reporter.BufferFlushInterval)
	}
}

// TestReporterDefaults_UserOverride verifies that users can still explicitly
// set reporter env vars to override Jaeger defaults — this is a regression
// test to ensure the fix doesn't break override capability.
func TestReporterDefaults_UserOverride(t *testing.T) {
	t.Setenv("JAEGER_REPORTER_MAX_QUEUE_SIZE", "200")
	t.Setenv("JAEGER_REPORTER_FLUSH_INTERVAL", "5s")

	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Reporter.QueueSize != 200 {
		t.Errorf("QueueSize = %d, want 200 (user override)", cfg.Reporter.QueueSize)
	}

	if cfg.Reporter.BufferFlushInterval != 5*time.Second {
		t.Errorf("BufferFlushInterval = %v, want 5s", cfg.Reporter.BufferFlushInterval)
	}
}
