package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/benjamin-lykins/terraform-provider-exedev/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure domainResource satisfies the resource.Resource and
// resource.ResourceWithImportState interfaces.
var (
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

// domainResourceModel maps the schema attributes for exedev_domain.
type domainResourceModel struct {
	// ID is "vm_name/domain".
	ID types.String `tfsdk:"id"`
	// VMName is the exe.dev VM that should serve the custom domain.
	VMName types.String `tfsdk:"vm_name"`
	// Domain is the custom hostname registered with exe.dev.
	Domain types.String `tfsdk:"domain"`
}

// domainAPIResponse is used to decode entries from `domain ls --json`.
type domainAPIResponse struct {
	VMName string `json:"vm_name"`
	Domain string `json:"domain"`
}

// domainListResponse is the envelope returned by `domain ls --json`.
type domainListResponse struct {
	VMName  string              `json:"vm_name"`
	Domains []domainAPIResponse `json:"domains"`
}

// domainResource is the concrete resource implementation.
type domainResource struct {
	client *client.Client
}

// NewDomainResource returns a new domainResource factory function.
func NewDomainResource() resource.Resource {
	return &domainResource{}
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers a custom domain for an [exe.dev](https://exe.dev) VM. " +
			"Create the DNS CNAME or ALIAS record first; exe.dev verifies DNS during `domain add`. " +
			"Cloudflare records must stay DNS-only, not proxied.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable identifier in `vm_name/domain` form.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "exe.dev VM name that should serve the custom domain. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Custom domain to register with exe.dev. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = c
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName := plan.VMName.ValueString()
	domain := plan.Domain.ValueString()

	if existing, found, err := r.findDomain(ctx, vmName, domain); err != nil {
		resp.Diagnostics.AddError("Reading custom domain before create", err.Error())
		return
	} else if found {
		applyDomainResponse(&plan, existing)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	cmd := fmt.Sprintf("domain add %s %s --json", client.ShellQuote(vmName), client.ShellQuote(domain))
	tflog.Debug(ctx, "Adding custom domain", map[string]any{"cmd": cmd})

	if _, err := r.client.Exec(cmd); err != nil {
		resp.Diagnostics.AddError("Adding custom domain", err.Error())
		return
	}

	applyDomainResponse(&plan, domainAPIResponse{VMName: vmName, Domain: domain})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName, domain, err := domainStateIdentity(state)
	if err != nil {
		resp.Diagnostics.AddError("Reading custom domain", err.Error())
		return
	}

	entry, found, err := r.findDomain(ctx, vmName, domain)
	if err != nil {
		resp.Diagnostics.AddError("Reading custom domain", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	applyDomainResponse(&state, entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyDomainResponse(&plan, domainAPIResponse{
		VMName: plan.VMName.ValueString(),
		Domain: plan.Domain.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName, domain, err := domainStateIdentity(state)
	if err != nil {
		resp.Diagnostics.AddError("Removing custom domain", err.Error())
		return
	}

	cmd := fmt.Sprintf("domain rm %s %s --json", client.ShellQuote(vmName), client.ShellQuote(domain))
	tflog.Debug(ctx, "Removing custom domain", map[string]any{"cmd": cmd})

	if _, err := r.client.Exec(cmd); err != nil {
		var execErr *client.ExecError
		if errors.As(err, &execErr) && (execErr.StatusCode == 404 || execErr.StatusCode == 422) {
			return // Already gone.
		}
		resp.Diagnostics.AddError("Removing custom domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// findDomain looks up a custom domain for a VM from `domain ls <vm> --json`.
func (r *domainResource) findDomain(ctx context.Context, vmName, domain string) (domainAPIResponse, bool, error) {
	cmd := fmt.Sprintf("domain ls %s --json", client.ShellQuote(vmName))
	tflog.Debug(ctx, "Listing custom domains", map[string]any{"cmd": cmd})

	body, err := r.client.Exec(cmd)
	if err != nil {
		var execErr *client.ExecError
		if errors.As(err, &execErr) && (execErr.StatusCode == 404 || execErr.StatusCode == 422) {
			return domainAPIResponse{}, false, nil
		}
		return domainAPIResponse{}, false, err
	}

	domains, err := parseDomainList(body)
	if err != nil {
		return domainAPIResponse{}, false, fmt.Errorf("parsing domain ls response: %w", err)
	}

	for _, entry := range domains {
		if entry.VMName == vmName && entry.Domain == domain {
			return entry, true, nil
		}
	}
	return domainAPIResponse{}, false, nil
}

func parseDomainList(body []byte) ([]domainAPIResponse, error) {
	body = trimBOM(body)

	var envelope domainListResponse
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Domains != nil {
		for i := range envelope.Domains {
			if envelope.Domains[i].VMName == "" {
				envelope.Domains[i].VMName = envelope.VMName
			}
		}
		return envelope.Domains, nil
	}

	var list []domainAPIResponse
	if err := json.Unmarshal(body, &list); err == nil {
		return list, nil
	}

	var single domainAPIResponse
	if err := json.Unmarshal(body, &single); err == nil && single.Domain != "" {
		return []domainAPIResponse{single}, nil
	}

	return nil, fmt.Errorf("unrecognised response shape: %s", body)
}

func domainStateIdentity(state domainResourceModel) (string, string, error) {
	vmName := ""
	if !state.VMName.IsNull() && !state.VMName.IsUnknown() {
		vmName = state.VMName.ValueString()
	}

	domain := ""
	if !state.Domain.IsNull() && !state.Domain.IsUnknown() {
		domain = state.Domain.ValueString()
	}

	if vmName != "" && domain != "" {
		return vmName, domain, nil
	}

	id := ""
	if !state.ID.IsNull() && !state.ID.IsUnknown() {
		id = state.ID.ValueString()
	}
	if id == "" {
		return "", "", errors.New("missing custom domain identity")
	}
	return splitDomainResourceID(id)
}

func domainResourceID(vmName, domain string) string {
	return vmName + "/" + domain
}

func splitDomainResourceID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import id in vm_name/domain form, got %q", id)
	}
	return parts[0], parts[1], nil
}

func applyDomainResponse(m *domainResourceModel, domain domainAPIResponse) {
	m.ID = types.StringValue(domainResourceID(domain.VMName, domain.Domain))
	m.VMName = types.StringValue(domain.VMName)
	m.Domain = types.StringValue(domain.Domain)
}
