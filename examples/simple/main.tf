terraform {
  required_version = ">= 1.8.0"
  required_providers {
    ceph = {
      source  = "gathering/ceph"
      version = "~> 0.1.0"
    }
  }
}

provider "ceph" {
  entity = "client.admin"
}

resource "ceph_wait_online" "wait" {
  cluster_name = "my-super-cluster"
}

resource "ceph_osd_pool" "mypool" {
  name        = "mypool"
  pg_num      = 64
  size        = 3
  application = ["rbd"]

  depends_on = [ceph_wait_online.wait]
}

resource "ceph_osd_pool" "cephfs_metadata" {
  name        = "cephfs_metadata"
  size        = 3
  application = ["cephfs"]

  depends_on = [ceph_wait_online.wait]
}

resource "ceph_osd_pool" "cephfs_data" {
  name        = "cephfs_data"
  size        = 3
  application = ["cephfs"]

  depends_on = [ceph_wait_online.wait]
}

resource "ceph_fs" "cephfs" {
  name          = "cephfs"
  metadata_pool = ceph_osd_pool.cephfs_metadata.name
  data_pools    = [ceph_osd_pool.cephfs_data.name]
}

resource "ceph_auth" "test" {
  entity = "client.test"
  caps = {
    "mon" = "profile rbd"
    "osd" = "profile rbd pool=mypool"
  }

  depends_on = [ceph_osd_pool.mypool]
}

data "ceph_auth" "test_data" {
  entity = "client.test"

  depends_on = [ceph_auth.test]
}
