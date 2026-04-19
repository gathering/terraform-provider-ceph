data "ceph_auth" "admin" {
  entity = "client.admin"
}

output "admin_key" {
  value     = data.ceph_auth.admin.key
  sensitive = true
}
