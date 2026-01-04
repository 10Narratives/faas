package validator

import (
	v "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	"google.golang.org/grpc"
)

func NewUnaryServerInterceptor(opts ...v.Option) grpc.UnaryServerInterceptor {
	return v.UnaryServerInterceptor(opts...)
}

func NewStreamServerInterceptor(opts ...v.Option) grpc.StreamServerInterceptor {
	return v.StreamServerInterceptor(opts...)
}
