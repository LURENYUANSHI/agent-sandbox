package trace

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// OTLP JSON structures for span export.
// These follow the OpenTelemetry Protocol JSON encoding.

// OTLPExport is the top-level OTLP export envelope.
type OTLPExport struct {
	ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

// ResourceSpans groups spans by resource.
type ResourceSpans struct {
	Resource  Resource     `json:"resource"`
	ScopeSpans []ScopeSpans `json:"scopeSpans"`
}

// Resource describes the entity producing telemetry.
type Resource struct {
	Attributes []KeyValue `json:"attributes"`
}

// ScopeSpans groups spans by instrumentation scope.
type ScopeSpans struct {
	Scope InstrumentationScope `json:"scope"`
	Spans []OTLPSpan           `json:"spans"`
}

// InstrumentationScope identifies the instrumentation library.
type InstrumentationScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// OTLPSpan represents a single span in OTLP format.
type OTLPSpan struct {
	TraceID           string     `json:"traceId"`
	SpanID            string     `json:"spanId"`
	ParentSpanID      string     `json:"parentSpanId,omitempty"`
	Name              string     `json:"name"`
	Kind              int        `json:"kind"`
	StartTimeUnixNano int64      `json:"startTimeUnixNano"`
	EndTimeUnixNano   int64      `json:"endTimeUnixNano"`
	Attributes        []KeyValue `json:"attributes,omitempty"`
	Status            SpanStatus `json:"status"`
}

// SpanStatus represents the status of a span.
type SpanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// KeyValue is an OTLP attribute key-value pair.
type KeyValue struct {
	Key   string     `json:"key"`
	Value AttrValue  `json:"value"`
}

// AttrValue holds a typed attribute value.
type AttrValue struct {
	StringValue string `json:"stringValue,omitempty"`
	IntValue    int64  `json:"intValue,omitempty"`
	BoolValue   bool   `json:"boolValue,omitempty"`
}

// OTLP span kind constants.
const (
	SpanKindInternal = 1
)

// OTLP status code constants.
const (
	StatusCodeUnset = 0
	StatusCodeOK    = 1
	StatusCodeError = 2
)

// ExportToOTLP converts trace events into OTLP JSON bytes.
func ExportToOTLP(events []*types.TraceEvent) ([]byte, error) {
	spans := make([]OTLPSpan, 0, len(events))

	// Use the first event's sandbox ID as trace ID, or generate one.
	traceID := "00000000000000000000000000000000"
	if len(events) > 0 && len(events[0].SandboxID) >= 16 {
		traceID = hex.EncodeToString([]byte(events[0].SandboxID))
		if len(traceID) > 32 {
			traceID = traceID[:32]
		}
		for len(traceID) < 32 {
			traceID += "0"
		}
	}

	for _, event := range events {
		span := eventToSpan(event, traceID)
		spans = append(spans, span)
	}

	export := OTLPExport{
		ResourceSpans: []ResourceSpans{
			{
				Resource: Resource{
					Attributes: []KeyValue{
						{Key: "service.name", Value: AttrValue{StringValue: "agent-sandbox"}},
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Scope: InstrumentationScope{
							Name:    "agent-sandbox/trace",
							Version: "0.1.0",
						},
						Spans: spans,
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling OTLP export: %w", err)
	}
	return data, nil
}

func eventToSpan(event *types.TraceEvent, traceID string) OTLPSpan {
	spanID := event.ID
	if len(spanID) > 16 {
		spanID = spanID[:16]
	}
	for len(spanID) < 16 {
		spanID += "0"
	}

	parentSpanID := ""
	if event.ParentID != "" {
		parentSpanID = event.ParentID
		if len(parentSpanID) > 16 {
			parentSpanID = parentSpanID[:16]
		}
		for len(parentSpanID) < 16 {
			parentSpanID += "0"
		}
	}

	endTime := event.Timestamp.Add(time.Duration(event.DurationNs))

	status := SpanStatus{Code: StatusCodeUnset}
	if event.Result != nil {
		if event.Result.Success {
			status = SpanStatus{Code: StatusCodeOK}
		} else {
			status = SpanStatus{Code: StatusCodeError, Message: event.Result.Error}
		}
	}

	attrs := []KeyValue{
		{Key: "sandbox.id", Value: AttrValue{StringValue: event.SandboxID}},
		{Key: "event.type", Value: AttrValue{StringValue: string(event.EventType)}},
	}

	if event.Action != nil {
		attrs = append(attrs, KeyValue{
			Key: "action.type", Value: AttrValue{StringValue: string(event.Action.Type)},
		})
	}

	if event.PolicyDecision != nil {
		attrs = append(attrs, KeyValue{
			Key: "policy.allowed", Value: AttrValue{BoolValue: event.PolicyDecision.Allowed},
		})
		if event.PolicyDecision.Rule != "" {
			attrs = append(attrs, KeyValue{
				Key: "policy.rule", Value: AttrValue{StringValue: event.PolicyDecision.Rule},
			})
		}
	}

	for k, v := range event.Attributes {
		attrs = append(attrs, KeyValue{
			Key: "custom." + k, Value: AttrValue{StringValue: v},
		})
	}

	name := string(event.EventType)
	if event.Action != nil {
		name = fmt.Sprintf("%s.%s", event.Action.Type, event.EventType)
	}

	return OTLPSpan{
		TraceID:           traceID,
		SpanID:            spanID,
		ParentSpanID:      parentSpanID,
		Name:              name,
		Kind:              SpanKindInternal,
		StartTimeUnixNano: event.Timestamp.UnixNano(),
		EndTimeUnixNano:   endTime.UnixNano(),
		Attributes:        attrs,
		Status:            status,
	}
}
