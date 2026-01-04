package recovery

import (
	r "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
)

func NewUnaryServerInterceptor(opts ...r.Option) grpc.UnaryServerInterceptor {
	return r.UnaryServerInterceptor(opts...)
}

func NewStreamServerInterceptor(opts ...r.Option) grpc.StreamServerInterceptor {
	return r.StreamServerInterceptor(opts...)
}
