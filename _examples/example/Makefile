help:
	@echo 'Build example'

fmt:
	echo "go formatting..."
	gofmt -w .

protos:
	echo "Compiling proto files..."
	mkdir -p models && protoc --go_out=./models/ ./proto/state.proto

proto-clean:
	rm -rf models/

build:
	mkdir -p bin/ && go build  --tags=example -o bin/example aumahesh.com/prose/example/cmd

all: fmt protos build

test:
	echo "Running tests"
	go test aumahesh.com/prose/example/...

test: fmt protos test

clean: proto-clean
	rm -rf bin/