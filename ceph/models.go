package ceph

import "github.com/hashicorp/terraform-plugin-framework/types"

type authModel struct {
	ID      types.String `tfsdk:"id"`
	Entity  types.String `tfsdk:"entity"`
	Caps    types.Map    `tfsdk:"caps"`
	Keyring types.String `tfsdk:"keyring"`
	Key     types.String `tfsdk:"key"`
}

type authDataSourceModel struct {
	Entity  types.String `tfsdk:"entity"`
	Caps    types.Map    `tfsdk:"caps"`
	Keyring types.String `tfsdk:"keyring"`
	Key     types.String `tfsdk:"key"`
}

type osdPoolModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	PgNum       types.Int64  `tfsdk:"pg_num"`
	Size        types.Int64  `tfsdk:"size"`
	MinSize     types.Int64  `tfsdk:"min_size"`
	CrushRule   types.String `tfsdk:"crush_rule"`
	Application types.List   `tfsdk:"application"`
}

// osdPoolDatasourceModel is the datasource variant — no `type` attribute.
type osdPoolDatasourceModel struct {
	Name        types.String `tfsdk:"name"`
	PgNum       types.Int64  `tfsdk:"pg_num"`
	Size        types.Int64  `tfsdk:"size"`
	MinSize     types.Int64  `tfsdk:"min_size"`
	CrushRule   types.String `tfsdk:"crush_rule"`
	Application types.List   `tfsdk:"application"`
}

type fsModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	MetadataPool types.String `tfsdk:"metadata_pool"`
	DataPools    types.Set    `tfsdk:"data_pools"`
}

type fsDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	MetadataPool types.String `tfsdk:"metadata_pool"`
	DataPools    types.Set    `tfsdk:"data_pools"`
}

type waitOnlineModel struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	Online      types.Bool   `tfsdk:"online"`
}
