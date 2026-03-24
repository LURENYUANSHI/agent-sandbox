package trace

import (
	"encoding/json"
	"fmt"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// OTelSpan is an OpenTelemetry-compatible span representation.
type OTelSpan struct {
	TraceID    string            `json:"traceId"`
	SpanID     string            `json:"spanId"`
	Name       string            `json:"name"`
	StartTime  int64             `json:"startTimeUnixNano"`
	EndTime    int64             `json:"endTimeUnixNano"`
	Attributes map[string]string `json:"attributes"`
	Status     string            `json:"status"`
}

// ExportToOTel converts trace events to OpenTelemetry-compatible spans.
func ExportToOTel(events []*types.TraceEvent) []OTelSpan {
	spans := make([]OTelSpan, 0, len(events))
	for _, e := range events {
		span := OTelSpan{
			TraceID:   e.TraceID,
			SpanID:    e.ID,
			Name:      fmt.Sprintf("%s.%s", e.Action.Type, e.Action.FileOp),
			StartTime: e.StartTime.UnixNano(),
			EndTime:   e.EndTime.UnixNano(),
			Attributes: map[string]string{
				"sandbox.id":      e.SandboxID,
				"action.type":     string(e.Action.Type),
				"action.path":     e.Action.Path,
				"action.file_op":  string(e.Action.FileOp),
				"action.host":     e.Action.Host,
				"action.command":  e.Action.Command,
				"decision":        string(e.Decision),
				"decision.reason": e.Reason,
			},
			Status: string(e.Decision),
		}
		spans = append(spans, span)
	}
	return spans
}

// ExportToJSON serializes spans as JSON bytes.
func ExportToJSON(events []*types.TraceEvent) ([]byte, error) {
	spans := ExportToOTel(events)
	return json.MarshalIndent(spans, "", "  ")
}
