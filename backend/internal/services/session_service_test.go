package services_test

import (
"context"
"testing"

"github.com/google/uuid"
"github.com/zukigit/chat/backend/internal/db"
"github.com/zukigit/chat/backend/internal/services"
pb "github.com/zukigit/chat/backend/proto/session"
"google.golang.org/grpc/codes"
)

func TestValidateSession(t *testing.T) {
sqlDB := setupTestDB(t)
sessionServer := services.NewSessionServer(sqlDB)

ids := createTestUsers(t, sqlDB, "alice")
loginID := uuid.New()

// Insert a session row directly so ValidateSession can find it.
if err := db.New(sqlDB).CreateSession(context.Background(), db.CreateSessionParams{
UserUserid: ids["alice"],
LoginID:    loginID,
}); err != nil {
t.Fatalf("setup CreateSession: %v", err)
}

cases := []struct {
name    string
loginID string
wantErr codes.Code
}{
{"valid login_id", loginID.String(), codes.OK},
{"unknown login_id", uuid.NewString(), codes.Unauthenticated},
{"invalid uuid", "not-a-uuid", codes.InvalidArgument},
{"empty login_id", "", codes.InvalidArgument},
}

for _, tc := range cases {
t.Run(tc.name, func(t *testing.T) {
resp, err := sessionServer.ValidateSession(context.Background(), &pb.ValidateSessionRequest{
LoginId: tc.loginID,
})
if got := grpcCode(err); got != tc.wantErr {
t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
}
if tc.wantErr == codes.OK && resp.GetUserId() != ids["alice"].String() {
t.Errorf("user_id: got %q, want %q", resp.GetUserId(), ids["alice"].String())
}
})
}
}
