package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Ratelimiter interface {
	Limit(ip string) (bool, error)
}

func RatelimiterMiddleware(next http.Handler, limiter Ratelimiter, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		pass, err := limiter.Limit(ip)
		if err != nil {
			logger.Error("error occurred while checking rate limit", "ip", ip, "error", err)
			http.Error(w, "rate limit check failed", http.StatusInternalServerError)
			return
		}

		if !pass {
			logger.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RatelimiterUnaryInterceptor(limiter Ratelimiter, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		p, ok := peer.FromContext(ctx)
		if !ok {
			logger.Error("failed to get peer from context")
			return nil, status.Error(codes.Internal, "failed to get peer info")
		}

		ip := p.Addr.String()
		pass, err := limiter.Limit(ip)
		if err != nil {
			logger.Error("error occurred while checking rate limit", "ip", ip, "error", err)
			return nil, status.Error(codes.Internal, "rate limit check failed")
		}

		if !pass {
			logger.Warn("rate limit exceeded", "ip", ip)
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
