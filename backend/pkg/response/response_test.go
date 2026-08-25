package response

import (
	"encoding/json"
	"testing"
)

func TestErrorReturnsStructuredJSON(t *testing.T) {
	got := Error(400, "validation_failed", "invalid request")
	if got.StatusCode != 400 {
		t.Fatalf("StatusCode = %d, want 400", got.StatusCode)
	}
	if got.Headers["content-type"] != "application/json" {
		t.Fatalf("content-type = %q", got.Headers["content-type"])
	}

	var body ErrorBody
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "validation_failed" {
		t.Fatalf("code = %q", body.Error.Code)
	}
}
