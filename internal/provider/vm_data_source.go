package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/benjamin-lykins/terraform-provider-exedev/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure vmDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &vmDataSource{}

// vmDataSourceModel maps schema data for the exedev_vm data source.
type vmDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Image        types.String `tfsdk:"image"`
	Disk         types.String `tfsdk:"disk"`
	Hostname     types.String `tfsdk:"hostname"`
	Status       types.String `tfsdk:"status"`
	Region       types.String `tfsdk:"region"`
	Integrations types.List   `tfsdk:"integrations"`
	Env          types.Map    `tfsdk:"env"`
}

// vmDataSource reads a single exe.dev VM.
type vmDataSource struct {
	client *client.Client
}

// NewVMDataSource returns a new vmDataSource factory function.
func NewVMDataSource() datasource.DataSource {
	return &vmDataSource{}
}

func (d *vmDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *vmDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads information about an existing [exe.dev](https://exe.dev) VM.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The VM name (matches `name`).",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the VM to look up.",
			},
			"image": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Container image used by the VM.",
			},
			"disk": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current disk size.",
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Public hostname (e.g. `my-vm.exe.xyz`).",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current VM status.",
			},
			"region": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Region where the VM is running.",
			},
			"integrations": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Integrations attached to the VM.",
			},
			"env": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Environment variables set on the VM.",
			},
		},
	}
}

func (d *vmDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *vmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config vmDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	cmd := fmt.Sprintf("ls --l --json %s", client.ShellQuote(name))
	tflog.Debug(ctx, "Reading VM (data source)", map[string]any{"cmd": cmd})

	body, err := d.client.Exec(cmd)
	if err != nil {
		var execErr *client.ExecError
		if errors.As(err, &execErr) && (execErr.StatusCode == 404 || execErr.StatusCode == 422) {
			resp.Diagnostics.AddError("VM not found", fmt.Sprintf("No VM named %q found: %s", name, execErr.Body))
			return
		}
		resp.Diagnostics.AddError("Reading VM", err.Error())
		return
	}

	body = trimBOM(body)

	// Parse response - try single object, then array.
	var vm vmAPIResponse
	if err := json.Unmarshal(body, &vm); err != nil || vm.Name == "" {
		var list []vmAPIResponse
		if err2 := json.Unmarshal(body, &list); err2 != nil {
			resp.Diagnostics.AddError("Parsing VM response", fmt.Sprintf("parse error: %v\nBody: %s", err, body))
			return
		}
		for _, v := range list {
			if v.Name == name {
				vm = v
				break
			}
		}
	}

	if vm.Name == "" {
		resp.Diagnostics.AddError("VM not found", fmt.Sprintf("No VM named %q found in response", name))
		return
	}

	config.ID = types.StringValue(vm.Name)
	config.Name = types.StringValue(vm.Name)
	if vm.Hostname != "" {
		config.Hostname = types.StringValue(vm.Hostname)
	} else {
		config.Hostname = types.StringValue(vm.Name + ".exe.xyz")
	}
	config.Image = types.StringValue(vm.Image)
	config.Disk = types.StringValue(vm.Disk)
	config.Status = types.StringValue(vm.Status)
	config.Region = types.StringValue(vm.Region)

	if config.Integrations.IsNull() || config.Integrations.IsUnknown() {
		config.Integrations = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if config.Env.IsNull() || config.Env.IsUnknown() {
		config.Env = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
