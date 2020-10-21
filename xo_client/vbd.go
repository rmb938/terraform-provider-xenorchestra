package xo_client

import (
	"context"
)

type VBD struct {
	ID       string `json:"id"`
	Bootable bool   `json:"bootable"`
	Device   string `json:"device"`
	CDDrive  bool   `json:"is_cd_drive"`
	Position string `json:"position"`
	VDI      string `json:"VDI"`
	VM       string `json:"VM"`
}

func (c *Client) GetVBDByID(ctx context.Context, id string) (*VBD, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "VBD", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(VBD{})
	if err != nil {
		return nil, err
	}

	VBD := interf.(VBD)
	return &VBD, nil
}

func (vbd *VBD) GetVDI(client *Client, ctx context.Context) (*VDI, error) {
	return client.GetVDIByID(ctx, vbd.VDI)
}

func (vbd *VBD) Delete(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vbd.ID,
	}

	return client.rpcConn.Call(ctx, "vbd.delete", params, nil)
}

func (vbd *VBD) Disconnect(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vbd.ID,
	}

	return client.rpcConn.Call(ctx, "vbd.disconnect", params, nil)
}
