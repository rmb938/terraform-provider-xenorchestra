package xo_client

import (
	"context"
)

type VDIMode string

var (
	VDIModeRO VDIMode = "RO"
	VDIModeRW VDIMode = "RW"
)

type VDI struct {
	ID                  string `json:"id"`
	Name                string `json:"name_label"`
	Description         string `json:"name_description"`
	Size                int    `json:"size"`
	StorageRepositoryID string `json:"$SR"`
	Pool                string `json:"$pool"`
}

func (c *Client) CreateVDI(ctx context.Context, name string, mode VDIMode, size int, storageRepository *StorageRepository) (*VDI, error) {
	params := map[string]interface{}{
		"mode": mode,
		"name": name,
		"sr":   storageRepository.ID,
		"size": size,
	}

	var vdiID string
	err := c.rpcConn.Call(ctx, "disk.create", params, &vdiID)
	if err != nil {
		return nil, err
	}

	return c.GetVDIByID(ctx, vdiID)
}

func (c *Client) GetVDIByName(ctx context.Context, storageRepositoryID, name string) (*VDI, error) {
	query := ObjectQuery{
		"$SR":        storageRepositoryID,
		"name_label": name,
	}

	objs, err := c.GetObjectsOfType(ctx, "VDI", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(VDI{})
	if err != nil {
		return nil, err
	}

	VDI := interf.(VDI)
	return &VDI, nil
}

func (c *Client) GetVDIByID(ctx context.Context, id string) (*VDI, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "VDI", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(VDI{})
	if err != nil {
		return nil, err
	}

	VDI := interf.(VDI)
	return &VDI, nil
}

func (vdi *VDI) Update(client *Client, ctx context.Context, name, description *string, size *int) error {
	params := map[string]interface{}{
		"id": vdi.ID,
	}

	if name != nil {
		params["name_label"] = name
	}

	if description != nil {
		params["name_description"] = description
	}

	if size != nil {
		params["size"] = size
	}

	return client.rpcConn.Call(ctx, "vdi.set", params, nil)
}

func (vdi *VDI) Delete(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vdi.ID,
	}

	return client.rpcConn.Call(ctx, "vdi.delete", params, nil)
}
