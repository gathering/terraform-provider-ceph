FROM golang:1.25-trixie AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends libcephfs-dev librbd-dev librados-dev unzip && \
    rm -rf /var/lib/apt/lists/*

# Install Terraform so tfplugindocs can use it from PATH instead of downloading
ARG TERRAFORM_VERSION=1.15.2
ARG TARGETARCH
RUN curl -fsSL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${TARGETARCH}.zip" -o /tmp/terraform.zip \
    && unzip /tmp/terraform.zip -d /usr/local/bin \
    && rm /tmp/terraform.zip

RUN curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2

WORKDIR /build

# Download dependencies first so they are cached independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy scripts so integration-test stage has them at build time.
COPY scripts/ scripts/

# Integration test stage: adds full Ceph daemons on top of the builder so
# micro-osd.sh can bootstrap a real single-node cluster for integration tests.
FROM builder AS integration-test

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ceph ceph-mds uuid-runtime && \
    rm -rf /var/lib/apt/lists/*

RUN chmod +x /build/scripts/micro-osd.sh /build/scripts/run-integration-tests.sh

ENTRYPOINT ["/build/scripts/run-integration-tests.sh"]

# Binary stage: build from source and export just the binary.
# Extract with: docker build --output=. --target=binary .
FROM builder AS binary-builder
COPY . .
RUN make build

FROM scratch AS binary
COPY --from=binary-builder /build/terraform-provider-ceph /terraform-provider-ceph
