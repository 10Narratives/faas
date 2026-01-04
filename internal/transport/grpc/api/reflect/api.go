package reflectapi

import (
	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewRegistration() grpcsrv.ServiceRegistration {
	return func(s *grpc.Server) {
		reflection.Register(s)
	}
}
