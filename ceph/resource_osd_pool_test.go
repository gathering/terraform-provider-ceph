package ceph

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// mockMonCommander satisfies the monCommander interface and records the last
// command received so tests can assert on what was sent.
type mockMonCommander struct {
	response []byte
	status   string
	err      error
	lastCmd  map[string]interface{}
}

func (m *mockMonCommander) MonCommand(cmd []byte) ([]byte, string, error) {
	_ = json.Unmarshal(cmd, &m.lastCmd)
	return m.response, m.status, m.err
}

// --- osdPoolSet ---

func TestOsdPoolSet(t *testing.T) {
	tests := []struct {
		name     string
		pool     string
		variable string
		value    string
		mockErr  error
		wantErr  bool
	}{
		{
			name:     "success",
			pool:     "testpool",
			variable: "size",
			value:    "3",
		},
		{
			name:     "propagates MonCommand error",
			pool:     "testpool",
			variable: "size",
			value:    "3",
			mockErr:  errors.New("EPERM: not authorized"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockMonCommander{response: []byte("{}"), err: tt.mockErr}

			err := osdPoolSet(mock, tt.pool, tt.variable, tt.value)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Verify the JSON command that was sent.
			if mock.lastCmd["prefix"] != "osd pool set" {
				t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "osd pool set")
			}
			if mock.lastCmd["pool"] != tt.pool {
				t.Errorf("pool = %v, want %q", mock.lastCmd["pool"], tt.pool)
			}
			if mock.lastCmd["var"] != tt.variable {
				t.Errorf("var = %v, want %q", mock.lastCmd["var"], tt.variable)
			}
			if mock.lastCmd["val"] != tt.value {
				t.Errorf("val = %v, want %q", mock.lastCmd["val"], tt.value)
			}
		})
	}
}

// --- osdPoolGetAll ---

// --- osdPoolApplicationEnable / Disable ---

func TestOsdPoolApplicationEnable(t *testing.T) {
	mock := &mockMonCommander{response: []byte("{}")}

	if err := osdPoolApplicationEnable(mock, "testpool", "rbd"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastCmd["prefix"] != "osd pool application enable" {
		t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "osd pool application enable")
	}
	if mock.lastCmd["pool"] != "testpool" {
		t.Errorf("pool = %v, want %q", mock.lastCmd["pool"], "testpool")
	}
	if mock.lastCmd["app"] != "rbd" {
		t.Errorf("app = %v, want %q", mock.lastCmd["app"], "rbd")
	}
}

func TestOsdPoolApplicationDisable(t *testing.T) {
	mock := &mockMonCommander{response: []byte("{}")}

	if err := osdPoolApplicationDisable(mock, "testpool", "rbd"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastCmd["prefix"] != "osd pool application disable" {
		t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "osd pool application disable")
	}
	// Ceph requires this flag to disable an application.
	if mock.lastCmd["yes_i_really_mean_it"] != true {
		t.Errorf("yes_i_really_mean_it = %v, want true", mock.lastCmd["yes_i_really_mean_it"])
	}
}

// --- osdPoolApplicationGet ---

func TestOsdPoolApplicationGet(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantApps []string
	}{
		{
			name:     "no applications",
			response: `{}`,
			wantApps: []string{},
		},
		{
			name:     "single application",
			response: `{"rbd": {}}`,
			wantApps: []string{"rbd"},
		},
		{
			name:     "multiple applications",
			response: `{"rbd": {}, "cephfs": {}}`,
			wantApps: []string{"cephfs", "rbd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockMonCommander{response: []byte(tt.response)}

			apps, err := osdPoolApplicationGet(mock, "testpool")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(apps) != len(tt.wantApps) {
				t.Fatalf("len(apps) = %d, want %d (got %v)", len(apps), len(tt.wantApps), apps)
			}
			// Use a set comparison since map iteration order is non-deterministic.
			got := make(map[string]bool, len(apps))
			for _, a := range apps {
				got[a] = true
			}
			for _, want := range tt.wantApps {
				if !got[want] {
					t.Errorf("missing application %q in result %v", want, apps)
				}
			}
		})
	}
}

func TestOsdPoolGetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		poolJSON := `{
			"pool": "testpool",
			"pool_id": 3,
			"size": 3,
			"min_size": 2,
			"pg_num": 32,
			"crush_rule": "replicated_rule"
		}`
		mock := &mockMonCommander{response: []byte(poolJSON)}

		pool, _, err := osdPoolGetAll(mock, "testpool")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pool.Pool != "testpool" {
			t.Errorf("Pool = %q, want %q", pool.Pool, "testpool")
		}
		if pool.Size != 3 {
			t.Errorf("Size = %d, want 3", pool.Size)
		}
		if pool.MinSize != 2 {
			t.Errorf("MinSize = %d, want 2", pool.MinSize)
		}
		if pool.PgNum != 32 {
			t.Errorf("PgNum = %d, want 32", pool.PgNum)
		}
		if pool.CrushRule != "replicated_rule" {
			t.Errorf("CrushRule = %q, want %q", pool.CrushRule, "replicated_rule")
		}

		// Verify the command sent to Ceph.
		if mock.lastCmd["prefix"] != "osd pool get" {
			t.Errorf("prefix = %v, want %q", mock.lastCmd["prefix"], "osd pool get")
		}
		if mock.lastCmd["pool"] != "testpool" {
			t.Errorf("pool = %v, want %q", mock.lastCmd["pool"], "testpool")
		}
		if mock.lastCmd["var"] != "all" {
			t.Errorf("var = %v, want %q", mock.lastCmd["var"], "all")
		}
	})

	t.Run("pool not found returns status and error", func(t *testing.T) {
		mock := &mockMonCommander{
			response: []byte(""),
			status:   "Error ENOENT: unrecognized pool 'missing'",
			err:      errors.New("exit status 2"),
		}

		_, status, err := osdPoolGetAll(mock, "missing")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(status, "ENOENT") {
			t.Errorf("status = %q, want it to contain ENOENT", status)
		}
	})

	t.Run("invalid JSON response returns error", func(t *testing.T) {
		mock := &mockMonCommander{response: []byte("not json")}

		_, _, err := osdPoolGetAll(mock, "testpool")
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}
