package clog_test

import (
	"testing"

	"github.com/nanjj/clog"
	"github.com/uber/jaeger-client-go/config"
)

func TestNewTracer(t *testing.T) {
	tcs := []struct {
		name string
	}{
		{"MyTracer01"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := clog.NewTracer(tc.name,
				config.Tag("runner", "127.0.0.1:54321"),
				config.Tag("leader", "127.0.0.1:54312"))
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Close()
			sp := tr.StartSpan("TestNewTracer")
			sp.Finish()
		})
	}
}
