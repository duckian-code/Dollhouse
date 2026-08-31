package observability

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestCorrelationIDPrefersHeaderAndLimitsLength(t *testing.T) {
	request := events.APIGatewayV2HTTPRequest{Headers: map[string]string{"X-Correlation-ID": strings.Repeat("a", 200)}}
	request.RequestContext.RequestID = "gateway-id"
	if got := CorrelationID(request); len(got) != maxCorrelationIDBytes {
		t.Fatalf("correlation ID length = %d", len(got))
	}
	request.Headers = nil
	if got := CorrelationID(request); got != "gateway-id" {
		t.Fatalf("correlation ID = %q", got)
	}
}

func TestWriteMetricsProducesEmbeddedMetricFormatWithoutSensitiveFields(t *testing.T) {
	var output bytes.Buffer
	err := WriteMetrics(&output, "test", RequestContext{CorrelationID: "correlation", RequestID: "request"},
		Metric{Name: "MoodUpdatesPublished", Value: 1, Unit: "Count"})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["environment"] != "test" || payload["MoodUpdatesPublished"] != float64(1) {
		t.Fatalf("payload = %#v", payload)
	}
	for _, forbidden := range []string{"status", "token", "body", "userId"} {
		if strings.Contains(output.String(), forbidden) {
			t.Fatalf("metric output contains %q: %s", forbidden, output.String())
		}
	}
}
