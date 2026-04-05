help:
	@echo 'build ProSe compiler'

fmt:
	echo "go formatting..."
	gofmt -w .

build:
	echo "building the compiler..."
	mkdir -p bin/ && go build -o bin/prose github.com/aumahesh/goprose/cmd

build-sim:
	echo "building the simulator..."
	mkdir -p bin/ && go build -o bin/prose-sim github.com/aumahesh/goprose/cmd/prose-sim

all: fmt build

test:
	echo "running tests"
	go test --cover github.com/aumahesh/goprose/...

test: fmt test

cleanexamples:
	rm -rf _examples/

clean:
	echo "cleaning..."
	rm -rf bin/
