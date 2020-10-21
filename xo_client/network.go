package xo_client

import (
	"context"
)

type Network struct {
	ID          string `json:"id"`
	Name        string `json:"name_label"`
	Description string `json:"name_description"`
	Pool        string `json:"$pool"`
}

func (c *Client) GetNetworkByID(ctx context.Context, id string) (*Network, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "network", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(Network{})
	if err != nil {
		return nil, err
	}

	network := interf.(Network)
	return &network, nil
}
