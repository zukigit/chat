package backend

import (
	"github.com/zukigit/chat/backend/internal/interceptors"
	"github.com/zukigit/chat/backend/internal/services"
	"github.com/zukigit/chat/proto/auth"
	"google.golang.org/grpc"
)

func main() {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryRecoveryInterceptor),
		grpc.StreamInterceptor(interceptors.StreamRecoveryInterceptor),
	)

	auth.RegisterAuthServer(srv, services.NewAuthServer())
}
