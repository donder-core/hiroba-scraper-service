start:
	go run cmd/service/server.go
debug:
	DEBUG=true go run cmd/service/server.go
mod:
	go mod tidy
	go mod vendor