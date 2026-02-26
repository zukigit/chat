package backend

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/zukigit/chat/backend/internal/interceptors"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/internal/services"
	"github.com/zukigit/chat/proto/auth"
	"google.golang.org/grpc"
)

func main() {
	dbHost := lib.Getenv("DB_HOST", "localhost")
	dbPasswd := lib.Getenv("DB_PASSWORD", "")
	dbSslMode := lib.Getenv("DB_SSLMODE", "disable")
	dsn := fmt.Sprintf("postgres://chat:%s@%s/chat?sslmode=%s", dbPasswd, dbHost, dbSslMode)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to database: %v", err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryRecoveryInterceptor),
		grpc.StreamInterceptor(interceptors.StreamRecoveryInterceptor),
	)

	auth.RegisterAuthServer(srv, services.NewAuthServer(sqlDB))
}
