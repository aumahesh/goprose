help:
	@echo 'Build ProSe Compiler'

fmt:
	echo "go formatting..."
	gofmt -w .

build:
	mkdir -p bin/ && go build  -o bin/prose github.com/aumahesh/goprose/cmd

all: fmt build

test:
	echo "Running tests"
	go test  github.com/aumahesh/goprose/...

test: fmt test

clean:
	rm -rf bin/
	rm -rf internal/templates/tmp/