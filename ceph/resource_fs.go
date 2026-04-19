package ceph

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &fsResource{}
var _ resource.ResourceWithImportState = &fsResource{}
var _ resource.ResourceWithConfigure = &fsResource{}

type fsResource struct {
	config *Config
}

func newFSResource() resource.Resource {
	return &fsResource{}
}

func (r *fsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fs"
}

func (r *fsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.config = req.ProviderData.(*Config)
}

func (r *fsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CephFS filesystem. The metadata and data pools must already exist before creating the filesystem.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The filesystem name, used as the resource identifier.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the filesystem.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metadata_pool": schema.StringAttribute{
				Required:    true,
				Description: "Pool used for filesystem metadata. Changing this forces recreation of the filesystem.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data_pools": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Data pools for the filesystem. At least one is required.",
			},
		},
	}
}

type fsListEntry struct {
	Name         string   `json:"name"`
	MetadataPool string   `json:"metadata_pool"`
	DataPools    []string `json:"data_pools"`
}

func fsGet(conn monCommander, name string) (*fsListEntry, error) {
	command, err := json.Marshal(map[string]interface{}{
		"prefix": "fs ls",
		"format": "json",
	})
	if err != nil {
		return nil, err
	}
	buf, _, err := conn.MonCommand(command)
	if err != nil {
		return nil, err
	}
	var fsList []fsListEntry
	if err = json.Unmarshal(buf, &fsList); err != nil {
		return nil, err
	}
	for i := range fsList {
		if fsList[i].Name == name {
			return &fsList[i], nil
		}
	}
	return nil, nil
}

func fsAddDataPool(conn monCommander, fsName, pool string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix":  "fs add_data_pool",
		"fs_name": fsName,
		"pool":    pool,
		"format":  "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func fsRemoveDataPool(conn monCommander, fsName, pool string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix":  "fs rm_data_pool",
		"fs_name": fsName,
		"pool":    pool,
		"format":  "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func (r *fsResource) fetchFromCeph(ctx context.Context, name string) (fsModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	conn, err := r.config.GetCephConnection()
	if err != nil {
		diags.AddError("Unable to connect to Ceph", err.Error())
		return fsModel{}, false, diags
	}

	fs, err := fsGet(conn, name)
	if err != nil {
		diags.AddError(fmt.Sprintf("Error reading filesystem %q", name), err.Error())
		return fsModel{}, false, diags
	}
	if fs == nil {
		return fsModel{}, false, diags
	}

	dpSet, d := types.SetValueFrom(ctx, types.StringType, fs.DataPools)
	diags.Append(d...)
	if diags.HasError() {
		return fsModel{}, false, diags
	}

	return fsModel{
		ID:           types.StringValue(fs.Name),
		Name:         types.StringValue(fs.Name),
		MetadataPool: types.StringValue(fs.MetadataPool),
		DataPools:    dpSet,
	}, true, diags
}

func (r *fsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan fsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	var dataPools []string
	resp.Diagnostics.Append(plan.DataPools.ElementsAs(ctx, &dataPools, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(dataPools) == 0 {
		resp.Diagnostics.AddError("data_pools must contain at least one pool", "")
		return
	}

	name := plan.Name.ValueString()
	command, err := json.Marshal(map[string]interface{}{
		"prefix":   "fs new",
		"fs_name":  name,
		"metadata": plan.MetadataPool.ValueString(),
		"data":     dataPools[0],
		"format":   "json",
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building fs new command", err.Error())
		return
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on fs new command", err.Error())
		return
	}

	for _, pool := range dataPools[1:] {
		if err := fsAddDataPool(conn, name, pool); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding data pool %q", pool), err.Error())
			return
		}
	}

	model, found, diags := r.fetchFromCeph(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Filesystem not found after create", name)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *fsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state fsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, found, diags := r.fetchFromCeph(ctx, state.Name.ValueString())
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

func (r *fsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state fsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	name := plan.Name.ValueString()

	if !plan.DataPools.Equal(state.DataPools) {
		var planPools, statePools []string
		resp.Diagnostics.Append(plan.DataPools.ElementsAs(ctx, &planPools, false)...)
		resp.Diagnostics.Append(state.DataPools.ElementsAs(ctx, &statePools, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		toAdd, toRemove := sliceDiff(statePools, planPools)

		for _, pool := range toAdd {
			if err := fsAddDataPool(conn, name, pool); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Error adding data pool %q", pool), err.Error())
				return
			}
		}
		for _, pool := range toRemove {
			if err := fsRemoveDataPool(conn, name, pool); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Error removing data pool %q", pool), err.Error())
				return
			}
		}
	}

	model, found, diags := r.fetchFromCeph(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Filesystem not found after update", name)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *fsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state fsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	name := state.Name.ValueString()

	failCmd, err := json.Marshal(map[string]interface{}{
		"prefix":  "fs fail",
		"fs_name": name,
		"format":  "json",
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building fs fail command", err.Error())
		return
	}
	if _, _, err = conn.MonCommand(failCmd); err != nil {
		resp.Diagnostics.AddError("Error on fs fail command", err.Error())
		return
	}

	rmCmd, err := json.Marshal(map[string]interface{}{
		"prefix":               "fs rm",
		"fs_name":              name,
		"yes_i_really_mean_it": true,
		"format":               "json",
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building fs rm command", err.Error())
		return
	}
	if _, _, err = conn.MonCommand(rmCmd); err != nil {
		resp.Diagnostics.AddError("Error on fs rm command", err.Error())
		return
	}
}

func (r *fsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
