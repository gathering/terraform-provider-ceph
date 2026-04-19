resource "ceph_auth" "myapp" {
  entity = "client.myapp"
  caps = {
    "mon" = "profile rbd"
    "osd" = "profile rbd pool=mypool"
  }
}

# The generated key and keyring are available as sensitive outputs
output "myapp_key" {
  value     = ceph_auth.myapp.key
  sensitive = true
}
