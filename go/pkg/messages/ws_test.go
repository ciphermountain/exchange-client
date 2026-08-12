package messages

import "testing"

func TestParseOrderStatus_Aliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		status OrderStatus
	}{
		{name: "open lowercase", input: "open", status: Open},
		{name: "open uppercase", input: "OPEN", status: Open},
		{name: "partial underscore", input: "partially_filled", status: Partial},
		{name: "partial spaced", input: "partially filled", status: Partial},
		{name: "filled closed alias", input: "closed", status: Filled},
		{name: "filled completed alias", input: "completed", status: Filled},
		{name: "cancelled american", input: "canceled", status: Cancelled},
		{name: "cancelled partial alias", input: "cancelled_partial", status: Cancelled},
		{name: "expired alias", input: "expired", status: Cancelled},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, err := ParseOrderStatus(tt.input)
			if err != nil {
				t.Fatalf("ParseOrderStatus(%q) returned unexpected error: %v", tt.input, err)
			}

			if status != tt.status {
				t.Fatalf("ParseOrderStatus(%q) = %q, want %q", tt.input, status, tt.status)
			}
		})
	}
}

func TestParseOrderStatus_Unknown(t *testing.T) {
	t.Parallel()

	if _, err := ParseOrderStatus("nonsense"); err == nil {
		t.Fatal("expected error for unknown status")
	}
}
