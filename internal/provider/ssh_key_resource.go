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

// Ensure sshKeyResource satisfies the resource.Resource and
// resource.ResourceWithImportState interfaces.
var (
	_ resource.Resource                = &sshKeyResource{}
	_ resource.ResourceWithImportState = &sshKeyResource{}
)

// sshKeyResourceModel maps the schema attributes for exedev_ssh_key.
type sshKeyResourceModel struct {
	// ID is the SSH key name (comment field of the public key, or renamed value).
	ID types.String `tfsdk:"id"`
	// PublicKey is the full SSH public key string (e.g. "ssh-ed25519 AAAA... comment").
	PublicKey types.String `tfsdk:"public_key"`
	// Name is the key name (comment). Computed from the public key on creation;
	// can be updated via ssh-key rename.
	Name types.String `tfsdk:"name"`
	// Fingerprint is the SHA256 fingerprint of the key (computed).
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// sshKeyAPIResponse is used to decode entries from `ssh-key list --json`.
type sshKeyAPIResponse struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

// sshKeyResource is the concrete resource implementation.
type sshKeyResource struct {
	client *client.Client
}

// NewSSHKeyResource returns a new sshKeyResource factory function.
func NewSSHKeyResource() resource.Resource {
	return &sshKeyResource{}
}

func (r *sshKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an SSH key associated with an [exe.dev](https://exe.dev) account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The SSH key name, used as the stable identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SSH public key string (e.g. `ssh-ed25519 AAAA... my-key`). Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Key name. Defaults to the comment in the public key. Can be changed via `ssh-key rename`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SHA256 fingerprint of the SSH public key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *sshKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pubKey := plan.PublicKey.ValueString()
	cmd := fmt.Sprintf("ssh-key add %s", client.ShellQuote(pubKey))
	tflog.Debug(ctx, "Adding SSH key", map[string]any{"cmd": "ssh-key add [public key redacted]"})

	body, err := r.client.Exec(cmd)
	if err != nil {
		resp.Diagnostics.AddError("Adding SSH key", err.Error())
		return
	}

	// Parse the response to get key details.
	key, err := parseSSHKeyResponse(body)
	if err != nil {
		// Fall back to extracting the comment from the public key.
		key.Name = extractKeyComment(pubKey)
		key.PublicKey = pubKey
	}

	if key.Name == "" {
		key.Name = extractKeyComment(pubKey)
	}

	// If a name override was specified and differs from the detected name, rename.
	desiredName := plan.Name.ValueString()
	if desiredName != "" && desiredName != key.Name && key.Name != "" {
		renameCmd := fmt.Sprintf("ssh-key rename %s %s", client.ShellQuote(key.Name), client.ShellQuote(desiredName))
		tflog.Debug(ctx, "Renaming SSH key after add", map[string]any{"cmd": renameCmd})
		if _, renameErr := r.client.Exec(renameCmd); renameErr != nil {
			resp.Diagnostics.AddError("Renaming SSH key", renameErr.Error())
			return
		}
		key.Name = desiredName
	}

	plan.ID = types.StringValue(key.Name)
	plan.Name = types.StringValue(key.Name)
	plan.Fingerprint = types.StringValue(key.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, found, err := r.findKey(ctx, state.ID.ValueString(), state.Fingerprint.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading SSH key", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(key.Name)
	state.Name = types.StringValue(key.Name)
	state.Fingerprint = types.StringValue(key.Fingerprint)
	if key.PublicKey != "" {
		state.PublicKey = types.StringValue(key.PublicKey)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state sshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currentName := state.ID.ValueString()
	desiredName := plan.Name.ValueString()

	if desiredName != "" && desiredName != currentName {
		cmd := fmt.Sprintf("ssh-key rename %s %s", client.ShellQuote(currentName), client.ShellQuote(desiredName))
		tflog.Debug(ctx, "Renaming SSH key", map[string]any{"cmd": cmd})
		if _, err := r.client.Exec(cmd); err != nil {
			resp.Diagnostics.AddError("Renaming SSH key", err.Error())
			return
		}
		currentName = desiredName
	}

	plan.ID = types.StringValue(currentName)
	plan.Name = types.StringValue(currentName)
	plan.Fingerprint = state.Fingerprint // fingerprint doesn't change on rename

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identifier := state.ID.ValueString()
	if fp := state.Fingerprint.ValueString(); fp != "" {
		identifier = fp
	}

	cmd := fmt.Sprintf("ssh-key remove %s", client.ShellQuote(identifier))
	tflog.Debug(ctx, "Removing SSH key", map[string]any{"cmd": cmd})

	if _, err := r.client.Exec(cmd); err != nil {
		var execErr *client.ExecError
		if errors.As(err, &execErr) && (execErr.StatusCode == 404 || execErr.StatusCode == 422) {
			return // Already gone
		}
		resp.Diagnostics.AddError("Removing SSH key", err.Error())
	}
}

func (r *sshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// findKey looks up a key by name or fingerprint from `ssh-key list --json`.
func (r *sshKeyResource) findKey(ctx context.Context, name, fingerprint string) (sshKeyAPIResponse, bool, error) {
	cmd := "ssh-key list --json"
	tflog.Debug(ctx, "Listing SSH keys", map[string]any{"cmd": cmd})

	body, err := r.client.Exec(cmd)
	if err != nil {
		return sshKeyAPIResponse{}, false, err
	}

	body = trimBOM(body)

	var keys []sshKeyAPIResponse
	if err := json.Unmarshal(body, &keys); err != nil {
		// Maybe a single object
		var single sshKeyAPIResponse
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return sshKeyAPIResponse{}, false, fmt.Errorf("parsing ssh-key list response: %v", err)
		}
		keys = []sshKeyAPIResponse{single}
	}

	for _, k := range keys {
		if k.Name == name || (fingerprint != "" && k.Fingerprint == fingerprint) {
			return k, true, nil
		}
	}
	return sshKeyAPIResponse{}, false, nil
}

// parseSSHKeyResponse parses a single SSH key JSON object from an add response.
func parseSSHKeyResponse(body []byte) (sshKeyAPIResponse, error) {
	body = trimBOM(body)
	var key sshKeyAPIResponse
	if err := json.Unmarshal(body, &key); err != nil {
		return sshKeyAPIResponse{}, err
	}
	return key, nil
}

// extractKeyComment returns the comment (third field) from an SSH public key string.
func extractKeyComment(pubKey string) string {
	parts := strings.Fields(pubKey)
	if len(parts) >= 3 {
		return strings.Join(parts[2:], " ")
	}
	return ""
}
