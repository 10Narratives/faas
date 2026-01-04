package grpcsrv

import "google.golang.org/grpc"

type ServiceRegistration func(s *grpc.Server)

type componentOptions struct {
	serverOptions []grpc.ServerOption
	serviceRegs   []ServiceRegistration
}

type ComponentOption func(co *componentOptions)

func defaultComponentOptions() *componentOptions {
	return &componentOptions{}
}

func WithServerOptions(options ...grpc.ServerOption) ComponentOption {
	return func(co *componentOptions) {
		co.serverOptions = append(co.serverOptions, options...)
	}
}

func WithServiceRegistration(regs ...ServiceRegistration) ComponentOption {
	return func(co *componentOptions) {
		co.serviceRegs = append(co.serviceRegs, regs...)
	}
}
