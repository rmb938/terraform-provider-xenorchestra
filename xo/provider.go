package xo

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("XOA_URL", nil),
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("XOA_USER", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("XOA_PASSWORD", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"xenorchestra_virtual_machine": resourceVirtualMachine(),
			"xenorchestra_disk":            resourceDisk(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"xenorchestra_pool":               dataSourcePool(),
			"xenorchestra_template":           dataSourceTemplate(),
			"xenorchestra_disk":               dataSourceDisk(),
			"xenorchestra_storage_repository": dataSourceStorageRepository(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	urlString := d.Get("url").(string)
	username := d.Get("username").(string)
	password := d.Get("password").(string)

	var diags diag.Diagnostics

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid URL",
			Detail:   err.Error(),
		})
		return nil, diags
	}

	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid URL",
			Detail:   "Scheme must be ws or wss",
		})
		return nil, diags
	}

	c, err := xo_client.NewClient(parsedURL)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create XO Client",
			Detail:   err.Error(),
		})
		return nil, diags
	}

	err = c.SignIn(ctx, username, password)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error signing into XO API",
			Detail:   err.Error(),
		})
		return nil, diags
	}

	return c, diags
}
