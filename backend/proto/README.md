## How to generate codes
```bash
protoc --go_out=. --go-grpc_out=. ./auth/auth.proto ./friendship/friendship.proto
```