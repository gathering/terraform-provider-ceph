package ceph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/rbd"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &osdPoolResource{}
var _ resource.ResourceWithImportState = &osdPoolResource{}
var _ resource.ResourceWithConfigure = &osdPoolResource{}

type osdPoolResource struct {
	config *Config
}

func newOSDPoolResource() resource.Resource {
	return &osdPoolResource{}
}

func (r *osdPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_osd_pool"
}

func (r *osdPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.config = req.ProviderData.(*Config)
}

func (r *osdPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ceph OSD pool. Pool deletion requires mon_allow_pool_delete = true in the Ceph configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The pool name, used as the resource identifier.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The pool type: replicated. Defaults to replicated. Currently only replicated pools are supported.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("replicated"),
				},
			},
			"pg_num": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of placement groups. Uses the cluster default when not set.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"size": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Replication factor (replicated pools only). Uses the cluster default when not set.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"min_size": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Minimum number of replicas required for I/O (replicated pools only). Uses the cluster default when not set.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"crush_rule": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "CRUSH rule name for the pool. Uses the cluster default when not set.",
			},
			"application": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Application tags enabled on the pool (e.g. rbd, cephfs, rgw). When rbd is included the pool is also initialized with rbd pool init.",
			},
		},
	}
}

type poolGetAllResponse struct {
	Pool      string `json:"pool"`
	PoolID    int    `json:"pool_id"`
	Size      int    `json:"size"`
	MinSize   int    `json:"min_size"`
	PgNum     int    `json:"pg_num"`
	CrushRule string `json:"crush_rule"`
}

// monCommander is satisfied by *rados.Conn without importing the rados package here.
type monCommander interface {
	MonCommand([]byte) ([]byte, string, error)
}

func osdPoolSet(conn monCommander, pool, variable, value string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix": "osd pool set",
		"pool":   pool,
		"var":    variable,
		"val":    value,
		"format": "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func osdPoolGetAll(conn monCommander, name string) (*poolGetAllResponse, string, error) {
	command, err := json.Marshal(map[string]interface{}{
		"prefix": "osd pool get",
		"pool":   name,
		"var":    "all",
		"format": "json",
	})
	if err != nil {
		return nil, "", err
	}
	buf, status, err := conn.MonCommand(command)
	if err != nil {
		return nil, status, err
	}
	var pool poolGetAllResponse
	if err = json.Unmarshal(buf, &pool); err != nil {
		return nil, status, err
	}
	return &pool, status, nil
}

func osdPoolApplicationEnable(conn monCommander, pool, app string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix": "osd pool application enable",
		"pool":   pool,
		"app":    app,
		"format": "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func osdPoolApplicationDisable(conn monCommander, pool, app string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix":               "osd pool application disable",
		"pool":                 pool,
		"app":                  app,
		"yes_i_really_mean_it": true,
		"format":               "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func osdPoolApplicationGet(conn monCommander, pool string) ([]string, error) {
	command, err := json.Marshal(map[string]interface{}{
		"prefix": "osd pool application get",
		"pool":   pool,
		"format": "json",
	})
	if err != nil {
		return nil, err
	}
	buf, _, err := conn.MonCommand(command)
	if err != nil {
		return nil, err
	}
	var apps map[string]interface{}
	if err = json.Unmarshal(buf, &apps); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(apps))
	for name := range apps {
		names = append(names, name)
	}
	return names, nil
}

// rbdPoolInit prepares a pool to host RBD images, equivalent to `rbd pool init`.
func rbdPoolInit(config *Config, poolName string) error {
	conn, err := config.GetCephConnection()
	if err != nil {
		return err
	}
	ioctx, err := conn.OpenIOContext(poolName)
	if err != nil {
		return err
	}
	initErr := rbd.PoolInit(ioctx, false)
	ioctx.Destroy()
	return initErr
}

func (r *osdPoolResource) fetchFromCeph(ctx context.Context, name string) (osdPoolModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	conn, err := r.config.GetCephConnection()
	if err != nil {
		diags.AddError("Unable to connect to Ceph", err.Error())
		return osdPoolModel{}, false, diags
	}

	pool, status, err := osdPoolGetAll(conn, name)
	if err != nil {
		if strings.Contains(status, "ENOENT") {
			return osdPoolModel{}, false, diags
		}
		diags.AddError(fmt.Sprintf("Error reading pool %q", name), err.Error())
		return osdPoolModel{}, false, diags
	}

	apps, err := osdPoolApplicationGet(conn, name)
	if err != nil {
		diags.AddError(fmt.Sprintf("Error reading applications for pool %q", name), err.Error())
		return osdPoolModel{}, false, diags
	}

	appList, d := types.ListValueFrom(ctx, types.StringType, apps)
	diags.Append(d...)
	if diags.HasError() {
		return osdPoolModel{}, false, diags
	}

	model := osdPoolModel{
		ID:          types.StringValue(pool.Pool),
		Name:        types.StringValue(pool.Pool),
		Type:        types.StringValue("replicated"),
		PgNum:       types.Int64Value(int64(pool.PgNum)),
		Size:        types.Int64Value(int64(pool.Size)),
		MinSize:     types.Int64Value(int64(pool.MinSize)),
		CrushRule:   types.StringValue(pool.CrushRule),
		Application: appList,
	}
	return model, true, diags
}

