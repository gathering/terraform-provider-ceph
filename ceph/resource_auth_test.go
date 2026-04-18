package ceph

import (
	"fmt"
	"strings"
	"testing"
)

func TestToCapsArray(t *testing.T) {
	tests := []struct {
		name    string
		caps    map[string]interface{}
		wantLen int
	}{
		{
			name:    "empty",
			caps:    map[string]interface{}{},
			wantLen: 0,
		},
		{
			name:    "nil",
			caps:    nil,
			wantLen: 0,
		},
		{
			name: "single cap",
			caps: map[string]interface{}{
				"mon": "allow *",
			},
			wantLen: 2,
		},
		{
			name: "multiple caps",
			caps: map[string]interface{}{
				"mon": "allow *",
				"osd": "allow rw",
				"mds": "allow rw",
			},
			wantLen: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toCapsArray(tt.caps)

			if len(result) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(result), tt.wantLen)
			}

			// Verify that every key-value pair in the output matches the input map.
			// Elements come in [key, value, key, value, ...] order.
			for i := 0; i < len(result); i += 2 {
				key := result[i]
				val := result[i+1]
				want, ok := tt.caps[key]
				if !ok {
					t.Errorf("unexpected key %q in output", key)
					continue
				}
				if val != want.(string) {
					t.Errorf("caps[%q] = %q, want %q", key, val, want)
				}
			}
		})
	}
}

func TestClientKeyringFormat(t *testing.T) {
	entity := "client.admin"
	key := "AQABCDEFGHIJKLMNOPQRSTUVWXYZabcdef012345=="

	result := fmt.Sprintf(clientKeyringFormat, entity, key)

	if !strings.HasPrefix(result, "["+entity+"]") {
		t.Errorf("keyring does not start with [%s], got: %q", entity, result)
	}
	if !strings.Contains(result, key) {
		t.Errorf("keyring does not contain key %q", key)
	}
}
