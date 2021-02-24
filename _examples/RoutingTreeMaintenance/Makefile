help:
	@echo 'Build RoutingTreeMaintenance'

fmt:
	echo "go formatting..."
	gofmt -w .

protos:
	echo "Compiling proto files..."
	mkdir -p models && protoc --go_out=./models/ ./proto/state.proto

proto-clean:
	rm -rf models/

build:
	mkdir -p bin/ && go build  --tags=RoutingTreeMaintenance -o bin/RoutingTreeMaintenance aumahesh.com/prose/RoutingTreeMaintenance/cmd

all: fmt protos build

test:
	echo "Running tests"
	go test aumahesh.com/prose/RoutingTreeMaintenance/...

test: fmt protos test

clean: proto-clean
	rm -rf bin/