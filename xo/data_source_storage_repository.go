package xo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func dataSourceStorageRepository() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageRepositoryRead,
		Schema: map[string]*schema.Schema{
			"pool_id": {
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

func dataSourceStorageRepositoryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	poolID := d.Get("pool_id").(string)
	name := d.Get("name").(string)
	template, err := c.GetStorageRepositoryByName(ctx, poolID, name)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting storage repository",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(template.ID)
	d.Set("description", template.Description)

	return nil
}
