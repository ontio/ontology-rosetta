GOFMT=gofmt
GC=go build
VERSION := $(shell git describe --always --tags --long)
BUILD_NODE_PAR = -ldflags "-X github.com/ontio/ontology/common/config.Version=$(VERSION)" #-race
DOCKER_IMAGE_NAME="ontology-rosetta"
DOCKER_VERSION="latest"
PWD := $(shell pwd)

ARCH=$(shell uname -m)
SRC_FILES = $(shell git ls-files | grep -e .go$ | grep -v _test.go)

rosetta-node: $(SRC_FILES)
	CGO_ENABLED=1 $(GC)  $(BUILD_NODE_PAR) -o rosetta-node main.go

rosetta-node-cross: rosetta-node-windows rosetta-node-linux rosetta-node-darwin

rosetta-node-windows:
	GOOS=windows GOARCH=amd64 $(GC) $(BUILD_NODE_PAR) -o rosetta-node-windows-amd64.exe main.go

rosetta-node-linux:
	GOOS=linux GOARCH=amd64 $(GC) $(BUILD_NODE_PAR) -o rosetta-node-linux-amd64 main.go

rosetta-node-darwin:
	GOOS=darwin GOARCH=amd64 $(GC) $(BUILD_NODE_PAR) -o rosetta-node-darwin-amd64 main.go

all-cross: rosetta-node-cross

docker:
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_VERSION) .

format:
	$(GOFMT) -w main.go

clean:
	rm -rf *.8 *.o *.out *.6 *exe coverage
	rm -rf rosetta-node rosetta-node-*

