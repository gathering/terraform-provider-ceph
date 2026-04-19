//go:build integration

package ceph

import (
	"os"
	"testing"
)

func integrationConfig(t *testing.T) *Config {
	t.Helper()
	confPath := os.Getenv("CEPH_CONF")
	if confPath == "" {
		confPath = "/etc/ceph/ceph.conf"
	}
	return &Config{
		ConfigPath: confPath,
		Entity:     "client.admin",
		Cluster:    "ceph",
	}
}
