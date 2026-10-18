package xifer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactMessageNestedSecrets(t *testing.T) {
	input := []byte(`{"safe":"unchanged","Token":"apikey|abc","items":[{"aPiKeY":"first","api_key":"second","KEY":3,"secret":true,"Password":"pass","signature":"sig","Authorization":"Bearer value"}]}`)

	redacted := redactMessage(input)
	for _, secret := range []string{"apikey|abc", "first", "second", "pass", `"sig"`, "Bearer value"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted message contains secret %q: %s", secret, redacted)
		}
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(redacted), &got); err != nil {
		t.Fatalf("redacted message is not valid JSON: %v", err)
	}
	if got["safe"] != "unchanged" {
		t.Errorf("safe field = %v, want unchanged", got["safe"])
	}
	if got["Token"] != "[REDACTED]" {
		t.Errorf("Token = %v, want [REDACTED]", got["Token"])
	}
	items := got["items"].([]any)
	nested := items[0].(map[string]any)
	for _, key := range []string{"aPiKeY", "api_key", "KEY", "secret", "Password", "signature", "Authorization"} {
		if nested[key] != "[REDACTED]" {
			t.Errorf("%s = %v, want [REDACTED]", key, nested[key])
		}
	}
}

func TestRedactMessageInvalidJSON(t *testing.T) {
	for _, input := range [][]byte{[]byte(`{"token":"SECRET"`), []byte(`{"safe":"ok"} trailing SECRET`)} {
		redacted := redactMessage(input)
		if strings.Contains(redacted, "SECRET") || strings.Contains(redacted, string(input)) {
			t.Fatalf("invalid JSON output exposes input: %s", redacted)
		}
		if !strings.Contains(redacted, "length=") {
			t.Errorf("invalid JSON output does not report message length: %s", redacted)
		}
	}
}
