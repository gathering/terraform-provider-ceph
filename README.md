# Terraform Ceph Provider

This Terraform provider is used to manage Ceph. It was created at CERN and forked by The Gathering and is currently used to configure Ceph. Contributions are welcome.

## Resources

| Resource | Description |
|---|---|
| `ceph_auth` | Create and manage cephx client entities and their capabilities |
| `ceph_fs` | Create and manage CephFS filesystems |
| `ceph_osd_pool` | Create and manage OSD pools (replicated) |
| `ceph_wait_online` | Block until the Ceph cluster is reachable — useful during bootstrap |

## Data Sources

| Data Source | Description |
|---|---|
| `ceph_auth` | Read key and capabilities for an existing cephx entity |
| `ceph_fs` | Read configuration of an existing CephFS filesystem |
| `ceph_osd_pool` | Read configuration of an existing OSD pool |

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= v1.8
- [Go](https://golang.org/doc/install) >= 1.25 (to build the provider)
- `librados-dev` and `librbd-dev` (Ceph C libraries, required at build time)
- [golangci-lint](https://golangci-lint.run/welcome/install/) (optional, for local linting — `brew install golangci-lint` on macOS)

## Usage

```hcl
terraform {
  required_providers {
    ceph = {
      source  = "gathering/ceph"
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

See the [`docs/`](docs/) directory for full provider, resource, and data source documentation.

## Building

The provider requires the Ceph `librados` C library at compile time. The easiest way to build without installing Ceph locally is with Docker:

```sh
# Build the provider binary inside Docker
docker build --target=builder -t terraform-provider-ceph-build .

# Extract the binary to the current directory
docker run --rm -v "$(pwd):/output" terraform-provider-ceph-build \
  cp /build/terraform-provider-ceph /output/
```

Or, using BuildKit's `--output`:
```sh
docker build --output=. --target=binary .
```

To build directly on a system with `librados-dev` installed:
```sh
make build
```

## Testing

### Unit tests

Unit tests use mocks and do not require a live Ceph cluster.

**With Docker** (recommended — no local Ceph libraries needed):
```sh
make docker-test           # run tests
make docker-test-verbose   # run tests with verbose output
```

**Locally** (requires `librados-dev`):
```sh
make test           # run tests
make test-verbose   # run tests with verbose output
```

To pass extra flags to `go test` (e.g. `-run TestOsdPool`):
```sh
make docker-test TEST_ARGS="-run TestOsdPool"
```

### Integration tests

Integration tests run against a real single-node Ceph cluster bootstrapped with [micro-osd](https://github.com/ceph/go-ceph/blob/master/testing/containers/micro-osd.sh) inside Docker. They exercise the Ceph API layer directly (no Terraform state). No local Ceph installation is required.

```sh
make docker-integration-image   # build the image (includes full Ceph daemons)
make docker-integration-test    # run integration and acceptance tests
```

The image is separate from the standard build image and only needs to be rebuilt when the Dockerfile or Ceph version changes.

### Acceptance tests

Acceptance tests (`//go:build acceptance`) run the full Terraform provider lifecycle — plan, apply, update, import, and destroy — against a real Ceph cluster. They are run automatically as part of `docker-integration-test`.

To run acceptance tests manually against an existing cluster:

```sh
CEPH_CONF=/etc/ceph/ceph.conf TF_ACC=1 go test -v -count=1 -tags acceptance ./ceph/
```

The tests create and clean up all Ceph resources (pools, auth entities, filesystems) automatically.

## Regenerating Documentation

Docs are generated from schema descriptions and example files using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs):

```sh
make docker-generate
```

Or locally (requires `librados-dev`):
```sh
go generate ./...
```

## Notes

- **Pool deletion** (`ceph_osd_pool`) requires `mon_allow_pool_delete = true` in the Ceph configuration.
- **Filesystem deletion** (`ceph_fs`) marks the filesystem as failed before removing it. Active MDS daemons should be stopped beforehand.
- The `key` and `keyring` provider attributes are marked sensitive and will not appear in plan output.
