package chat_ai

import "google.golang.org/grpc"

type Client struct {
	api        ChatServiceClient
	connection *grpc.ClientConn
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		api:        NewChatServiceClient(conn),
		connection: conn,
	}
}

func (c *Client) Api() ChatServiceClient {
	return c.api
}

func (c *Client) Close() error {
	err := c.connection.Close()
	if err != nil {
		return err
	}

	return nil
}
