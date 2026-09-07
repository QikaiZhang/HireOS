# proto 为 source of truth，生成产物禁止手改
.PHONY: generate build test bench run-gateway run-echoworker

generate:
	protoc -I . \
		--go_out=. --go_opt=module=media-gateway \
		--go-grpc_out=. --go-grpc_opt=module=media-gateway \
		proto/media.proto

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/echoworker ./cmd/echoworker

test:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./internal/relay

run-gateway: build
	./bin/gateway

run-echoworker: build
	./bin/echoworker
