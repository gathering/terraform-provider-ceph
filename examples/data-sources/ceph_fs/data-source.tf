data "ceph_fs" "cephfs" {
  name = "cephfs"
}

output "cephfs_metadata_pool" {
  value = data.ceph_fs.cephfs.metadata_pool
}
