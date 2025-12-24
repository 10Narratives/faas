stop:
	docker compose down

clean:
	docker compose down -v --remove-orphans

build-faas-agent:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-agent  ./cmd/faas-agent/

build: build-faas-agent

start: build
	docker compose up --build --detach