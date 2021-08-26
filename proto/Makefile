all:
	mkdir -p v1
	protoc \
		-I/usr/local/include -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
		-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0 \
		--go_out=./v1 \
		--go_opt=paths=source_relative \
		--go-grpc_out=./v1 \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=logtostderr=true:./v1 \
		--grpc-gateway_opt=paths=source_relative \
		--swagger_out=allow_merge=true,merge_file_name=api:./v1 \
		./api.v1.proto
