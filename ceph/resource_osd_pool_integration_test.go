//go:build integration

package ceph

import (
	"encoding/json"
	"testing"
)

func TestIntegrationOSDPool(t *testing.T) {
	config := integrationConfig(t)
	conn, err := config.GetCephConnection()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	const name = "tftest-pool"

	t.Cleanup(func() {
		cmd, _ := json.Marshal(map[string]interface{}{
			"prefix":                      "osd pool delete",
			"pool":                        name,
			"pool2":                       name,
			"yes_i_really_really_mean_it": true,
			"format":                      "json",
		})
		conn.MonCommand(cmd) //nolint:errcheck
	})

	t.Run("create", func(t *testing.T) {
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix":    "osd pool create",
			"pool":      name,
			"pool_type": "replicated",
			"format":    "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, _, err = conn.MonCommand(cmd); err != nil {
			t.Fatalf("create pool: %v", err)
		}
	})

	t.Run("read", func(t *testing.T) {
		pool, _, err := osdPoolGetAll(conn, name)
		if err != nil {
			t.Fatalf("read pool: %v", err)
		}
		if pool.Pool != name {
			t.Errorf("pool name = %q, want %q", pool.Pool, name)
		}
	})

	t.Run("set crush_rule", func(t *testing.T) {
		pool, _, err := osdPoolGetAll(conn, name)
		if err != nil {
			t.Fatalf("read pool before update: %v", err)
		}
		// Set crush_rule to its current value — verifies osdPoolSet sends a
		// valid command without requiring multiple OSDs or safety flags.
		if err := osdPoolSet(conn, name, "crush_rule", pool.CrushRule); err != nil {
			t.Fatalf("set crush_rule: %v", err)
		}
		updated, _, err := osdPoolGetAll(conn, name)
		if err != nil {
			t.Fatalf("read pool after update: %v", err)
		}
		if updated.CrushRule != pool.CrushRule {
			t.Errorf("crush_rule = %q, want %q", updated.CrushRule, pool.CrushRule)
		}
	})

	t.Run("application enable and get", func(t *testing.T) {
		if err := osdPoolApplicationEnable(conn, name, "rbd"); err != nil {
			t.Fatalf("enable application: %v", err)
		}
		apps, err := osdPoolApplicationGet(conn, name)
		if err != nil {
			t.Fatalf("get applications: %v", err)
		}
		found := false
		for _, a := range apps {
			if a == "rbd" {
				found = true
			}
		}
		if !found {
			t.Errorf("application rbd not found in %v", apps)
		}
	})

	t.Run("application disable", func(t *testing.T) {
		if err := osdPoolApplicationDisable(conn, name, "rbd"); err != nil {
			t.Fatalf("disable application: %v", err)
		}
		apps, err := osdPoolApplicationGet(conn, name)
		if err != nil {
			t.Fatalf("get applications: %v", err)
		}
		for _, a := range apps {
			if a == "rbd" {
				t.Errorf("application rbd still present after disable")
			}
		}
	})
}
