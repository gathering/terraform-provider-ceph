//go:build integration

package ceph

import (
	"encoding/json"
	"testing"
)

func TestIntegrationFS(t *testing.T) {
	config := integrationConfig(t)
	conn, err := config.GetCephConnection()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	const (
		fsName    = "tftest-fs"
		metaPool  = "tftest-fs-meta"
		dataPool  = "tftest-fs-data"
		extraPool = "tftest-fs-data2"
	)

	createPool := func(t *testing.T, name string) {
		t.Helper()
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix":    "osd pool create",
			"pool":      name,
			"pool_type": "replicated",
			"format":    "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, _, err := conn.MonCommand(cmd); err != nil {
			t.Fatalf("create pool %q: %v", name, err)
		}
	}

	deletePool := func(name string) {
		cmd, _ := json.Marshal(map[string]interface{}{
			"prefix":                      "osd pool delete",
			"pool":                        name,
			"pool2":                       name,
			"yes_i_really_really_mean_it": true,
			"format":                      "json",
		})
		conn.MonCommand(cmd) //nolint:errcheck
	}

	t.Cleanup(func() {
		// Fail and remove the filesystem before deleting pools.
		failCmd, _ := json.Marshal(map[string]interface{}{
			"prefix":  "fs fail",
			"fs_name": fsName,
			"format":  "json",
		})
		conn.MonCommand(failCmd) //nolint:errcheck

		rmCmd, _ := json.Marshal(map[string]interface{}{
			"prefix":               "fs rm",
			"fs_name":              fsName,
			"yes_i_really_mean_it": true,
			"format":               "json",
		})
		conn.MonCommand(rmCmd) //nolint:errcheck

		deletePool(metaPool)
		deletePool(dataPool)
		deletePool(extraPool)
	})

	createPool(t, metaPool)
	createPool(t, dataPool)
	createPool(t, extraPool)

	t.Run("create", func(t *testing.T) {
		cmd, err := json.Marshal(map[string]interface{}{
			"prefix":   "fs new",
			"fs_name":  fsName,
			"metadata": metaPool,
			"data":     dataPool,
			"format":   "json",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, _, err := conn.MonCommand(cmd); err != nil {
			t.Fatalf("create fs: %v", err)
		}
	})

	t.Run("read", func(t *testing.T) {
		fs, err := fsGet(conn, fsName)
		if err != nil {
			t.Fatalf("fsGet: %v", err)
		}
		if fs == nil {
			t.Fatalf("filesystem %q not found", fsName)
		}
		if fs.Name != fsName {
			t.Errorf("name = %q, want %q", fs.Name, fsName)
		}
		if fs.MetadataPool != metaPool {
			t.Errorf("metadata_pool = %q, want %q", fs.MetadataPool, metaPool)
		}
		found := false
		for _, p := range fs.DataPools {
			if p == dataPool {
				found = true
			}
		}
		if !found {
			t.Errorf("data pool %q not found in %v", dataPool, fs.DataPools)
		}
	})

	t.Run("add data pool", func(t *testing.T) {
		if err := fsAddDataPool(conn, fsName, extraPool); err != nil {
			t.Fatalf("add data pool: %v", err)
		}
		fs, err := fsGet(conn, fsName)
		if err != nil {
			t.Fatalf("fsGet: %v", err)
		}
		found := false
		for _, p := range fs.DataPools {
			if p == extraPool {
				found = true
			}
		}
		if !found {
			t.Errorf("extra pool %q not found in %v", extraPool, fs.DataPools)
		}
	})

	t.Run("remove data pool", func(t *testing.T) {
		if err := fsRemoveDataPool(conn, fsName, extraPool); err != nil {
			t.Fatalf("remove data pool: %v", err)
		}
		fs, err := fsGet(conn, fsName)
		if err != nil {
			t.Fatalf("fsGet: %v", err)
		}
		for _, p := range fs.DataPools {
			if p == extraPool {
				t.Errorf("extra pool %q still present after removal", extraPool)
			}
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		fs, err := fsGet(conn, "does-not-exist")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fs != nil {
			t.Errorf("expected nil for missing fs, got %+v", fs)
		}
	})
}
