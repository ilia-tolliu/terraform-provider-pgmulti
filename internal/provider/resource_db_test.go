// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/jackc/pgx/v5"
)

const (
	config = `
resource "docker_image" "postgres" {
  name = "postgres"
  keep_locally = true
}

resource "docker_container" "postgres" {
  name  = "tf_postgres"
  image = docker_image.postgres.image_id
  env = [
    "POSTGRES_USER=root",
    "POSTGRES_PASSWORD=12345"
   ]
  ports {
    internal = 5432
  }
  healthcheck {
  	test = [
      "CMD-SHELL",
      "sh -c 'pg_isready'"
    ]
    interval = "1s"
    timeout = "5s"
    retries = "5"
  }
  wait = true
}
`
)

func TestAccPgmultiDb(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"pgmulti": providerserver.NewProtocol6WithError(NewPgmulti()),
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"docker": {
				Source: "kreuzwerker/docker",
			},
		},
		PreCheck: func() {

		},
		Steps: []resource.TestStep{
			{
				Config: config + `
resource "pgmulti_db" "test_db" {
  hostname        = "localhost"
  port            = docker_container.postgres.ports[0].external
  master_username = "root"
  master_password = "12345"
  db_name         = "test_db"
}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					checkDbResourceExists{},
					checkDbExistence{
						shouldExist: true,
					},
					checkDbInteraction{},
				},
			},
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					checkDbExistence{
						shouldExist: false,
					},
				},
			},
		},
	})
}

var dbAttrs dbAttributes = dbAttributes{}
var _ statecheck.StateCheck = checkDbResourceExists{}
var _ statecheck.StateCheck = checkDbExistence{}
var _ statecheck.StateCheck = checkDbInteraction{}

type dbAttributes struct {
	hostname       string
	port           string
	masterUsername string
	masterPassword string
	dbName         string
	dbUsername     string
	dbPassword     string
}

type checkDbResourceExists struct{}

func (s checkDbResourceExists) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	var resource *tfjson.StateResource

	if req.State == nil {
		resp.Error = fmt.Errorf("state is nil")
	}

	if req.State.Values == nil {
		resp.Error = fmt.Errorf("state does not contain any state values")
	}

	if req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("state does not contain a root module")
	}

	for _, r := range req.State.Values.RootModule.Resources {
		if r.Name == "test_db" {
			resource = r

			break
		}
	}

	if resource == nil {
		resp.Error = fmt.Errorf("Resource not found in state")

		return
	}

	dbAttrs.hostname, _ = resource.AttributeValues["hostname"].(string)
	port, _ := resource.AttributeValues["port"].(json.Number)
	dbAttrs.port = port.String()
	dbAttrs.masterUsername, _ = resource.AttributeValues["master_username"].(string)
	dbAttrs.masterPassword, _ = resource.AttributeValues["master_password"].(string)
	dbAttrs.dbName, _ = resource.AttributeValues["db_name"].(string)
	dbAttrs.dbUsername, _ = resource.AttributeValues["db_username"].(string)
	dbAttrs.dbPassword, _ = resource.AttributeValues["db_password"].(string)

}

type checkDbExistence struct {
	shouldExist bool
}

func (s checkDbExistence) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbAttrs.masterUsername,
		dbAttrs.masterPassword,
		dbAttrs.hostname,
		dbAttrs.port,
		"postgres",
	)

	conn, err := connectDb(ctx, connStr)
	if err != nil {
		resp.Error = fmt.Errorf("Failed to connect to Db: %s", err)

		return
	}

	oid, err := getDbOid(ctx, conn, dbAttrs.dbName)
	if !s.shouldExist && errors.Is(err, pgx.ErrNoRows) {
		return
	} else if s.shouldExist && errors.Is(err, pgx.ErrNoRows) {
		resp.Error = fmt.Errorf("Database %s doesn't exist when it should", dbAttrs.dbName)

		return
	} else if err != nil {
		resp.Error = fmt.Errorf("Failed to check if DB exists: %s", err)

		return
	}

	if !s.shouldExist && oid > 0 {
		resp.Error = fmt.Errorf("Database %s exist when it shouldn't", dbAttrs.dbName)

		return
	}
}

type checkDbInteraction struct{}

func (s checkDbInteraction) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbAttrs.dbUsername,
		dbAttrs.dbPassword,
		dbAttrs.hostname,
		dbAttrs.port,
		dbAttrs.dbName,
	)

	conn, err := connectDb(ctx, connStr)
	if err != nil {
		resp.Error = fmt.Errorf("Failed to connect to Db: %s", err)

		return
	}

	err = createTable(ctx, conn, "test_table")
	if err != nil {
		resp.Error = fmt.Errorf("Failed to create table: %s", err)

		return
	}
}
