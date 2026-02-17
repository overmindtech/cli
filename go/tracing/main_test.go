package tracing

import (
	"testing"
)

func TestTracingResource(t *testing.T) {
	resource := tracingResource("test-component")
	if resource == nil {
		t.Error("Could not initialize tracing resource. Check the log!")
	}
}
