// Package provider implements the dmn Terraform provider (functions only).
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type dmnProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &dmnProvider{version: version} }
}

func (p *dmnProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dmn"
	resp.Version = p.version
}

func (p *dmnProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *dmnProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *dmnProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *dmnProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *dmnProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		NewEvaluateFunction,
	}
}
