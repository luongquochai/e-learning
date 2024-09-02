clean:
	rm pb/*

gen:
	protoc --proto_path=proto proto/*.proto --go-grpc_opt=require_unimplemented_servers=false --go_out=:pb --go-grpc_out=:pb

server:
	go run cmd/server/main.go -port 8080

client:
	go run cmd/client/main.go -address 0.0.0.0:8080

test:
	go test -cover -race ./...