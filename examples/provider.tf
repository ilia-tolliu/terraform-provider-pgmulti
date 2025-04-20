# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    pgmulti = {
      source  = "ilia-tolliu/pgmulti"
      version = "1.0.0"
    }
  }
}

provider "pgmulti" {
  # Nothing to configure
}

# Create a database on a running PostgreSQL instance
resource "pgmulti_db" "example" {
  hostname        = "localhost"
  port            = 5432
  master_username = "root"
  master_password = "12345"
  db_name         = "example_db"
}
