stop:
	docker compose down

clean:
	docker compose down -v --remove-orphans

build-faas-agent:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-agent  ./cmd/faas-agent/
	
build-faas-gateway:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas-gateway ./cmd/faas-gateway

cli:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/faas ./cmd/faas-cli
	sudo install -m 0755 build/faas /usr/local/bin/faas

build: build-faas-agent build-faas-gateway

start: build
	docker compose up --build --detach

generate:
	buf generate

mocks:
	mockery