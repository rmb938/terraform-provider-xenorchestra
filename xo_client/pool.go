package xo_client

import (
	"context"
)

type Pool struct {
	ID          string `json:"id"`
	Name        string `json:"name_label"`
	Description string `json:"name_description"`
}

func (c *Client) GetPoolByName(ctx context.Context, name string) (*Pool, error) {
	query := ObjectQuery{
		"name_label": name,
	}

	objs, err := c.GetObjectsOfType(ctx, "pool", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(Pool{})
	if err != nil {
		return nil, err
	}

	pool := interf.(Pool)
	return &pool, nil
}
