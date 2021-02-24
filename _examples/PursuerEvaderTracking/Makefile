help:
	@echo 'Build PursuerEvaderTracking'

fmt:
	echo "go formatting..."
	gofmt -w .

protos:
	echo "Compiling proto files..."
	mkdir -p models && protoc --go_out=./models/ ./proto/state.proto

proto-clean:
	rm -rf models/

build:
	mkdir -p bin/ && go build  --tags=PursuerEvaderTracking -o bin/PursuerEvaderTracking aumahesh.com/prose/PursuerEvaderTracking/cmd

all: fmt protos build

test:
	echo "Running tests"
	go test aumahesh.com/prose/PursuerEvaderTracking/...

test: fmt protos test

clean: proto-clean
	rm -rf bin/