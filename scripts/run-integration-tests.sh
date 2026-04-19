#!/bin/bash
set -euo pipefail

CEPH_DIR=$(mktemp -d)

cleanup() {
    pkill ceph || true
    rm -rf "$CEPH_DIR"
}
trap cleanup EXIT

# Minimal featureset: mon + osd + mgr is sufficient for all provider operations.
# MDS is not required since we only exercise the Ceph API (pool/auth/fs management),
# not actual filesystem mounts.
export CEPH_FEATURESET="mon osd mgr selftest"
/build/scripts/micro-osd.sh "$CEPH_DIR"

export CEPH_CONF="$CEPH_DIR/ceph.conf"

echo "==> Running integration tests"
go test -v -count=1 -tags integration ./ceph/

echo "==> Running acceptance tests"
TF_ACC=1 go test -v -count=1 -tags acceptance ./ceph/
