package grpcsrv

import (
	"context"
	"errors"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

type Component struct {
	address string
	server  *grpc.Server
}

func NewComponent(address string, opts ...ComponentOption) *Component {
	options := defaultComponentOptions()
	for _, opt := range opts {
		opt(options)
	}

	server := grpc.NewServer(options.serverOptions...)
	for _, reg := range options.serviceRegs {
		reg(server)
	}

	return &Component{
		address: address,
		server:  server,
	}
}

func (c *Component) Startup(ctx context.Context) error {
	lis, err := net.Listen("tcp", c.address)
	if err != nil {
		return fmt.Errorf("cannot listen address %s: %w", c.address, err)
	}

	channel := make(chan error)
	go func() {
		defer close(channel)
		select {
		case channel <- c.server.Serve(lis):
		case <-ctx.Done():
		}
	}()

	select {
	case err := <-channel:
		return fmt.Errorf("error while serve %s: %w", c.address, err)
	case <-ctx.Done():
		return nil
	}
}

func (c *Component) Shutdown(ctx context.Context) error {
	channel := make(chan struct{})
	go func() {
		c.server.GracefulStop()
		close(channel)
	}()

	select {
	case <-channel:
		return nil
	case <-ctx.Done():
		c.server.Stop()
		return errors.New("shutdown context exceeded")
	}
}
