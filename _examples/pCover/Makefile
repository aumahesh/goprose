help:
	@echo 'Build pCover'

fmt:
	echo "go formatting..."
	gofmt -w .

protos:
	echo "Compiling proto files..."
	mkdir -p models && protoc --go_out=./models/ ./proto/state.proto

proto-clean:
	rm -rf models/

build:
	mkdir -p bin/ && go build  --tags=pCover -o bin/pCover aumahesh.com/prose/pCover/cmd

all: fmt protos build

test:
	echo "Running tests"
	go test aumahesh.com/prose/pCover/...

test: fmt protos test

clean: proto-clean
	rm -rf bin/