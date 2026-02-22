start:
	go run cmd/service/server.go
debug:
	DEBUG=true go run cmd/service/server.go
mod:
	go mod tidy
	go mod vendor
up:
	docker compose -f docker-compose.yml up --build -d
clean:
	docker compose -f docker-compose.yml down -v --rmi local
full:
	make clean
	make up