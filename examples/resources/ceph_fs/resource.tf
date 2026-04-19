resource "ceph_osd_pool" "cephfs_metadata" {
  name        = "cephfs_metadata"
  size        = 3
  application = ["cephfs"]
}

resource "ceph_osd_pool" "cephfs_data" {
  name        = "cephfs_data"
  size        = 3
  application = ["cephfs"]
}

resource "ceph_fs" "cephfs" {
  name          = "cephfs"
  metadata_pool = ceph_osd_pool.cephfs_metadata.name
  data_pools    = [ceph_osd_pool.cephfs_data.name]
}

# Client with read-write access to the filesystem
resource "ceph_auth" "cephfs_client" {
  entity = "client.cephfs"
  caps = {
    "mon" = "allow r"
    "mds" = "allow rw"
    "osd" = "allow rw tag cephfs data=${ceph_fs.cephfs.name}"
  }
}

output "cephfs_client_keyring" {
  value     = ceph_auth.cephfs_client.keyring
  sensitive = true
}
