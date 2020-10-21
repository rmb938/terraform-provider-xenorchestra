package xo_client

import (
	"context"
	"net/url"
	"path"

	gws "github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/sourcegraph/jsonrpc2/websocket"
)

type Client struct {
	rpcConn *jsonrpc2.Conn
}

type ObjectQuery map[string]string

func NewClient(u *url.URL) (*Client, error) {

	dialer := gws.Dialer{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}

	u.Path = path.Join(u.Path, "api") + "/"

	ws, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	objStream := websocket.NewObjectStream(ws)

	rpcConn := jsonrpc2.NewConn(context.Background(), objStream, &noopHandler{})

	return &Client{
		rpcConn: rpcConn,
	}, nil
}

func (c *Client) Close() error {
	return c.rpcConn.Close()
}

func (c *Client) SignIn(ctx context.Context, username, password string) error {
	params := map[string]interface{}{
		"email":    username,
		"password": password,
	}
	var reply map[string]interface{}
	return c.rpcConn.Call(ctx, "session.signInWithPassword", params, &reply)
}

type noopHandler struct{}

func (*noopHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {}
