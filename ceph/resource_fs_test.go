package ceph

import (
	"errors"
	"testing"
)

// --- fsGet ---

func TestFsGet(t *testing.T) {
	fsList := `[
		{"name":"cephfs","metadata_pool":"cephfs_metadata","data_pools":["cephfs_data"]},
		{"name":"other","metadata_pool":"other_metadata","data_pools":["other_data"]}
	]`

	t.Run("found", func(t *testing.T) {
		mock := &mockMonCommander{response: []byte(fsList)}

		fs, err := fsGet(mock, "cephfs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fs == nil {
			t.Fatal("expected entry, got nil")
		}
		if fs.Name != "cephfs" {
			t.Errorf("Name = %q, want %q", fs.Name, "cephfs")
		}
		if fs.MetadataPool != "cephfs_metadata" {
			t.Errorf("MetadataPool = %q, want %q", fs.MetadataPool, "cephfs_metadata")
		}
		if len(fs.DataPools) != 1 || fs.DataPools[0] != "cephfs_data" {
			t.Errorf("DataPoolList = %v, want [cephfs_data]", fs.DataPools)
		}
		if mock.lastCmd["prefix"] != "fs ls" {
			t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "fs ls")
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		mock := &mockMonCommander{response: []byte(fsList)}

		fs, err := fsGet(mock, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fs != nil {
			t.Errorf("expected nil, got %+v", fs)
		}
	})

	t.Run("MonCommand error propagated", func(t *testing.T) {
		mock := &mockMonCommander{err: errors.New("connection refused")}

		_, err := fsGet(mock, "cephfs")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		mock := &mockMonCommander{response: []byte("not json")}

		_, err := fsGet(mock, "cephfs")
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}

// --- fsAddDataPool ---

func TestFsAddDataPool(t *testing.T) {
	mock := &mockMonCommander{response: []byte("{}")}

	if err := fsAddDataPool(mock, "cephfs", "extra_data"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastCmd["prefix"] != "fs add_data_pool" {
		t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "fs add_data_pool")
	}
	if mock.lastCmd["fs_name"] != "cephfs" {
		t.Errorf("fs_name = %v, want %q", mock.lastCmd["fs_name"], "cephfs")
	}
	if mock.lastCmd["pool"] != "extra_data" {
		t.Errorf("pool = %v, want %q", mock.lastCmd["pool"], "extra_data")
	}
}

// --- fsRemoveDataPool ---

func TestFsRemoveDataPool(t *testing.T) {
	mock := &mockMonCommander{response: []byte("{}")}

	if err := fsRemoveDataPool(mock, "cephfs", "extra_data"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastCmd["prefix"] != "fs rm_data_pool" {
		t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "fs rm_data_pool")
	}
	if mock.lastCmd["fs_name"] != "cephfs" {
		t.Errorf("fs_name = %v, want %q", mock.lastCmd["fs_name"], "cephfs")
	}
	if mock.lastCmd["pool"] != "extra_data" {
		t.Errorf("pool = %v, want %q", mock.lastCmd["pool"], "extra_data")
	}
}
