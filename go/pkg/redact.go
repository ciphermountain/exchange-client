package xifer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var secretJSONKeys = map[string]struct{}{
	"token":         {},
	"api_key":       {},
	"apikey":        {},
	"key":           {},
	"secret":        {},
	"password":      {},
	"signature":     {},
	"authorization": {},
}

func redactMessage(message []byte) string {
	decoder := json.NewDecoder(bytes.NewReader(message))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Sprintf("[invalid JSON message length=%d]", len(message))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Sprintf("[invalid JSON message length=%d]", len(message))
	}

	redactJSONValue(value)
	redacted, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("[invalid JSON message length=%d]", len(message))
	}

	return string(redacted)
}

func redactJSONValue(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, nested := range value {
			if _, secret := secretJSONKeys[strings.ToLower(key)]; secret {
				value[key] = "[REDACTED]"
				continue
			}
			redactJSONValue(nested)
		}
	case []any:
		for _, nested := range value {
			redactJSONValue(nested)
		}
	}
}
