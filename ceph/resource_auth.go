package ceph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &authResource{}
var _ resource.ResourceWithImportState = &authResource{}
var _ resource.ResourceWithConfigure = &authResource{}

type authResource struct {
	config *Config
}

func newAuthResource() resource.Resource {
	return &authResource{}
}

func (r *authResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth"
}

func (r *authResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.config = req.ProviderData.(*Config)
}

func (r *authResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This resource allows you to create a ceph client and retrieve its key and/or keyring.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity name, used as the resource identifier.",
			},
			"entity": schema.StringAttribute{
				Required:    true,
				Description: "The entity name (i.e.: client.admin)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"caps": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The caps of the entity",
			},
			"keyring": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The cephx keyring of the entity",
			},
			"key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The cephx key of the entity",
			},
		},
	}
}

type authResponse struct {
	Entity string            `json:"entity"`
	Key    string            `json:"key"`
	Caps   map[string]string `json:"caps"`
}

const clientKeyringFormat = `[%s]
	key = %s
`

func authModelFromResponse(ctx context.Context, responses []authResponse) (authModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(responses) == 0 {
		diags.AddError("Empty response", "No data returned by ceph auth command")
		return authModel{}, diags
	}
	r := responses[0]
	caps, d := types.MapValueFrom(ctx, types.StringType, r.Caps)
	diags.Append(d...)
	if diags.HasError() {
		return authModel{}, diags
	}
	return authModel{
		ID:      types.StringValue(r.Entity),
		Entity:  types.StringValue(r.Entity),
		Caps:    caps,
		Key:     types.StringValue(r.Key),
		Keyring: types.StringValue(fmt.Sprintf(clientKeyringFormat, r.Entity, r.Key)),
	}, diags
}

func toCapsArray(caps map[string]string) []string {
	ret := make([]string, 0, len(caps)*2)
	for key, val := range caps {
		ret = append(ret, key, val)
	}
	return ret
}

func (r *authResource) fetchFromCeph(ctx context.Context, entity string) (authModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	conn, err := r.config.GetCephConnection()
	if err != nil {
		diags.AddError("Unable to connect to Ceph", err.Error())
		return authModel{}, false, diags
	}

	command, err := json.Marshal(map[string]interface{}{
		"prefix": "auth get",
		"format": "json",
		"entity": entity,
	})
	if err != nil {
		diags.AddError("Error building auth get command", err.Error())
		return authModel{}, false, diags
	}

	buf, status, err := conn.MonCommand(command)
	if err != nil {
		if strings.Contains(status, "ENOENT") {
			return authModel{}, false, diags
		}
		diags.AddError("Error on auth get command", err.Error())
		return authModel{}, false, diags
	}

	var authResponses []authResponse
	if err = json.Unmarshal(buf, &authResponses); err != nil {
		diags.AddError("Error unmarshaling auth response", err.Error())
		return authModel{}, false, diags
	}

	model, d := authModelFromResponse(ctx, authResponses)
	diags.Append(d...)
	return model, !diags.HasError(), diags
}

func (r *authResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	var caps map[string]string
	resp.Diagnostics.Append(plan.Caps.ElementsAs(ctx, &caps, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	command, err := json.Marshal(map[string]interface{}{
		"prefix": "auth get-or-create",
		"format": "json",
		"entity": plan.Entity.ValueString(),
		"caps":   toCapsArray(caps),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building auth command", err.Error())
		return
	}

	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on get-or-create command", err.Error())
		return
	}

	model, _, diags := r.fetchFromCeph(ctx, plan.Entity.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *authResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, found, diags := r.fetchFromCeph(ctx, state.Entity.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *authResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	var caps map[string]string
	resp.Diagnostics.Append(plan.Caps.ElementsAs(ctx, &caps, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	command, err := json.Marshal(map[string]interface{}{
		"prefix": "auth caps",
		"format": "json",
		"entity": plan.Entity.ValueString(),
		"caps":   toCapsArray(caps),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building auth caps command", err.Error())
		return
	}

	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on auth caps command", err.Error())
		return
	}

	model, _, diags := r.fetchFromCeph(ctx, plan.Entity.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *authResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	command, err := json.Marshal(map[string]interface{}{
		"prefix": "auth rm",
		"format": "json",
		"entity": state.Entity.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building auth rm command", err.Error())
		return
	}

	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on auth rm command", err.Error())
		return
	}
}

func (r *authResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("entity"), req, resp)
}
