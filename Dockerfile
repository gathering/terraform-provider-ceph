FROM golang:1.25-trixie AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends libcephfs-dev librbd-dev librados-dev && \
    rm -rf /var/lib/apt/lists/*

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

WORKDIR /build

# Download dependencies first so they are cached independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# Integration test stage: adds full Ceph daemons on top of the builder so
# micro-osd.sh can bootstrap a real single-node cluster for integration tests.
FROM builder AS integration-test

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ceph ceph-mds uuid-runtime && \
    rm -rf /var/lib/apt/lists/*

RUN chmod +x /build/scripts/micro-osd.sh /build/scripts/run-integration-tests.sh

ENTRYPOINT ["/build/scripts/run-integration-tests.sh"]

# Export stage: copy just the binary into a scratch image so it can be
# extracted with: docker build --output=. --target=binary .
FROM scratch AS binary
COPY --from=builder /build/terraform-provider-ceph /terraform-provider-ceph
