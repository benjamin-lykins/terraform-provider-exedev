// Package provider implements the exe.dev Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/benjamin-lykins/terraform-provider-exedev/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ExeDevProvider satisfies the provider.Provider interface.
var _ provider.Provider = &ExeDevProvider{}

// ExeDevProvider is the provider implementation.
type ExeDevProvider struct{}

// ExeDevProviderModel maps provider schema data.
type ExeDevProviderModel struct {
	Token types.String `tfsdk:"token"`
}

// New returns a new ExeDevProvider for use by the Terraform Plugin Framework.
func New() provider.Provider {
	return &ExeDevProvider{}
}

func (p *ExeDevProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "exedev"
}

func (p *ExeDevProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing [exe.dev](https://exe.dev) resources via the HTTPS API.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Bearer token for the exe.dev HTTPS API. " +
					"Can also be set via the `EXEDEV_TOKEN` environment variable. " +
					"Generate a token with: `ssh-key generate-api-key --label=terraform --exp=365d`",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *ExeDevProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ExeDevProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("EXEDEV_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing exe.dev API token",
			"Set the `token` provider attribute or the `EXEDEV_TOKEN` environment variable. "+
				"Generate a token with: ssh-key generate-api-key --label=terraform --exp=365d",
		)
		return
	}

	c := client.New(token)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *ExeDevProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVMResource,
		NewSSHKeyResource,
	}
}

func (p *ExeDevProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVMDataSource,
	}
}
