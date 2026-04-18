BIN            = terraform-provider-ceph
GOFMT_FILES   ?= $$(find . -name '*.go')
GO_ARGS       ?=
DOCKER_IMAGE  ?= terraform-provider-ceph-build
TEST_ARGS     ?=

all: build

$(BIN): ceph main.go go.mod go.sum
	go build $(GO_ARGS) -o $@

build: $(BIN)

debug: GO_ARGS += -gcflags=all="-N -l"
debug: $(BIN)

fmt:
	go generate ./...
	gofmt -s -w $(GOFMT_FILES)

# Run unit tests locally (requires librados-dev)
test:
	go test $(TEST_ARGS) ./ceph/

# Run unit tests with verbose output locally
test-verbose:
	go test -v $(TEST_ARGS) ./ceph/

# Build the Docker image used for building and testing
docker-image:
	docker build --target=builder -t $(DOCKER_IMAGE) .

# Run unit tests inside Docker (no local Ceph libraries required)
docker-test:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		go test $(TEST_ARGS) ./ceph/

# Run unit tests inside Docker with verbose output
docker-test-verbose:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		go test -v $(TEST_ARGS) ./ceph/

docker-fmt:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		gofmt -s -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...

docker-lint:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		golangci-lint run ./...

docker-vet:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		go vet ./...

docker-generate:
	docker run --rm -v "$$(pwd):/build" -w /build $(DOCKER_IMAGE) \
		go generate ./...

# Build the integration test Docker image (includes full Ceph daemons + micro-osd)
docker-integration-image:
	docker build --target=integration-test -t $(DOCKER_IMAGE)-integration .

# Run integration tests against a real micro-osd Ceph cluster inside Docker
docker-integration-test:
	docker run --rm --privileged $(DOCKER_IMAGE)-integration

.PHONY: all build debug fmt lint test test-verbose docker-image docker-test docker-test-verbose docker-fmt docker-lint docker-vet docker-generate docker-integration-image docker-integration-test
