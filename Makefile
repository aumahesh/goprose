help:
	@echo 'build ProSe compiler'

fmt:
	echo "go formatting..."
	gofmt -w .

build:
	echo "building the compiler..."
	mkdir -p bin/ && go build  -o bin/prose github.com/aumahesh/goprose/cmd

all: fmt build

test:
	echo "running tests"
	go test  github.com/aumahesh/goprose/...

test: fmt test

clean:
	echo "cleaning..."
	rm -rf bin/
	rm -rf _generatedModules
	rm -rf _examples