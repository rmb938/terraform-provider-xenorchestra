package xo_client

import (
	"context"
)

type StorageRepository struct {
	ID          string `json:"id"`
	Name        string `json:"name_label"`
	Description string `json:"name_description"`
	Type        string `json:"SR_type"`
	Pool        string `json:"$pool"`
}

func (c *Client) GetStorageRepositoryByName(ctx context.Context, poolID, name string) (*StorageRepository, error) {
	query := ObjectQuery{
		"name_label": name,
		"$pool":      poolID,
	}

	objs, err := c.GetObjectsOfType(ctx, "SR", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(StorageRepository{})
	if err != nil {
		return nil, err
	}

	storageRepository := interf.(StorageRepository)
	return &storageRepository, nil
}

func (c *Client) GetStorageRepositoryByID(ctx context.Context, id string) (*StorageRepository, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "SR", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(StorageRepository{})
	if err != nil {
		return nil, err
	}

	storageRepository := interf.(StorageRepository)
	return &storageRepository, nil
}
