help:
	@echo 'Build dijkstra_gcl_do'

fmt:
	echo "go formatting..."
	gofmt -w .

protos:
	echo "Compiling proto files..."
	mkdir -p models && protoc --go_out=./models/ ./proto/state.proto

proto-clean:
	rm -rf models/

build:
	mkdir -p bin/ && go build  --tags=dijkstra_gcl_do -o bin/dijkstra_gcl_do aumahesh.com/prose/dijkstra_gcl_do/cmd

all: fmt protos build

test:
	echo "Running tests"
	go test aumahesh.com/prose/dijkstra_gcl_do/...

test: fmt protos test

clean: proto-clean
	rm -rf bin/