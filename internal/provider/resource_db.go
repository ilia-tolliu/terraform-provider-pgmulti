// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ilia-tolliu/terraform-provider-pgmulti/internal/gen"
	"github.com/jackc/pgx/v5"
)

type ResourceDb struct{}

type ResourceDbModel struct {
	Hostname       types.String `tfsdk:"hostname"`
	Port           types.Int32  `tfsdk:"port"`
	MasterUsername types.String `tfsdk:"master_username"`
	MasterPassword types.String `tfsdk:"master_password"`
	DbName         types.String `tfsdk:"db_name"`
	DbUsername     types.String `tfsdk:"db_username"`
	DbPassword     types.String `tfsdk:"db_password"`
	Id             types.Int32  `tfsdk:"id"`
}

func NewResourceDb() resource.Resource {
	return &ResourceDb{}
}

func (r *ResourceDb) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db"
}

func (r *ResourceDb) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Database resource, one of possibly many on a PostgreSQL server instance.",

		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Description: "PostgreSQL server instance hostname.",
				Required:    true,
			},
			"port": schema.Int32Attribute{
				Description: "PostgreSQL server instance port.",
				Required:    true,
			},
			"master_username": schema.StringAttribute{
				Description: "PostgreSQL server instance master username.",
				Required:    true,
			},
			"master_password": schema.StringAttribute{
				Description: "PostgreSQL server instance master password.",
				Required:    true,
			},
			"db_name": schema.StringAttribute{
				Description: "Database name to create.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"db_username": schema.StringAttribute{
				Description: "Generated database owner username.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"db_password": schema.StringAttribute{
				Description: "Generated database owner password.",
				Required:    false,
				Optional:    false,
				Computed:    true,
				Sensitive:   true,
			},
			"id": schema.Int32Attribute{
				Description: "Generated database OID in pg_catalog.pg_database.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
		},
	}
}

func (r *ResourceDb) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceDbModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dbUrl := connStr(ctx, plan)
	conn, err := connectDb(ctx, dbUrl)
	if err != nil {
		resp.Diagnostics.AddError("Db Create Error", fmt.Sprintf("failed to connect to PostgreSQL server at %s: %s", dbUrl, err))
		return
	}
	defer conn.Close(ctx)

	username, password, err := createUser(ctx, conn)
	if err != nil {
		resp.Diagnostics.AddError("Db Create Error", fmt.Sprintf("failed to create user: %s", err))
		return
	}
	plan.DbUsername = types.StringValue(username)
	plan.DbPassword = types.StringValue(password)

	err = createDb(ctx, conn, plan.DbName.ValueString(), plan.DbUsername.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Db Create Error", fmt.Sprintf("failed to create database: %s", err))
		return
	}

	dbOid, err := getDbOid(ctx, conn, plan.DbName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Db Create Error", fmt.Sprintf("failed to get created database: %s", err))
		return
	}
	plan.Id = types.Int32Value(dbOid)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceDb) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceDbModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dbUrl := connStr(ctx, data)
	conn, err := connectDb(ctx, dbUrl)
	if err != nil {
		resp.Diagnostics.AddError("Db Read Error", fmt.Sprintf("failed to connect to RDS instance: %s", err))
		return
	}
	defer conn.Close(ctx)

	dbOid, err := getDbOid(ctx, conn, data.DbName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Db Read Error", fmt.Sprintf("failed to get created database: %s", err))
		return
	}
	data.Id = types.Int32Value(dbOid)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDb) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ResourceDb) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceDbModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dbUrl := connStr(ctx, data)
	conn, err := connectDb(ctx, dbUrl)
	if err != nil {
		resp.Diagnostics.AddError("Db Delete Error", fmt.Sprintf("failed to connect to RDS instance: %s", err))
		return
	}
	defer conn.Close(ctx)

	err = dropDb(ctx, conn, data.DbName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Db Delete Error", fmt.Sprintf("failed to drop database: %s", err))
		return
	}
}

func connStr(ctx context.Context, data ResourceDbModel) string {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres",
		data.MasterUsername.ValueString(),
		data.MasterPassword.ValueString(),
		data.Hostname.ValueString(),
		data.Port.ValueInt32(),
	)
	tflog.Info(ctx, "RDS instance URL", map[string]any{
		"conn_str": connStr,
	})

	return connStr
}

func connectDb(ctx context.Context, dbUrl string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, dbUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	return conn, nil
}

func createUser(ctx context.Context, conn *pgx.Conn) (string, string, error) {
	username := gen.Name()
	password := gen.Password()

	sql := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s' CREATEDB", username, password)
	_, err := conn.Exec(ctx, sql)

	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

func createDb(ctx context.Context, conn *pgx.Conn, dbName string, username string) error {
	sql := fmt.Sprintf(`
		CREATE DATABASE %s
			WITH OWNER = %s
			ENCODING = 'UTF8'
			LC_COLLATE = 'en_US.utf8'
			LC_CTYPE = 'en_US.utf8'
			TABLESPACE = pg_default
			CONNECTION LIMIT = -1
	`, dbName, username)
	_, err := conn.Exec(ctx, sql)

	return err
}

func createTable(ctx context.Context, conn *pgx.Conn, tableName string) error {
	sql := fmt.Sprintf(`
		CREATE TABLE %s (
    		id int8 PRIMARY KEY
     	)
	`, tableName)
	_, err := conn.Exec(ctx, sql)

	return err
}

func dropDb(ctx context.Context, conn *pgx.Conn, dbName string) error {
	sql := fmt.Sprintf("DROP DATABASE %s", dbName)
	_, err := conn.Exec(ctx, sql)

	return err
}

func getDbOid(ctx context.Context, conn *pgx.Conn, dbName string) (int32, error) {
	var dbOid int32
	err := conn.QueryRow(
		ctx,
		"SELECT oid::int4 FROM pg_catalog.pg_database WHERE lower(datname) = lower($1)",
		dbName,
	).
		Scan(&dbOid)
	if err != nil {
		return 0, fmt.Errorf("failed to get DB oid: %w", err)
	}

	return dbOid, nil
}
