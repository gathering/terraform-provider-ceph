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