func (r *osdPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan osdPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	poolType := "replicated"
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		poolType = plan.Type.ValueString()
	}

	createCmd := map[string]interface{}{
		"prefix":    "osd pool create",
		"pool":      plan.Name.ValueString(),
		"pool_type": poolType,
		"format":    "json",
	}
	if !plan.PgNum.IsNull() && !plan.PgNum.IsUnknown() {
		createCmd["pg_num"] = plan.PgNum.ValueInt64()
	}

	command, err := json.Marshal(createCmd)
	if err != nil {
		resp.Diagnostics.AddError("Error building pool create command", err.Error())
		return
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on pool create command", err.Error())
		return
	}

	name := plan.Name.ValueString()

	if !plan.Size.IsNull() && !plan.Size.IsUnknown() {
		if err := osdPoolSet(conn, name, "size", fmt.Sprintf("%d", plan.Size.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error setting size", err.Error())
			return
		}
	}
	if !plan.MinSize.IsNull() && !plan.MinSize.IsUnknown() {
		if err := osdPoolSet(conn, name, "min_size", fmt.Sprintf("%d", plan.MinSize.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error setting min_size", err.Error())
			return
		}
	}
	if !plan.CrushRule.IsNull() && !plan.CrushRule.IsUnknown() {
		if err := osdPoolSet(conn, name, "crush_rule", plan.CrushRule.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error setting crush_rule", err.Error())
			return
		}
	}

	var apps []string
	if !plan.Application.IsNull() && !plan.Application.IsUnknown() {
		resp.Diagnostics.Append(plan.Application.ElementsAs(ctx, &apps, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	for _, app := range apps {
		if err := osdPoolApplicationEnable(conn, name, app); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error enabling application %q", app), err.Error())
			return
		}
		if app == "rbd" {
			if err := rbdPoolInit(r.config, name); err != nil {
				resp.Diagnostics.AddWarning("rbd pool init failed",
					fmt.Sprintf("Could not initialize pool %q for RBD: %s. Manual initialization may be required.", name, err))
			}
		}
	}

	model, found, diags := r.fetchFromCeph(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Pool not found after create", name)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *osdPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state osdPoolModel
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

func (r *osdPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state osdPoolModel
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

	if !plan.PgNum.Equal(state.PgNum) {
		if err := osdPoolSet(conn, name, "pg_num", fmt.Sprintf("%d", plan.PgNum.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error updating pg_num", err.Error())
			return
		}
	}
	if !plan.Size.Equal(state.Size) {
		if err := osdPoolSet(conn, name, "size", fmt.Sprintf("%d", plan.Size.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error updating size", err.Error())
			return
		}
	}
	if !plan.MinSize.Equal(state.MinSize) {
		if err := osdPoolSet(conn, name, "min_size", fmt.Sprintf("%d", plan.MinSize.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error updating min_size", err.Error())
			return
		}
	}
	if !plan.CrushRule.Equal(state.CrushRule) {
		if err := osdPoolSet(conn, name, "crush_rule", plan.CrushRule.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating crush_rule", err.Error())
			return
		}
	}

	if !plan.Application.Equal(state.Application) {
		var planApps, stateApps []string
		resp.Diagnostics.Append(plan.Application.ElementsAs(ctx, &planApps, false)...)
		resp.Diagnostics.Append(state.Application.ElementsAs(ctx, &stateApps, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		toAdd, toRemove := sliceDiff(stateApps, planApps)

		for _, app := range toRemove {
			if err := osdPoolApplicationDisable(conn, name, app); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Error disabling application %q", app), err.Error())
				return
			}
		}
		for _, app := range toAdd {
			if err := osdPoolApplicationEnable(conn, name, app); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Error enabling application %q", app), err.Error())
				return
			}
			if app == "rbd" {
				if err := rbdPoolInit(r.config, name); err != nil {
					resp.Diagnostics.AddWarning("rbd pool init failed",
						fmt.Sprintf("Could not initialize pool %q for RBD: %s. Manual initialization may be required.", name, err))
				}
			}
		}
	}

	model, found, diags := r.fetchFromCeph(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Pool not found after update", name)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *osdPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state osdPoolModel
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
	command, err := json.Marshal(map[string]interface{}{
		"prefix":                      "osd pool delete",
		"pool":                        name,
		"pool2":                       name,
		"yes_i_really_really_mean_it": true,
		"format":                      "json",
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building pool delete command", err.Error())
		return
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		resp.Diagnostics.AddError("Error on pool delete command", err.Error())
		return
	}
}

func (r *osdPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
