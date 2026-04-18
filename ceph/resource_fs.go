package ceph

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type fsListEntry struct {
	Name         string   `json:"name"`
	MetadataPool string   `json:"metadata_pool"`
	DataPoolList []string `json:"data_pool_list"`
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
		"prefix":   "fs add_data_pool",
		"fs_name":  fsName,
		"poolname": pool,
		"format":   "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func fsRemoveDataPool(conn monCommander, fsName, pool string) error {
	command, err := json.Marshal(map[string]interface{}{
		"prefix":   "fs rm_data_pool",
		"fs_name":  fsName,
		"poolname": pool,
		"format":   "json",
	})
	if err != nil {
		return err
	}
	_, _, err = conn.MonCommand(command)
	return err
}

func resourceFS() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a CephFS filesystem. The metadata and data pools must already exist before creating the filesystem.",
		CreateContext: resourceFSCreate,
		ReadContext:   resourceFSRead,
		UpdateContext: resourceFSUpdate,
		DeleteContext: resourceFSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the filesystem.",
			},
			"metadata_pool": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Pool used for filesystem metadata. Changing this forces recreation of the filesystem.",
			},
			"data_pools": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Data pools for the filesystem. At least one is required.",
			},
		},
	}
}

func resourceFSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)
	metadataPool := d.Get("metadata_pool").(string)
	dataPools := d.Get("data_pools").(*schema.Set).List()

	command, err := json.Marshal(map[string]interface{}{
		"prefix":   "fs new",
		"fs_name":  name,
		"metadata": metadataPool,
		"data":     dataPools[0].(string),
		"format":   "json",
	})
	if err != nil {
		return diag.Errorf("Error resource_fs unable to create fs new JSON command: %s", err)
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		return diag.Errorf("Error resource_fs on fs new command: %s", err)
	}

	d.SetId(name)

	for _, pool := range dataPools[1:] {
		if err := fsAddDataPool(conn, name, pool.(string)); err != nil {
			return diag.Errorf("Error resource_fs adding data pool %q: %s", pool.(string), err)
		}
	}

	return resourceFSRead(ctx, d, meta)
}

func resourceFSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}

	fs, err := fsGet(conn, d.Id())
	if err != nil {
		return diag.Errorf("Error resource_fs reading filesystem %q: %s", d.Id(), err)
	}
	if fs == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("name", fs.Name); err != nil {
		return diag.Errorf("Unable to set name: %s", err)
	}
	if err := d.Set("metadata_pool", fs.MetadataPool); err != nil {
		return diag.Errorf("Unable to set metadata_pool: %s", err)
	}
	if err := d.Set("data_pools", fs.DataPoolList); err != nil {
		return diag.Errorf("Unable to set data_pools: %s", err)
	}

	return nil
}

func resourceFSUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	if d.HasChange("data_pools") {
		old, new := d.GetChange("data_pools")
		toAdd := new.(*schema.Set).Difference(old.(*schema.Set))
		toRemove := old.(*schema.Set).Difference(new.(*schema.Set))

		for _, pool := range toAdd.List() {
			if err := fsAddDataPool(conn, name, pool.(string)); err != nil {
				return diag.Errorf("Error resource_fs adding data pool %q: %s", pool.(string), err)
			}
		}
		for _, pool := range toRemove.List() {
			if err := fsRemoveDataPool(conn, name, pool.(string)); err != nil {
				return diag.Errorf("Error resource_fs removing data pool %q: %s", pool.(string), err)
			}
		}
	}

	return resourceFSRead(ctx, d, meta)
}

func resourceFSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	failCmd, err := json.Marshal(map[string]interface{}{
		"prefix":  "fs fail",
		"fs_name": name,
		"format":  "json",
	})
	if err != nil {
		return diag.Errorf("Error resource_fs unable to create fs fail JSON command: %s", err)
	}
	if _, _, err = conn.MonCommand(failCmd); err != nil {
		return diag.Errorf("Error resource_fs on fs fail command: %s", err)
	}

	rmCmd, err := json.Marshal(map[string]interface{}{
		"prefix":               "fs rm",
		"fs_name":              name,
		"yes_i_really_mean_it": true,
		"format":               "json",
	})
	if err != nil {
		return diag.Errorf("Error resource_fs unable to create fs rm JSON command: %s", err)
	}
	if _, _, err = conn.MonCommand(rmCmd); err != nil {
		return diag.Errorf("Error resource_fs on fs rm command: %s", err)
	}

	return nil
}
