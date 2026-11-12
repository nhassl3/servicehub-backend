package health

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Pinger — всё, что умеет проверять свою доступность (например, *pgxpool.Pool).
type Pinger interface {
	Ping(ctx context.Context) error
}

const (
	// ServiceReadiness — "готов принимать трафик" (зависит от БД).
	ServiceReadiness = "readiness"
	// ServiceLiveness — "процесс жив" (БД не проверяем, иначе k8s будет зря рестартить под).
	ServiceLiveness = "liveness"
)

// Register вешает Health-сервис на gRPC-сервер и запускает фоновую проверку зависимостей.
// Возвращает *health.Server, чтобы в graceful shutdown вызвать Shutdown().
func Register(ctx context.Context, srv *grpc.Server, db Pinger, interval time.Duration) *health.Server {
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)

	// "" — общий статус сервера (то, что спрашивают по умолчанию).
	hs.SetServingStatus(ServiceLiveness, healthpb.HealthCheckResponse_SERVING)
	hs.SetServingStatus(ServiceReadiness, healthpb.HealthCheckResponse_NOT_SERVING)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	check := func() {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		status := healthpb.HealthCheckResponse_SERVING
		if err := db.Ping(pctx); err != nil {
			status = healthpb.HealthCheckResponse_NOT_SERVING
		}
		hs.SetServingStatus(ServiceReadiness, status)
		hs.SetServingStatus("", status)
	}

	check() // первая проверка сразу, чтобы не ждать interval
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				check()
			}
		}
	}()

	return hs
}
