package ceph

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOSDPool() *schema.Resource {
	return &schema.Resource{
		Description: "This data source allows you to get information about an existing Ceph OSD pool.",
		ReadContext: dataSourceOSDPoolRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the pool.",
			},
			"pg_num": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of placement groups.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Replication factor.",
			},
			"min_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Minimum number of replicas required for I/O.",
			},
			"crush_rule": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CRUSH rule name for the pool.",
			},
			"application": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Application tags enabled on the pool (e.g. rbd, cephfs, rgw).",
			},
		},
	}
}

func dataSourceOSDPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	pool, status, err := osdPoolGetAll(conn, name)
	if err != nil {
		if strings.Contains(status, "ENOENT") {
			return diag.Errorf("Pool %q not found", name)
		}
		return diag.Errorf("Error data_source_osd_pool reading pool %q: %s", name, err)
	}

	d.SetId(name)

	if err := d.Set("pg_num", pool.PgNum); err != nil {
		return diag.Errorf("Unable to set pg_num: %s", err)
	}
	if err := d.Set("size", pool.Size); err != nil {
		return diag.Errorf("Unable to set size: %s", err)
	}
	if err := d.Set("min_size", pool.MinSize); err != nil {
		return diag.Errorf("Unable to set min_size: %s", err)
	}
	if err := d.Set("crush_rule", pool.CrushRule); err != nil {
		return diag.Errorf("Unable to set crush_rule: %s", err)
	}

	apps, err := osdPoolApplicationGet(conn, name)
	if err != nil {
		return diag.Errorf("Error data_source_osd_pool reading applications for pool %q: %s", name, err)
	}
	if err := d.Set("application", apps); err != nil {
		return diag.Errorf("Unable to set application: %s", err)
	}

	return nil
}
