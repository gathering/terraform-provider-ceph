package ceph

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &waitOnlineResource{}
var _ resource.ResourceWithConfigure = &waitOnlineResource{}

type waitOnlineResource struct {
	config *Config
}

func newWaitOnlineResource() resource.Resource {
	return &waitOnlineResource{}
}

func (r *waitOnlineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wait_online"
}

func (r *waitOnlineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.config = req.ProviderData.(*Config)
}

func (r *waitOnlineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This dummy resource is waiting for Ceph to be online at creation time for up to 1 hour. This is useful for example on a bootstrap procedure.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "That's a workaround to actually have an id, set this to something unique (i.e.: the cluster name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"online": schema.BoolAttribute{
				Computed:    true,
				Description: "If the cluster is online, only checked at creation time (always true)",
			},
		},
	}
}

func (r *waitOnlineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan waitOnlineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "starting ceph_wait_online")

	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	for {
		_, err := r.config.GetCephConnection()
		if err == nil {
			tflog.Debug(ctx, "ceph_wait_online: cluster is online")
			break
		}
		tflog.Debug(ctx, "ceph_wait_online: cluster not yet reachable", map[string]any{"error": err.Error()})
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Timeout waiting for Ceph to come online", ctx.Err().Error())
			return
		case <-time.After(10 * time.Second):
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, waitOnlineModel{
		ClusterName: plan.ClusterName,
		Online:      types.BoolValue(true),
	})...)
}

func (r *waitOnlineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// State is stable after creation; no live check on read.
}

func (r *waitOnlineResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes are RequiresReplace or Computed; Update is never called.
}

func (r *waitOnlineResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Nothing to delete — this resource has no Ceph-side representation.
}
