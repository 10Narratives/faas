package healthapi

import (
	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func NewRegistration() grpcsrv.ServiceRegistration {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("faas.functions.v1.FunctionService", grpc_health_v1.HealthCheckResponse_SERVING)

	return func(s *grpc.Server) {
		grpc_health_v1.RegisterHealthServer(s, healthServer)
	}
}
