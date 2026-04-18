resource "ceph_wait_online" "wait" {
  cluster_name = "my-cluster"

  timeouts {
    create = "30m"
  }
}
