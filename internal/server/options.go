package server

import "kamaRPC/internal/codec"

type HandleOption func(*Handler) error

func WithHandlerCodec(t codec.Type) HandleOption {
	return func(c *Handler) error {
		cc, err := codec.New(t)
		if err != nil {
			return err
		}
		c.codec = cc
		return nil
	}
}

type ServerOption func(*Server) error

func WithServerCodec(t codec.Type) ServerOption {
	return func(c *Server) error {
		cc, err := codec.New(t)
		if err != nil {
			return err
		}
		c.codec = cc
		return nil
	}
}
