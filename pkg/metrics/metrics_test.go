package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func resetRegistry() {
	// Use a fresh registry for each test to avoid double-registration panics.
	reg := prometheus.NewRegistry()

	SandboxesActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agentsandbox_sandboxes_active",
		Help: "Number of currently active sandboxes",
	})
	ActionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agentsandbox_actions_total",
		Help: "Total actions executed",
	}, []string{"action_type", "effect"})
	ActionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "agentsandbox_action_duration_seconds",
		Help: "Action execution duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"action_type"})
	PolicyEvaluations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agentsandbox_policy_evaluations_total",
		Help: "Total policy evaluations",
	}, []string{"effect"})
	APIRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agentsandbox_api_requests_total",
		Help: "Total API requests",
	}, []string{"method", "path", "status"})
	APILatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "agentsandbox_api_latency_seconds",
		Help: "API request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	reg.MustRegister(
		SandboxesActive,
		ActionsTotal,
		ActionDuration,
		PolicyEvaluations,
		APIRequests,
		APILatency,
	)
}

func TestMetricsRegistered(t *testing.T) {
	resetRegistry()
	// If we reach here without panic, all metrics are registered successfully.
}

func TestCounterIncrements(t *testing.T) {
	resetRegistry()

	ActionsTotal.WithLabelValues("file.read", "allowed").Inc()
	ActionsTotal.WithLabelValues("file.read", "allowed").Inc()
	ActionsTotal.WithLabelValues("network.connect", "denied").Inc()

	var m dto.Metric

	if err := ActionsTotal.WithLabelValues("file.read", "allowed").Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if got := m.GetCounter().GetValue(); got != 2 {
		t.Errorf("ActionsTotal(file.read, allowed) = %v, want 2", got)
	}

	if err := ActionsTotal.WithLabelValues("network.connect", "denied").Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if got := m.GetCounter().GetValue(); got != 1 {
		t.Errorf("ActionsTotal(network.connect, denied) = %v, want 1", got)
	}

	PolicyEvaluations.WithLabelValues("allow").Inc()
	PolicyEvaluations.WithLabelValues("allow").Inc()
	PolicyEvaluations.WithLabelValues("deny").Inc()

	if err := PolicyEvaluations.WithLabelValues("allow").Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if got := m.GetCounter().GetValue(); got != 2 {
		t.Errorf("PolicyEvaluations(allow) = %v, want 2", got)
	}
}

func TestHistogramRecordsValues(t *testing.T) {
	resetRegistry()

	ActionDuration.WithLabelValues("shell.exec").Observe(0.1)
	ActionDuration.WithLabelValues("shell.exec").Observe(0.5)
	ActionDuration.WithLabelValues("shell.exec").Observe(1.0)

	var m dto.Metric
	observer := ActionDuration.WithLabelValues("shell.exec")
	histogram := observer.(prometheus.Histogram)
	if err := histogram.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	h := m.GetHistogram()
	if got := h.GetSampleCount(); got != 3 {
		t.Errorf("ActionDuration sample count = %v, want 3", got)
	}
	if got := h.GetSampleSum(); got != 1.6 {
		t.Errorf("ActionDuration sample sum = %v, want 1.6", got)
	}
}

func TestGaugeIncrementDecrement(t *testing.T) {
	resetRegistry()

	SandboxesActive.Inc()
	SandboxesActive.Inc()
	SandboxesActive.Dec()

	var m dto.Metric
	if err := SandboxesActive.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if got := m.GetGauge().GetValue(); got != 1 {
		t.Errorf("SandboxesActive = %v, want 1", got)
	}
}
