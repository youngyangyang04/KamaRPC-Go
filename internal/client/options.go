package client

import (
	"kamaRPC/internal/codec"
	"kamaRPC/internal/loadbalance"
	"time"
)

type ClientOption func(*Client) error

func WithClientCodec(t codec.Type) ClientOption {
	return func(c *Client) error {
		cc, err := codec.New(t)
		if err != nil {
			return err
		}
		c.codec = cc
		return nil
	}
}

func WithClientTimeout(d time.Duration) ClientOption {
	return func(c *Client) error {
		c.timeout = d
		return nil
	}
}

func WithClientLoadBalancer(lb loadbalance.LoadBalancer) ClientOption {
	return func(c *Client) error {
		c.lb = lb
		return nil
	}
}
