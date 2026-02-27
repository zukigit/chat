package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/zukigit/chat/internal/lib"
)

// UnaryRecoveryInterceptor recovers from panics in RPC handlers
func UnaryRecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			lib.ErrorLog.Printf("Panic recovered in %s: %v", info.FullMethod, r)
		}
	}()

	return handler(ctx, req)
}
