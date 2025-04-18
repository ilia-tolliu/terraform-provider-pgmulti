package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &PgmultiProvider{}

// var _ provider.ProviderWithFunctions = &MultidbProvider{}
// var _ provider.ProviderWithEphemeralResources = &MultidbProvider{}

type PgmultiProvider struct {
	name    string
	version string
}

func NewPgmulti() provider.Provider {
	return &PgmultiProvider{
		name:    "pgmulti",
		version: "0.1",
	}
}

func (p *PgmultiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
}

func (p *PgmultiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
}

func (p *PgmultiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *PgmultiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewResourceDb,
	}
}

func (p *PgmultiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = p.name
	resp.Version = p.version
}
