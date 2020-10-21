package xo_client

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type Objects map[string]interface{}

func (c *Client) GetObjectsOfType(ctx context.Context, apiType string, query ObjectQuery) (*Objects, error) {
	filter := map[string]string{
		"type": apiType,
	}

	for k, v := range query {
		filter[k] = v
	}

	params := map[string]interface{}{
		"filter": filter,
	}

	objs := &Objects{}
	err := c.rpcConn.Call(ctx, "xo.getAllObjects", params, objs)
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func (o *Objects) ConvertToSlice(obj interface{}) (interface{}, error) {
	t := reflect.TypeOf(obj)
	objs := reflect.MakeSlice(reflect.SliceOf(t), 0, 0)

	for _, obj := range *o {
		objBytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}

		item := reflect.New(t)
		err = json.Unmarshal(objBytes, item.Interface())
		if err != nil {
			return nil, err
		}

		objs = reflect.Append(objs, item.Elem())
	}

	return objs.Interface(), nil
}

func (o *Objects) ConvertToSingle(obj interface{}) (interface{}, error) {
	interf, err := o.ConvertToSlice(obj)
	if err != nil {
		return nil, err
	}

	interfSlice := reflect.ValueOf(interf)

	if interfSlice.Len() == 0 {
		return nil, NotFoundError
	}

	if interfSlice.Len() > 1 {
		return nil, fmt.Errorf("found multiple")
	}

	return interfSlice.Index(0).Interface(), nil
}
