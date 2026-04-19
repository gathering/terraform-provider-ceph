# Connect using a config file on the local filesystem
provider "ceph" {
  config_path = "/etc/ceph/ceph.conf"
  entity      = "client.admin"
}

# Connect by supplying the key directly (e.g. from a secret store)
provider "ceph" {
  mon_host = "192.168.1.10:6789"
  entity   = "client.admin"
  key      = var.ceph_admin_key
}
