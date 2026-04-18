package ceph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/rbd"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

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
// It uses meta to obtain a connection so the caller does not need to import rados.
func rbdPoolInit(meta interface{}, poolName string) error {
	conn, err := meta.(*Config).GetCephConnection()
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

func resourceOSDPool() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Ceph OSD pool. Pool deletion requires mon_allow_pool_delete = true in the Ceph configuration.",
		CreateContext: resourceOSDPoolCreate,
		ReadContext:   resourceOSDPoolRead,
		UpdateContext: resourceOSDPoolUpdate,
		DeleteContext: resourceOSDPoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the pool.",
			},
			"type": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          "replicated",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"replicated"}, false)),
				Description:      "The pool type: replicated. Defaults to replicated. Currently only replicated pools are supported.",
			},
			"pg_num": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "Number of placement groups. Uses the cluster default when not set.",
			},
			"size": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "Replication factor (replicated pools only). Uses the cluster default when not set.",
			},
			"min_size": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "Minimum number of replicas required for I/O (replicated pools only). Uses the cluster default when not set.",
			},
			"crush_rule": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "CRUSH rule name for the pool. Uses the cluster default when not set.",
			},
			"application": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Description: "Application tags enabled on the pool (e.g. rbd, cephfs, rgw). " +
					"When rbd is included the pool is also initialized with rbd pool init.",
			},
		},
	}
}

func resourceOSDPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	createCmd := map[string]interface{}{
		"prefix":    "osd pool create",
		"pool":      name,
		"pool_type": d.Get("type").(string),
		"format":    "json",
	}
	if v, ok := d.GetOk("pg_num"); ok {
		createCmd["pg_num"] = v.(int)
	}

	command, err := json.Marshal(createCmd)
	if err != nil {
		return diag.Errorf("Error resource_osd_pool unable to create pool creation JSON command: %s", err)
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		return diag.Errorf("Error resource_osd_pool on pool create command: %s", err)
	}

	d.SetId(name)

	if v, ok := d.GetOk("size"); ok {
		if err := osdPoolSet(conn, name, "size", fmt.Sprintf("%d", v.(int))); err != nil {
			return diag.Errorf("Error resource_osd_pool setting size: %s", err)
		}
	}
	if v, ok := d.GetOk("min_size"); ok {
		if err := osdPoolSet(conn, name, "min_size", fmt.Sprintf("%d", v.(int))); err != nil {
			return diag.Errorf("Error resource_osd_pool setting min_size: %s", err)
		}
	}
	if v, ok := d.GetOk("crush_rule"); ok {
		if err := osdPoolSet(conn, name, "crush_rule", v.(string)); err != nil {
			return diag.Errorf("Error resource_osd_pool setting crush_rule: %s", err)
		}
	}

	for _, app := range d.Get("application").(*schema.Set).List() {
		appName := app.(string)
		if err := osdPoolApplicationEnable(conn, name, appName); err != nil {
			return diag.Errorf("Error resource_osd_pool enabling application %q: %s", appName, err)
		}
		if appName == "rbd" {
			if err := rbdPoolInit(meta, name); err != nil {
				return diag.Errorf("Error resource_osd_pool on rbd pool init: %s", err)
			}
		}
	}

	return resourceOSDPoolRead(ctx, d, meta)
}

func resourceOSDPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Id()

	pool, status, err := osdPoolGetAll(conn, name)
	if err != nil {
		if strings.Contains(status, "ENOENT") {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error resource_osd_pool reading pool %q: %s", name, err)
	}

	if err := d.Set("name", pool.Pool); err != nil {
		return diag.Errorf("Unable to set name: %s", err)
	}
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
		return diag.Errorf("Error resource_osd_pool reading applications for pool %q: %s", name, err)
	}
	if err := d.Set("application", apps); err != nil {
		return diag.Errorf("Unable to set application: %s", err)
	}

	return nil
}

func resourceOSDPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	if d.HasChange("pg_num") {
		if err := osdPoolSet(conn, name, "pg_num", fmt.Sprintf("%d", d.Get("pg_num").(int))); err != nil {
			return diag.Errorf("Error resource_osd_pool updating pg_num: %s", err)
		}
	}
	if d.HasChange("size") {
		if err := osdPoolSet(conn, name, "size", fmt.Sprintf("%d", d.Get("size").(int))); err != nil {
			return diag.Errorf("Error resource_osd_pool updating size: %s", err)
		}
	}
	if d.HasChange("min_size") {
		if err := osdPoolSet(conn, name, "min_size", fmt.Sprintf("%d", d.Get("min_size").(int))); err != nil {
			return diag.Errorf("Error resource_osd_pool updating min_size: %s", err)
		}
	}
	if d.HasChange("crush_rule") {
		if err := osdPoolSet(conn, name, "crush_rule", d.Get("crush_rule").(string)); err != nil {
			return diag.Errorf("Error resource_osd_pool updating crush_rule: %s", err)
		}
	}

	if d.HasChange("application") {
		old, new := d.GetChange("application")
		toRemove := old.(*schema.Set).Difference(new.(*schema.Set))
		toAdd := new.(*schema.Set).Difference(old.(*schema.Set))

		for _, app := range toRemove.List() {
			if err := osdPoolApplicationDisable(conn, name, app.(string)); err != nil {
				return diag.Errorf("Error resource_osd_pool disabling application %q: %s", app.(string), err)
			}
		}
		for _, app := range toAdd.List() {
			appName := app.(string)
			if err := osdPoolApplicationEnable(conn, name, appName); err != nil {
				return diag.Errorf("Error resource_osd_pool enabling application %q: %s", appName, err)
			}
			if appName == "rbd" {
				if err := rbdPoolInit(meta, name); err != nil {
					return diag.Errorf("Error resource_osd_pool on rbd pool init: %s", err)
				}
			}
		}
	}

	return resourceOSDPoolRead(ctx, d, meta)
}

func resourceOSDPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(*Config).GetCephConnection()
	if err != nil {
		return diag.Errorf("Unable to connect to Ceph: %s", err)
	}
	name := d.Get("name").(string)

	command, err := json.Marshal(map[string]interface{}{
		"prefix":                      "osd pool delete",
		"pool":                        name,
		"pool2":                       name,
		"yes_i_really_really_mean_it": true,
		"format":                      "json",
	})
	if err != nil {
		return diag.Errorf("Error resource_osd_pool unable to create delete JSON command: %s", err)
	}
	if _, _, err = conn.MonCommand(command); err != nil {
		return diag.Errorf("Error resource_osd_pool on pool delete command: %s", err)
	}

	return nil
}
