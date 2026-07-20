package llm_client

import (
	"net/http"
	"strings"
	"testing"
)

func TestFormatLLMHTTPErrorUnsupportedModel(t *testing.T) {
	body := []byte(`{
    "error": {
        "code": "400",
        "message": "Unsupported model MiMo-V2.5"
    }
}`)
	err := formatLLMHTTPError(http.StatusBadRequest, "MiMo-V2.5", body)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "模型不支持") {
		t.Fatalf("want 模型不支持, got %q", msg)
	}
	if !strings.Contains(msg, "MiMo-V2.5") {
		t.Fatalf("want model name in message, got %q", msg)
	}
	if strings.Contains(msg, "\n") {
		t.Fatalf("error should be single line, got %q", msg)
	}
}
