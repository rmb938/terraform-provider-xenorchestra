package xo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func resourceDisk() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiskCreate,
		ReadContext:   resourceDiskRead,
		UpdateContext: resourceDiskUpdate,
		DeleteContext: resourceDiskDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"storage_repository_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(xo_client.VDIModeRW),
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{string(xo_client.VDIModeRO), string(xo_client.VDIModeRW)}, false),
			},
		},
		CustomizeDiff: customdiff.ForceNewIfChange("size", func(ctx context.Context, old, new, meta interface{}) bool {
			return new.(int) < old.(int)
		}),
	}
}

func resourceDiskCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	mode := xo_client.VDIMode(d.Get("mode").(string))
	size := d.Get("size").(int) * 1024 * 1024 * 1024
	storageRepositoryID := d.Get("storage_repository_id").(string)

	storageRepository, err := c.GetStorageRepositoryByID(ctx, storageRepositoryID)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting storage repository",
				Detail:   err.Error(),
			},
		}
	}

	vdi, err := c.CreateVDI(ctx, name, mode, size, storageRepository)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error creating disk",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(vdi.ID)

	// need to update to set description (kind dumb I know)
	err = vdi.Update(c, ctx, nil, &description, nil)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error updating disk",
				Detail:   err.Error(),
			},
		}
	}

	return resourceDiskRead(ctx, d, m)
}

func resourceDiskRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	vdi, err := c.GetVDIByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting disk",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(vdi.ID)
	d.Set("name", vdi.Name)
	d.Set("description", vdi.Description)
	d.Set("size", vdi.Size/1024/1024/1024)
	d.Set("storage_repository_id", vdi.StorageRepositoryID)

	return nil
}

func resourceDiskUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	var name *string
	var description *string
	var size *int

	vdi, err := c.GetVDIByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting disk",
				Detail:   err.Error(),
			},
		}
	}

	if d.HasChange("name") {
		name = func(i string) *string { return &i }(d.Get("name").(string))
	}

	if d.HasChange("description") {
		name = func(i string) *string { return &i }(d.Get("description").(string))
	}

	if d.HasChange("size") {
		size = func(i int) *int { return &i }(d.Get("size").(int) * 1024 * 1024 * 1024)
	}

	err = vdi.Update(c, ctx, name, description, size)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error updating disk",
				Detail:   err.Error(),
			},
		}
	}

	return resourceDiskRead(ctx, d, m)
}

func resourceDiskDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	vdi, err := c.GetVDIByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting disk",
				Detail:   err.Error(),
			},
		}
	}

	err = vdi.Delete(c, ctx)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error deleting disk",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId("")

	return nil
}
