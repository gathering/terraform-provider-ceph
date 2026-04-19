# Terraform Ceph Provider

A Terraform provider for managing Ceph resources. Originally created at CERN, forked and maintained by The Gathering. Contributions are welcome.

## Resources and Data Sources

| Type | Name | Description |
|------|------|-------------|
| Resource | `ceph_auth` | Manage cephx client entities and their capabilities |
| Resource | `ceph_fs` | Manage CephFS filesystems |
| Resource | `ceph_osd_pool` | Manage OSD pools (replicated) |
| Resource | `ceph_wait_online` | Block until the Ceph cluster is reachable |
| Data Source | `ceph_auth` | Read key and capabilities for an existing cephx entity |
| Data Source | `ceph_fs` | Read configuration of an existing CephFS filesystem |
| Data Source | `ceph_osd_pool` | Read configuration of an existing OSD pool |

See the [`docs/`](docs/) directory for full attribute documentation.

## Requirements

- Terraform >= 1.0 or OpenTofu >= 1.0
- Go >= 1.25 (to build the provider)
- `librados-dev` and `librbd-dev` (required at build time; not needed with Docker)
- `librados` and `librbd` (required at runtime on the machine running Terraform)

## Usage

```hcl
terraform {
  required_providers {
    ceph = {
      source = "gathering/ceph"
    }
  }
}

# Authenticate via a local config file
provider "ceph" {
  config_path = "/etc/ceph/ceph.conf"
  entity      = "client.admin"
}

# Or supply credentials directly
provider "ceph" {
  mon_host = "192.168.1.10:6789"
  entity   = "client.admin"
  key      = var.ceph_admin_key
}
```

## Building

With Docker (no local Ceph libraries needed):

```sh
docker build --output=. --target=binary .
```

Locally (requires `librados-dev`):

```sh
make build
```

## Testing

Docker images only need to be rebuilt when `go.mod`/`go.sum`, `scripts/`, or system packages change — not on source edits.

### Unit tests

```sh
make docker-image                         # Build the image
make docker-test                          # run in Docker (recommended)
make docker-test TEST_ARGS="-run TestFoo" # run a specific test
make test                                 # run locally (requires librados-dev)
```

### Integration and acceptance tests

Integration tests exercise the Ceph API directly against a real single-node cluster (micro-osd). Acceptance tests run the full Terraform lifecycle — plan, apply, update, import, destroy.

```sh
make docker-integration-image   # build image once (includes Ceph daemons)
make docker-integration-test    # run all integration and acceptance tests
```

To run acceptance tests against an existing cluster:

```sh
CEPH_CONF=/etc/ceph/ceph.conf TF_ACC=1 go test -v -count=1 -tags acceptance ./ceph/
```

## Regenerating Documentation

```sh
make docker-generate      # with Docker
go generate ./...         # locally (requires librados-dev)
```

## Notes

- **Pool deletion** requires `mon_allow_pool_delete = true` in the Ceph configuration.
- **Filesystem deletion** marks the filesystem as failed before removing it. Active MDS daemons should be stopped beforehand.
- The `key` and `keyring` attributes are marked sensitive and will not appear in plan output.
