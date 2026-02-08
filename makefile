start:
	go run cmd/main.go
debug:
	DEBUG=true go run cmd/main.go
mod:
	go mod tidy
	go mod vendor