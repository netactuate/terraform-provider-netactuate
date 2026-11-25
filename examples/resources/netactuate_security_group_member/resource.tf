resource "netactuate_security_group_member" "example" {
  security_group_id = 12345
  vpc_id = 23456
  mbpkgid = 34567
}
