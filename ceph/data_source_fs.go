package ceph

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFS() *schema.Resource {
	return &schema.Resource{
		Description: "This data source allows you to get information about an existing CephFS filesystem.",
		ReadContext: dataSourceFSRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the filesystem.",
			},
			"metadata_pool": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Pool used for filesystem metadata.",
			},
			"data_pools": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Data pools attached to the filesystem.",
			},
		},
	}
}

func dataSourceFSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	fs, err := fsGet(conn, name)
	if err != nil {
		return diag.Errorf("Error data_source_fs reading filesystem %q: %s", name, err)
	}
	if fs == nil {
		return diag.Errorf("Filesystem %q not found", name)
	}

	d.SetId(name)

	if err := d.Set("metadata_pool", fs.MetadataPool); err != nil {
		return diag.Errorf("Unable to set metadata_pool: %s", err)
	}
	if err := d.Set("data_pools", fs.DataPoolList); err != nil {
		return diag.Errorf("Unable to set data_pools: %s", err)
	}

	return nil
}
