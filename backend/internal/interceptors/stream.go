package interceptors

import (
	"google.golang.org/grpc"

	"github.com/zukigit/chat/backend/internal/lib"
)

// StreamRecoveryInterceptor recovers from panics in stream handlers
func StreamRecoveryInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	defer func() {
		if r := recover(); r != nil {
			lib.ErrorLog.Printf("Panic recovered in stream %s: %v", info.FullMethod, r)
		}
	}()

	return handler(srv, ss)
}
