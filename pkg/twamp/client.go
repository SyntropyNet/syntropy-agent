package twamp

import (
	"fmt"
	"net"
)

type SessionConfig struct {
	Port    int
	Padding int
	Timeout int
	TOS     int
}

type Client struct {
	host     string
	testPort uint16
	conn     net.Conn

	test   *twampTest
	config SessionConfig
}

func NewClient(hostname string, config SessionConfig) (*Client, error) {
	// connect to remote host
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", hostname, TwampControlPort))
	if err != nil {
		return nil, err
	}

	// create a new Connection
	client := &Client{
		host:   hostname,
		conn:   conn,
		config: config,
	}

	// check for greeting message from TWAMP server
	err = recvServerGreeting(client.GetConnection())
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = sendClientSetupResponse(client.GetConnection())
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	err = recvServerStartMessage(client.GetConnection())
	if err != nil {
		return nil, err
	}

	err = client.createSession()
	if err != nil {
		return nil, err
	}

	err = client.createTest()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) PaddingSize() uint {
	return uint(c.config.Padding)
}

func (c *Client) GetHost() string {
	return c.host
}

func (c *Client) GetConnection() net.Conn {
	return c.conn
}

func (c *Client) Close() error {
	c.stopSession()
	return c.GetConnection().Close()
}

func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
