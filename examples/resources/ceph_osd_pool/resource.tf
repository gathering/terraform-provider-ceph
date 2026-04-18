# RBD pool — application enable + rbd pool init are handled automatically
resource "ceph_osd_pool" "kubernetes" {
  name        = "kubernetes"
  pg_num      = 64
  size        = 3
  application = ["rbd"]
}

# Pool with no application tag
resource "ceph_osd_pool" "scratch" {
  name = "scratch"
}
