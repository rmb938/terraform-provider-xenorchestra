package xo_client

import (
	"context"
)

type TemplateInfo struct {
	Disks []map[string]interface{} `json:"disks"`
}

type Template struct {
	ID           string       `json:"id"`
	Name         string       `json:"name_label"`
	Description  string       `json:"name_description"`
	VBDs         []string     `json:"$VBDs"`
	TemplateInfo TemplateInfo `json:"template_info"`
	Pool         string       `json:"$pool"`
}

func (c *Client) GetTemplateByID(ctx context.Context, id string) (*Template, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "VM-template", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(Template{})
	if err != nil {
		return nil, err
	}

	template := interf.(Template)
	return &template, nil
}

func (c *Client) GetTemplateByName(ctx context.Context, poolID, name string) (*Template, error) {
	query := ObjectQuery{
		"name_label": name,
		"$pool":      poolID,
	}

	objs, err := c.GetObjectsOfType(ctx, "VM-template", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(Template{})
	if err != nil {
		return nil, err
	}

	template := interf.(Template)
	return &template, nil
}

func (t *Template) GetVBDs(client *Client, ctx context.Context, includeCDDrive bool) ([]VBD, error) {
	var VBDs []VBD

	for _, VBDID := range t.VBDs {
		VBD, err := client.GetVBDByID(ctx, VBDID)
		if err != nil {
			return nil, err
		}

		if includeCDDrive == false && VBD.CDDrive == true {
			continue
		}

		VBDs = append(VBDs, *VBD)
	}

	return VBDs, nil
}
