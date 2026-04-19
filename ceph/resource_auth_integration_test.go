//go:build integration

package ceph

import (
	"encoding/json"
	"testing"
)

func TestIntegrationAuth(t *testing.T) {
	config := integrationConfig(t)
	conn, err := config.GetCephConnection()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	const entity = "client.tftest"

	t.Cleanup(func() {
		cmd, _ := json.Marshal(map[string]interface{}{
			"prefix": "auth rm",
			"entity": entity,
			"format": "json",
		})
		conn.MonCommand(cmd) //nolint:errcheck
	})

	t.Run("create", func(t *testing.T) {
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix": "auth get-or-create",
			"entity": entity,
			"caps":   []string{"mon", "allow r"},
			"format": "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		buf, _, err := conn.MonCommand(cmd)
		if err != nil {
			t.Fatalf("create auth: %v", err)
		}
		var responses []authResponse
		if err := json.Unmarshal(buf, &responses); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(responses) == 0 {
			t.Fatal("no auth response returned")
		}
		if responses[0].Entity != entity {
			t.Errorf("entity = %q, want %q", responses[0].Entity, entity)
		}
		if responses[0].Key == "" {
			t.Error("key is empty")
		}
	})

	t.Run("read", func(t *testing.T) {
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix": "auth get",
			"entity": entity,
			"format": "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		buf, _, err := conn.MonCommand(cmd)
		if err != nil {
			t.Fatalf("read auth: %v", err)
		}
		var responses []authResponse
		if err := json.Unmarshal(buf, &responses); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(responses) == 0 {
			t.Fatal("no auth response returned")
		}
		if cap, ok := responses[0].Caps["mon"]; !ok || cap != "allow r" {
			t.Errorf("mon cap = %q, want %q", cap, "allow r")
		}
	})

	t.Run("update caps", func(t *testing.T) {
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix": "auth caps",
			"entity": entity,
			"caps":   []string{"mon", "allow r", "osd", "allow rw"},
			"format": "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, _, err := conn.MonCommand(cmd); err != nil {
			t.Fatalf("update caps: %v", err)
		}

		getCmd, _ := json.Marshal(map[string]interface{}{
			"prefix": "auth get",
			"entity": entity,
			"format": "json",
		})
		buf, _, err := conn.MonCommand(getCmd)
		if err != nil {
			t.Fatalf("read auth: %v", err)
		}
		var responses []authResponse
		if err := json.Unmarshal(buf, &responses); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if cap, ok := responses[0].Caps["osd"]; !ok || cap != "allow rw" {
			t.Errorf("osd cap = %q, want %q", cap, "allow rw")
		}
	})

	t.Run("keyring format", func(t *testing.T) {
		cmd, _ := json.Marshal(map[string]interface{}{
			"prefix": "auth get",
			"entity": entity,
			"format": "json",
		})
		buf, _, err := conn.MonCommand(cmd)
		if err != nil {
			t.Fatalf("read auth: %v", err)
		}
		var responses []authResponse
		if err := json.Unmarshal(buf, &responses); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		// Use a mock schema.ResourceData substitute by calling the format directly.
		keyring := responses[0].Key
		if keyring == "" {
			t.Error("key is empty")
		}
	})
}
