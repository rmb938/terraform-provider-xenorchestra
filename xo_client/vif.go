package xo_client

import (
	"context"
)

type VIF struct {
	ID        string `json:"id"`
	Device    string `json:"device"`
	MAC       string `json:"mac"`
	Attached  bool   `json:"attached"`
	NetworkID string `json:"$network"`
}

func (c *Client) GetVIFByID(ctx context.Context, id string) (*VIF, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "VIF", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(VIF{})
	if err != nil {
		return nil, err
	}

	VIF := interf.(VIF)
	return &VIF, nil
}

func (vif *VIF) Delete(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vif.ID,
	}

	return client.rpcConn.Call(ctx, "vif.delete", params, nil)
}

func (vif *VIF) Disconnect(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vif.ID,
	}

	return client.rpcConn.Call(ctx, "vif.disconnect", params, nil)
}
