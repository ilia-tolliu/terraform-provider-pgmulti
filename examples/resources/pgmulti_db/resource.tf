# Copyright (c) HashiCorp, Inc.

resource "pgmulti_db" "example" {
  hostname        = "localhost"
  port            = 5432
  master_username = "root"
  master_password = "12345"
  db_name         = "example_db"
}
