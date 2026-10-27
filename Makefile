.PHONY: init run build test up down logs

# One-time: fetch dependencies and generate go.sum
init:
	go get github.com/gin-gonic/gin@latest go.mongodb.org/mongo-driver/v2@latest
	go mod tidy

# Run locally against a Mongo on localhost:27017 (e.g. `docker compose up -d mongo` + MONGO_URI)
run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -trimpath -o bin/api ./cmd/api

test:
	go test ./...

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api
