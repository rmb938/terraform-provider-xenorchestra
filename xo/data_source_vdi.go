package xo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func dataSourceDisk() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiskRead,
		Schema: map[string]*schema.Schema{
			"storage_repository_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDiskRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	storageRepositoryID := d.Get("storage_repository_id").(string)
	name := d.Get("name").(string)
	vdi, err := c.GetVDIByName(ctx, storageRepositoryID, name)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting storage repository",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(vdi.ID)
	d.Set("description", vdi.Description)

	return nil
}
