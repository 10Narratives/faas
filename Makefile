
generate:
	buf generate

build: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-agent  ./cmd/faas-agent/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-gateway ./cmd/faas-gateway
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-cli ./cmd/faas-cli
	docker compose build --parallel

start:
	docker compose up -d

stop:
	docker compose stop

ps:
	docker ps --format "table {{.ID}}\t{{.Names}}\t{{.Ports}}\t{{.Status}}"

clean:
	docker compose down -v --remove-orphans
