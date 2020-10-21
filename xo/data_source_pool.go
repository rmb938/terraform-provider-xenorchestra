package xo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func dataSourcePool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePoolRead,
		Schema: map[string]*schema.Schema{
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

func dataSourcePoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	name := d.Get("name").(string)
	pool, err := c.GetPoolByName(ctx, name)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting pool",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(pool.ID)
	d.Set("description", pool.Description)

	return nil
}
